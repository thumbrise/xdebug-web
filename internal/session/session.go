package session

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/thumbrise/xdebug-web/internal/dbgp"
)

const maxMsgBuf = 64

var ErrUnknownCmdType = errors.New("unknown command type")

type Session struct {
	conn   *dbgp.Conn
	state  *State
	cmds   chan dbgp.Command
	subs   []chan *State
	done   chan struct{}
	txID   int
	logger *slog.Logger
}

func NewSession(conn *dbgp.Conn, logger *slog.Logger) *Session {
	return &Session{
		conn:   conn,
		logger: logger.With("subsystem", "session"),
		state:  NewState(),
		txID:   0,
		cmds:   make(chan dbgp.Command, maxMsgBuf),
		done:   make(chan struct{}),
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

	s.state.SetInit(initInfo)
	s.broadcast()

	s.logger.InfoContext(ctx, "xdebug connected",
		"language", initInfo.Language,
		"file", initInfo.FileURI,
		"ideKey", initInfo.IdeKey,
	)

	return s.commandLoop(ctx)
}

func (s *Session) commandLoop(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

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

	s.state.SetStepStatus(result.Status, result.Filename, result.Lineno)

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
		s.state.SetRunning()
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

func (s *Session) broadcast() {
	snap := s.state.Snapshot()

	for _, sub := range s.subs {
		select {
		case sub <- snap:
		default:
		}
	}
}
