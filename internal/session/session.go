package session

import (
	"context"
	"log/slog"
	"sync"

	"github.com/thumbrise/xdebug-web/pkg/plugins"
)

const maxMsgBuf = 64

type Session struct {
	id        string
	dbg       plugins.Debugger
	subs      []chan *plugins.State
	mu        sync.RWMutex
	lastState *plugins.State
}

func NewSession(id string, dbg plugins.Debugger) *Session {
	return &Session{id: id, dbg: dbg}
}

func (s *Session) ID() string {
	return s.id
}

func (s *Session) Run(ctx context.Context) error {
	go func() {
		err := s.dbg.Run(ctx)
		if err != nil {
			slog.ErrorContext(ctx, "session run error", slog.String("error", err.Error()))
		}
	}()

	s.mu.RLock()
	subs := make([]chan *plugins.State, len(s.subs))
	copy(subs, s.subs)
	s.mu.RUnlock()

	for state := range s.dbg.State() {
		s.mu.Lock()
		s.lastState = state

		for _, sub := range s.subs {
			select {
			case sub <- state:
			default:
			}
		}
		s.mu.Unlock()
	}

	return nil
}

func (s *Session) SendCommand(cmd plugins.Command) {
	select {
	case s.dbg.Commands() <- cmd:
	default:
	}
}

func (s *Session) Subscribe() chan *plugins.State {
	ch := make(chan *plugins.State, maxMsgBuf)

	s.mu.Lock()
	s.subs = append(s.subs, ch)
	s.mu.Unlock()

	return ch
}

func (s *Session) Unsubscribe(ch chan *plugins.State) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, sub := range s.subs {
		if sub == ch {
			s.subs = append(s.subs[:i], s.subs[i+1:]...)

			return
		}
	}
}

func (s *Session) Debugger() plugins.Debugger {
	return s.dbg
}

func (s *Session) CurrentState() *plugins.State {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.lastState
}

func (s *Session) Close() {
	err := s.dbg.Close()
	if err != nil {
		slog.Error("close dbg", slog.String("error", err.Error()), slog.String("session_id", s.id))
	}
}
