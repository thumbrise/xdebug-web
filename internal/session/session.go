package session

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/thumbrise/xdebug-web/internal/dbgp"
	"github.com/thumbrise/xdebug-web/internal/pathmap"
)

const maxMsgBuf = 64

var (
	ErrUnknownCmdType        = errors.New("unknown command type")
	ErrMissingBreakpointArgs = errors.New("breakpoint command missing required args")
)

type Session struct {
	conn        *dbgp.Conn
	state       *State
	cmds        chan dbgp.Command
	subs        []chan *State
	done        chan struct{}
	txID        int
	logger      *slog.Logger
	remoteRoot  string
	root        string
	breakpoints map[string]dbgp.Breakpoint // file:line → bp
}

func NewSession(conn *dbgp.Conn, logger *slog.Logger, remoteRoot, root string) *Session {
	return &Session{
		conn:        conn,
		logger:      logger.With("subsystem", "session"),
		state:       NewState(),
		txID:        0,
		cmds:        make(chan dbgp.Command, maxMsgBuf),
		done:        make(chan struct{}),
		remoteRoot:  remoteRoot,
		root:        root,
		breakpoints: make(map[string]dbgp.Breakpoint),
	}
}

func (s *Session) Run(ctx context.Context) error {
	initData, err := s.conn.ReadMessage(ctx)
	if err != nil {
		return fmt.Errorf("read init: %w", err)
	}

	initInfo, err := dbgp.ParseInit(initData)
	if err != nil {
		return fmt.Errorf("parse init: %w", err)
	}

	relativePath := pathmap.ToRelative(initInfo.FileURI, s.remoteRoot)

	s.state.SetInit(initInfo, relativePath)
	s.broadcast()

	s.logger.InfoContext(ctx, "xdebug connected",
		"language", initInfo.Language,
		"file", initInfo.FileURI,
		"relative", relativePath,
		"ideKey", initInfo.IdeKey,
	)

	return s.commandLoop(ctx)
}

func (s *Session) commandLoop(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case cmd := <-s.cmds:
			result, err := s.executeCommand(ctx, cmd)
			if err != nil {
				s.logger.ErrorContext(ctx, "command", "cmd", cmd.Type, "error", err)
				continue
			}
			if done := s.afterCommand(ctx, result); done {
				return nil
			}

		default:
			if done, err := s.processBreak(ctx); err != nil {
				return err
			} else if done {
				return nil
			}
		}
	}
}

func (s *Session) processBreak(ctx context.Context) (bool, error) {
	result, err := s.doStep(ctx, "step_into")
	if err != nil {
		return false, fmt.Errorf("initial step: %w", err)
	}

	s.state.SetStepStatus(result.Status, s.toRelative(result.Filename), result.Lineno)

	switch result.Status {
	case dbgp.StatusBreak:
		return false, s.handleBreak(ctx)

	case dbgp.StatusStopping, dbgp.StatusStopped:
		s.state.SetStopped()
		s.broadcast()

		return true, nil

	case dbgp.StatusRunning:
		s.state.SetRunning()

	case dbgp.StatusStarting:
	}

	return false, nil
}

func (s *Session) handleBreak(ctx context.Context) error {
	if err := s.refreshStack(ctx); err != nil {
		s.logger.WarnContext(ctx, "refresh stack", "error", err)
	}

	if err := s.refreshLocals(ctx); err != nil {
		s.logger.WarnContext(ctx, "refresh locals", "error", err)
	}

	s.broadcast()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case cmd := <-s.cmds:
			result, err := s.executeCommand(ctx, cmd)
			if err != nil {
				s.logger.ErrorContext(ctx, "command", "cmd", cmd.Type, "error", err)

				if cmd.Reply != nil {
					cmd.Reply <- err
				}

				continue
			}

			if cmd.Reply != nil {
				cmd.Reply <- result
			}

			if done := s.afterCommand(ctx, result); done {
				return nil
			}
		}
	}
}

func (s *Session) afterCommand(ctx context.Context, result *dbgp.StepResult) bool {
	s.state.SetStepStatus(result.Status, s.toRelative(result.Filename), result.Lineno)

	switch result.Status {
	case dbgp.StatusBreak:
		if err := s.refreshStack(ctx); err != nil {
			s.logger.WarnContext(ctx, "refresh stack", "error", err)
		}

		if err := s.refreshLocals(ctx); err != nil {
			s.logger.WarnContext(ctx, "refresh locals", "error", err)
		}

		s.broadcast()

	case dbgp.StatusStopping, dbgp.StatusStopped:
		s.state.SetStopped()
		s.broadcast()

		return true

	case dbgp.StatusRunning:
		s.broadcast()

	case dbgp.StatusStarting:
	}

	return false
}

func (s *Session) executeCommand(ctx context.Context, cmd dbgp.Command) (*dbgp.StepResult, error) {
	switch cmd.Type {
	case dbgp.CmdStepInto, dbgp.CmdStepOver, dbgp.CmdStepOut, dbgp.CmdRun:
		return s.doStep(ctx, string(cmd.Type))

	case dbgp.CmdStop:
		_ = s.conn.Close()

		return &dbgp.StepResult{Status: dbgp.StatusStopped}, nil

	case dbgp.CmdBreak:
		return &dbgp.StepResult{Status: dbgp.StatusBreak}, nil

	case dbgp.CmdBreakpointSet:
		if err := s.handleBreakpointSet(ctx, cmd); err != nil {
			return nil, err
		}

		return &dbgp.StepResult{Status: dbgp.StatusBreak}, nil

	case dbgp.CmdBreakpointRemove:
		if err := s.handleBreakpointRemove(ctx, cmd); err != nil {
			return nil, err
		}

		return &dbgp.StepResult{Status: dbgp.StatusBreak}, nil

	default:
		return nil, fmt.Errorf("%s: %w", cmd.Type, ErrUnknownCmdType)
	}
}

