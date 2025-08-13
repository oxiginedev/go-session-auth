package memory

import (
	"context"
	"sync"
	"time"

	"github.com/oxiginedev/go-session/internal/session"
)

type MemoryStore struct {
	mu       sync.RWMutex
	sessions map[string]*session.Session
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		sessions: make(map[string]*session.Session),
	}
}

func (m *MemoryStore) Get(_ context.Context, id string) (*session.Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ses, exists := m.sessions[id]
	if !exists {
		return nil, session.ErrSessionNotFound
	}

	if time.Now().After(ses.GetExpiresAt()) {
		delete(m.sessions, id)
		return nil, session.ErrSessionExpired
	}

	return ses, nil
}

func (m *MemoryStore) Set(_ context.Context, session *session.Session) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.sessions[session.GetID()] = session
	return nil
}

func (m *MemoryStore) Delete(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.sessions, id)
	return nil
}

func (m *MemoryStore) DeleteExpired(_ context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()

	for i, ses := range m.sessions {
		if now.After(ses.GetExpiresAt()) {
			delete(m.sessions, i)
		}
	}

	return nil
}

func (m *MemoryStore) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.sessions = make(map[string]*session.Session)
	return nil
}
