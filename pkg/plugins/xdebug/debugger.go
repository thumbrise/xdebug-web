package xdebug

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/thumbrise/xdebug-web/pkg/plugins"
)

const maxMsgBuf = 64

var (
	errMissingBreakpointArgs = errors.New("breakpoint command missing required args")
)

type state struct {
	status       plugins.Status
	currentFile  string
	currentLine  int
	stack        []plugins.Frame
	locals       []plugins.Variable
	breakpoints  []plugins.Breakpoint
}

func newState() *state {
	return &state{
		status:      plugins.StatusStarting,
		stack:       make([]plugins.Frame, 0),
		locals:      make([]plugins.Variable, 0),
		breakpoints: make([]plugins.Breakpoint, 0),
	}
}

type Debugger struct {
	conn        *conn
	logger      *slog.Logger
	remoteRoot  string
	root        string

	txID        int
	cmds        chan plugins.Command
	stateCh     chan *plugins.State
	breakpoints map[string]plugins.Breakpoint // file:line → bp
	internal    *state

	initFileURI string
}

func NewDebugger(conn *conn, logger *slog.Logger, root, remoteRoot string) *Debugger {
	return &Debugger{
		conn:        conn,
		logger:      logger.With("subsystem", "xdebug"),
		cmds:        make(chan plugins.Command, maxMsgBuf),
		stateCh:     make(chan *plugins.State, maxMsgBuf),
		breakpoints: make(map[string]plugins.Breakpoint),
		internal:    newState(),
		root:        root,
		remoteRoot:  remoteRoot,
	}
}

func (d *Debugger) Info() plugins.Info {
	return plugins.Info{Name: "xdebug", Language: "php"}
}

func (d *Debugger) Commands() chan<- plugins.Command {
	return d.cmds
}

func (d *Debugger) State() <-chan *plugins.State {
	return d.stateCh
}

func (d *Debugger) Close() error {
	return d.conn.Close()
}

// emit sends a snapshot of internal state to the state channel.
func (d *Debugger) emit() {
	s := &plugins.State{
		Status:       d.internal.status,
		CurrentFile:  d.internal.currentFile,
		CurrentLine:  d.internal.currentLine,
		Stack:        d.internal.stack,
		Locals:       d.internal.locals,
		Globals:      make([]plugins.Variable, 0),
		Breakpoints:  d.internal.breakpoints,
		Capabilities: plugins.Capabilities{HasStack: true, HasLocals: true},
	}

	select {
	case d.stateCh <- s:
	default:
	}
}

func (d *Debugger) Run(ctx context.Context) error {
	defer close(d.stateCh)

	initData, err := d.conn.ReadMessage(ctx)
	if err != nil {
		return fmt.Errorf("read init: %w", err)
	}

	initInfo, err := parseInit(initData)
	if err != nil {
		return fmt.Errorf("parse init: %w", err)
	}

	relativePath := toRelative(initInfo.FileURI, d.remoteRoot)
	d.internal.currentFile = relativePath
	d.initFileURI = initInfo.FileURI
	d.emit()

	d.logger.InfoContext(ctx, "xdebug connected",
		"language", initInfo.Language,
		"file", initInfo.FileURI,
		"relative", relativePath,
		"ideKey", initInfo.IdeKey,
	)

	return d.commandLoop(ctx)
}

func (d *Debugger) commandLoop(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case cmd := <-d.cmds:
			result, err := d.executeCommand(ctx, cmd)
			if err != nil {
				d.logger.ErrorContext(ctx, "command", "cmd", cmd.Type, "error", err)

				if cmd.Reply != nil {
					cmd.Reply <- err
				}

				continue
			}

			if cmd.Reply != nil {
				cmd.Reply <- result
			}

			if done := d.afterCommand(ctx, result); done {
				return nil
			}

		default:
			if done, err := d.processBreak(ctx); err != nil {
				return err
			} else if done {
				return nil
			}
		}
	}
}

func (d *Debugger) processBreak(ctx context.Context) (bool, error) {
	for {
		result, err := d.doStep(ctx, "run")
		if err != nil {
			return false, fmt.Errorf("run: %w", err)
		}

		d.internal.status = result.Status
		d.internal.currentFile = toRelative(result.Filename, d.remoteRoot)
		d.internal.currentLine = result.Lineno

		switch result.Status {
		case plugins.StatusBreak:
			// hasPrefix("/") = outside project → keep running
			if strings.HasPrefix(d.internal.currentFile, "/") {
				d.emit()
				continue
			}
			return false, d.handleBreak(ctx)

		case plugins.StatusStopping, plugins.StatusStopped:
			d.emit()
			return true, nil

		case plugins.StatusRunning:
			d.emit()
			return false, nil
		}
	}
}

func (d *Debugger) handleBreak(ctx context.Context) error {
	if err := d.refreshStack(ctx); err != nil {
		d.logger.WarnContext(ctx, "refresh stack", "error", err)
	}

	if err := d.refreshLocals(ctx); err != nil {
		d.logger.WarnContext(ctx, "refresh locals", "error", err)
	}

	d.emit()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case cmd := <-d.cmds:
			result, err := d.executeCommand(ctx, cmd)
			if err != nil {
				d.logger.ErrorContext(ctx, "command", "cmd", cmd.Type, "error", err)

				if cmd.Reply != nil {
					cmd.Reply <- err
				}

				continue
			}

			if cmd.Reply != nil {
				cmd.Reply <- result
			}

			if done := d.afterCommand(ctx, result); done {
				return nil
			}
		}
	}
}

