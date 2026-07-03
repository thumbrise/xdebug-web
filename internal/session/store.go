package session

import (
	"sync"
)

type Store struct {
	mu       sync.RWMutex
	sessions []*Session
}

func NewStore() *Store {
	return &Store{}
}

func (s *Store) Add(session *Session) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sessions = append(s.sessions, session)
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