func (s *Session) doStep(ctx context.Context, stepCmd string) (*dbgp.StepResult, error) {
	s.txID++
	cmd := fmt.Sprintf("%s -i %d\x00", stepCmd, s.txID)

	if err := s.conn.SendMessage(ctx, cmd); err != nil {
		return nil, fmt.Errorf("send %s: %w", stepCmd, err)
	}

	data, err := s.conn.ReadMessage(ctx)
	if err != nil {
		return nil, fmt.Errorf("read %s response: %w", stepCmd, err)
	}

	return dbgp.ParseStepResponse(data)
}

func (s *Session) refreshStack(ctx context.Context) error {
	s.txID++
	cmd := dbgp.FormatStackGetCmd(s.txID)

	if err := s.conn.SendMessage(ctx, cmd); err != nil {
		return fmt.Errorf("send stack_get: %w", err)
	}

	data, err := s.conn.ReadMessage(ctx)
	if err != nil {
		return fmt.Errorf("read stack_get: %w", err)
	}

	frames, err := dbgp.ParseStackGetResponse(data)
	if err != nil {
		return err
	}

	for i := range frames {
		frames[i].Filename = s.toRelative(frames[i].Filename)
	}

	s.state.UpdateStack(frames)

	return nil
}

func (s *Session) refreshLocals(ctx context.Context) error {
	s.txID++
	cmd := dbgp.FormatContextGetCmd(s.txID, 0)

	if err := s.conn.SendMessage(ctx, cmd); err != nil {
		return fmt.Errorf("send context_get: %w", err)
	}

	data, err := s.conn.ReadMessage(ctx)
	if err != nil {
		return fmt.Errorf("read context_get: %w", err)
	}

	vars, err := dbgp.ParseContextGetResponse(data)
	if err != nil {
		return err
	}

	s.state.UpdateLocals(vars)

	return nil
}

func (s *Session) Subscribe() chan *State {
	ch := make(chan *State, maxMsgBuf)
	s.subs = append(s.subs, ch)

	return ch
}

func (s *Session) Unsubscribe(ch chan *State) {
	for i, sub := range s.subs {
		if sub == ch {
			s.subs = append(s.subs[:i], s.subs[i+1:]...)

			break
		}
	}
}

func (s *Session) SendCommand(cmd dbgp.Command) {
	s.cmds <- cmd
}

func (s *Session) Done() <-chan struct{} {
	return s.done
}

func (s *Session) State() *State {
	return s.state.Snapshot()
}

func (s *Session) toRelative(uri string) string {
	return pathmap.ToRelative(uri, s.remoteRoot)
}

func (s *Session) toURI(relativePath string) string {
	return pathmap.ToURI(relativePath, s.root, s.remoteRoot)
}

func (s *Session) handleBreakpointSet(ctx context.Context, cmd dbgp.Command) error {
	file := cmd.Args["file"]

	lineStr := cmd.Args["line"]
	if file == "" || lineStr == "" {
		return ErrMissingBreakpointArgs
	}

	line, err := strconv.Atoi(lineStr)
	if err != nil {
		return fmt.Errorf("breakpoint_set invalid line: %w", err)
	}

	s.txID++

	uri := s.toURI(file)
	bpCmd := dbgp.FormatBreakpointSetCmd(s.txID, uri, line)

	if err := s.conn.SendMessage(ctx, bpCmd); err != nil {
		return fmt.Errorf("send breakpoint_set: %w", err)
	}

	data, err := s.conn.ReadMessage(ctx)
	if err != nil {
		return fmt.Errorf("read breakpoint_set: %w", err)
	}

	bpID, err := dbgp.ParseBreakpointSetResponse(data)
	if err != nil {
		return fmt.Errorf("parse breakpoint_set: %w", err)
	}

	key := file + ":" + lineStr
	s.breakpoints[key] = dbgp.Breakpoint{ID: bpID, File: file, Line: line}
	s.syncBreakpointsState()

	return nil
}

func (s *Session) handleBreakpointRemove(ctx context.Context, cmd dbgp.Command) error {
	id := cmd.Args["id"]
	if id == "" {
		return ErrMissingBreakpointArgs
	}

	s.txID++

	rmCmd := dbgp.FormatBreakpointRemoveCmd(s.txID, id)

	if err := s.conn.SendMessage(ctx, rmCmd); err != nil {
		return fmt.Errorf("send breakpoint_remove: %w", err)
	}

	data, err := s.conn.ReadMessage(ctx)
	if err != nil {
		return fmt.Errorf("read breakpoint_remove: %w", err)
	}

	if _, err := dbgp.ParseStepResponse(data); err != nil {
		return fmt.Errorf("parse breakpoint_remove: %w", err)
	}

	for k, bp := range s.breakpoints {
		if bp.ID == id {
			delete(s.breakpoints, k)

			break
		}
	}

	s.syncBreakpointsState()

	return nil
}

func (s *Session) syncBreakpointsState() {
	bps := make([]dbgp.Breakpoint, 0, len(s.breakpoints))

	for _, bp := range s.breakpoints {
		bps = append(bps, bp)
	}

	s.state.UpdateBreakpoints(bps)
}

func (s *Session) broadcast() {
	snap := s.state.Snapshot()

	for _, sub := range s.subs {
		select {
		case sub <- snap:
		default:
		}
	}
}
