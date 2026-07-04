package session

import (
	"strings"
	"sync"

	"github.com/thumbrise/xdebug-web/pkg/plugins"
)

type Store struct {
	mu          sync.RWMutex
	sessions    []*Session
	listeners   []chan *Session
	breakpoints map[string]bool // "file:line" → active
}

func NewStore() *Store {
	return &Store{breakpoints: make(map[string]bool)}
}

func (s *Store) SaveBreakpoint(file, line string) {
	key := file + ":" + line

	s.mu.Lock()
	s.breakpoints[key] = true
	s.mu.Unlock()
}

func (s *Store) DeleteBreakpoint(file, line string) {
	key := file + ":" + line

	s.mu.Lock()
	delete(s.breakpoints, key)
	s.mu.Unlock()
}

func (s *Store) SyncBreakpoints(sess *Session) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for key := range s.breakpoints {
		file, lineStr, ok := strings.Cut(key, ":")
		if !ok {
			continue
		}

		sess.SendCommand(plugins.Command{
			Type: plugins.CmdBreakpointSet,
			Args: map[string]string{"file": file, "line": lineStr},
		})
	}
}

func (s *Store) SubscribeNew() chan *Session {
	ch := make(chan *Session, 4)

	s.mu.Lock()
	s.listeners = append(s.listeners, ch)
	s.mu.Unlock()

	return ch
}

func (s *Store) UnsubscribeNew(ch chan *Session) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, l := range s.listeners {
		if l == ch {
			s.listeners = append(s.listeners[:i], s.listeners[i+1:]...)

			return
		}
	}
}

func (s *Store) Add(session *Session) {
	s.mu.Lock()
	s.sessions = append(s.sessions, session)

	listeners := make([]chan *Session, len(s.listeners))
	copy(listeners, s.listeners)
	s.mu.Unlock()

	for _, l := range listeners {
		select {
		case l <- session:
		default:
		}
	}
}

func (s *Store) Remove(session *Session) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, ss := range s.sessions {
		if ss == session {
			s.sessions = append(s.sessions[:i], s.sessions[i+1:]...)

			return
		}
	}
}

func (s *Store) Active() *Session {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.sessions) == 0 {
		return nil
	}

	return s.sessions[len(s.sessions)-1]
}

func (s *Store) All() []*Session {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]*Session, len(s.sessions))
	copy(out, s.sessions)

	return out
}