func (d *Debugger) afterCommand(ctx context.Context, result *stepResult) bool {
	d.internal.status = result.Status
	d.internal.currentFile = toRelative(result.Filename, d.remoteRoot)
	d.internal.currentLine = result.Lineno

	switch result.Status {
	case plugins.StatusBreak:
		if err := d.refreshStack(ctx); err != nil {
			d.logger.WarnContext(ctx, "refresh stack", "error", err)
		}

		if err := d.refreshLocals(ctx); err != nil {
			d.logger.WarnContext(ctx, "refresh locals", "error", err)
		}

		d.emit()

	case plugins.StatusStopping, plugins.StatusStopped:
		d.emit()
		return true

	case plugins.StatusRunning:
		d.emit()

	case plugins.StatusStarting:
	}

	return false
}

func (d *Debugger) executeCommand(ctx context.Context, cmd plugins.Command) (*stepResult, error) {
	switch cmd.Type {
	case plugins.CmdStepInto, plugins.CmdStepOver, plugins.CmdStepOut, plugins.CmdRun:
		return d.doStep(ctx, string(cmd.Type))

	case plugins.CmdStop:
		_ = d.conn.Close()

		return &stepResult{Status: plugins.StatusStopped}, nil

	case plugins.CmdBreakpointSet:
		if err := d.handleBreakpointSet(ctx, cmd); err != nil {
			return nil, err
		}

		return &stepResult{Status: d.internal.status}, nil

	case plugins.CmdBreakpointRemove:
		if err := d.handleBreakpointRemove(ctx, cmd); err != nil {
			return nil, err
		}

		return &stepResult{Status: d.internal.status}, nil

	default:
		return nil, fmt.Errorf("%s: %w", cmd.Type, plugins.ErrUnknownCommand)
	}
}

func (d *Debugger) doStep(ctx context.Context, stepCmd string) (*stepResult, error) {
	d.txID++

	cmd := fmt.Sprintf("%s -i %d\x00", stepCmd, d.txID)

	if err := d.conn.SendMessage(ctx, cmd); err != nil {
		return nil, fmt.Errorf("send %s: %w", stepCmd, err)
	}

	data, err := d.conn.ReadMessage(ctx)
	if err != nil {
		return nil, fmt.Errorf("read %s response: %w", stepCmd, err)
	}

	return parseStepResponse(data)
}

func (d *Debugger) refreshStack(ctx context.Context) error {
	d.txID++

	cmd := formatStackGetCmd(d.txID)

	if err := d.conn.SendMessage(ctx, cmd); err != nil {
		return fmt.Errorf("send stack_get: %w", err)
	}

	data, err := d.conn.ReadMessage(ctx)
	if err != nil {
		return fmt.Errorf("read stack_get: %w", err)
	}

	frames, err := parseStackGetResponse(data)
	if err != nil {
		return err
	}

	for i := range frames {
		frames[i].Filename = toRelative(frames[i].Filename, d.remoteRoot)
	}

	d.internal.stack = frames

	return nil
}

func (d *Debugger) refreshLocals(ctx context.Context) error {
	d.txID++

	cmd := formatContextGetCmd(d.txID, 0)

	if err := d.conn.SendMessage(ctx, cmd); err != nil {
		return fmt.Errorf("send context_get: %w", err)
	}

	data, err := d.conn.ReadMessage(ctx)
	if err != nil {
		return fmt.Errorf("read context_get: %w", err)
	}

	vars, err := parseContextGetResponse(data)
	if err != nil {
		return err
	}

	d.internal.locals = vars

	return nil
}

func (d *Debugger) handleBreakpointSet(ctx context.Context, cmd plugins.Command) error {
	file := cmd.Args["file"]

	lineStr := cmd.Args["line"]

	if file == "" || lineStr == "" {
		return errMissingBreakpointArgs
	}

	line, err := strconv.Atoi(lineStr)
	if err != nil {
		return fmt.Errorf("breakpoint_set invalid line: %w", err)
	}

	d.txID++

	uri := toURI(file, d.root, d.remoteRoot)
	bpCmd := formatBreakpointSetCmd(d.txID, uri, line)

	if err := d.conn.SendMessage(ctx, bpCmd); err != nil {
		return fmt.Errorf("send breakpoint_set: %w", err)
	}

	data, err := d.conn.ReadMessage(ctx)
	if err != nil {
		return fmt.Errorf("read breakpoint_set: %w", err)
	}

	bpID, err := parseBreakpointSetResponse(data)
	if err != nil {
		return fmt.Errorf("parse breakpoint_set: %w", err)
	}

	key := file + ":" + lineStr
	d.breakpoints[key] = plugins.Breakpoint{ID: bpID, File: file, Line: line}
	d.syncBreakpoints()

	return nil
}

func (d *Debugger) handleBreakpointRemove(ctx context.Context, cmd plugins.Command) error {
	id := cmd.Args["id"]

	if id == "" {
		return errMissingBreakpointArgs
	}

	d.txID++

	rmCmd := formatBreakpointRemoveCmd(d.txID, id)

	if err := d.conn.SendMessage(ctx, rmCmd); err != nil {
		return fmt.Errorf("send breakpoint_remove: %w", err)
	}

	data, err := d.conn.ReadMessage(ctx)
	if err != nil {
		return fmt.Errorf("read breakpoint_remove: %w", err)
	}

	if _, err := parseStepResponse(data); err != nil {
		return fmt.Errorf("parse breakpoint_remove: %w", err)
	}

	for k, bp := range d.breakpoints {
		if bp.ID == id {
			delete(d.breakpoints, k)
			break
		}
	}

	d.syncBreakpoints()

	return nil
}

func (d *Debugger) syncBreakpoints() {
	bps := make([]plugins.Breakpoint, 0, len(d.breakpoints))

	for _, bp := range d.breakpoints {
		bps = append(bps, bp)
	}

	d.internal.breakpoints = bps
}
