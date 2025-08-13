package session

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/oxiginedev/go-session/utils"
)

var (
	ErrSessionNotFound = errors.New("session not found")
	ErrSessionExpired  = errors.New("session has expired")
	ErrSessionHijacked = errors.New("session is hijacked")
	ErrCSRFMismatch    = errors.New("csrf token mismatch")

	DefaultSessionConfig = &SessionConfig{
		Name:             "go_session",
		Timeout:          time.Hour,
		MaxAge:           24 * time.Hour,
		Secure:           false,
		HTTPOnly:         false,
		SameSite:         http.SameSiteLaxMode,
		EncryptSession:   true,
		EnableCSRF:       true,
		RegenerateOnAuth: true,
		CleanupInterval:  time.Hour,
		TokenLength:      32,
	}
)

type Session struct {
	id           string         `json:"id"`
	userID       string         `json:"user_id"`
	payload      map[string]any `json:"payload"`
	ipAddress    string         `json:"ip_address"`
	userAgent    string         `json:"user_agent"`
	lastActivity time.Time      `json:"last_activity"`
	createdAt    time.Time      `json:"created_at"`
	expiresAt    time.Time      `json:"expires_at"`
	fingerprint  string         `json:"fingerprint"`

	mu sync.RWMutex
}

func (s *Session) Get(key string) any {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.payload[key]
}

func (s *Session) Put(key string, value any) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.payload[key] = value
}

func (s *Session) Has(key string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.payload[key]; ok {
		return true
	}

	return false
}

func (s *Session) GetID() string {
	return s.id
}

func (s *Session) GetExpiresAt() time.Time {
	return s.expiresAt
}

type Store interface {
	// Get retrieves a session by ID
	Get(ctx context.Context, id string) (*Session, error)
	// Set stores a session
	Set(ctx context.Context, session *Session) error
	// Delete removes a session
	Delete(ctx context.Context, id string) error
	// DeleteExpired removes expired sessions
	DeleteExpired(ctx context.Context) error
	// Close closes the store connection
	Close() error
}

type SessionConfig struct {
	Name     string
	Timeout  time.Duration
	MaxAge   time.Duration
	Domain   string
	Secure   bool
	HTTPOnly bool
	SameSite http.SameSite

	EncryptSession bool

	EnableCSRF       bool
	RegenerateOnAuth bool
	CleanupInterval  time.Duration

	TokenLength int
}

type Manager struct {
	store    Store
	config   *SessionConfig
	stopChan chan struct{}
	wg       sync.WaitGroup
}

func NewManager(store Store, config *SessionConfig) *Manager {
	if config == nil {
		config = DefaultSessionConfig
	}

	m := &Manager{
		store:    store,
		config:   config,
		stopChan: make(chan struct{}),
	}

	if config.CleanupInterval > 0 {
		m.startCleanupJob()
	}

	return m
}

func (m *Manager) StartSession(ctx context.Context, r *http.Request) (*Session, error) {
	var session *Session

	cookie, err := r.Cookie(m.config.Name)
	if err == nil {
		session, err = m.store.Get(r.Context(), cookie.Value)
		if err != nil && !errors.Is(err, ErrSessionNotFound) {
			return nil, err
		}
	}

	if session == nil || m.validateSession(r, session) != nil {
		if session, err = m.freshSession(r); err != nil {
			return nil, err
		}
	}

	return session, nil
}

func (m *Manager) SaveSession(ctx context.Context, session *Session) error {
	session.lastActivity = time.Now()

	if err := m.store.Set(ctx, session); err != nil {
		return err
	}

	return nil
}

func (m *Manager) GetSession(r *http.Request) (*Session, error) {
	cookie, err := r.Cookie(m.config.Name)
	if err != nil {
		return nil, ErrSessionNotFound
	}

	session, err := m.store.Get(r.Context(), cookie.Value)
	if err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			return nil, ErrSessionNotFound
		}

		return nil, fmt.Errorf("session: failed to retrieve session - %w", err)
	}

	if err := m.validateSession(r, session); err != nil {
		if err := m.store.Delete(r.Context(), session.id); err != nil {
			return nil, fmt.Errorf("failed to delete session")
		}

		return nil, err
	}

	session.lastActivity = time.Now()
	if err := m.store.Set(r.Context(), session); err != nil {
		return nil, fmt.Errorf("failed to update session - %w", err)
	}

	return session, nil
}

func (m *Manager) RegenerateSession(ctx context.Context, session *Session) error {
	session.mu.Lock()
	defer session.mu.Unlock()

	old := session.id
	if err := m.store.Delete(ctx, old); err != nil {
		return err
	}

	new, err := utils.RandomString(m.config.TokenLength)
	if err != nil {
		return err
	}

	session.id = new
	session.lastActivity = time.Now()
	session.fingerprint = ""

	return nil
}

func (m *Manager) DestroySession(w http.ResponseWriter, r *http.Request) error {
	cookie, err := r.Cookie(m.config.Name)
	if err != nil {
		return nil
	}

	if err := m.store.Delete(r.Context(), cookie.Value); err != nil {
		return err
	}

	http.SetCookie(w, &http.Cookie{
		Name:     m.config.Name,
		Value:    "",
		Domain:   m.config.Domain,
		Path:     "/",
		Expires:  time.Unix(0, 0),
		Secure:   m.config.Secure,
		HttpOnly: m.config.HTTPOnly,
		SameSite: m.config.SameSite,
	})

	return nil
}

func (m *Manager) ValidateCSRFToken(r *http.Request, session *Session) error {
	if !m.config.EnableCSRF {
		return nil
	}

	csrf, ok := session.Get("csrf_token").(string)
	if !ok {
		return ErrCSRFMismatch
	}

	token := r.Header.Get("X-XSRF-TOKEN")
	if utils.IsStringEmpty(token) {
		token = r.FormValue("csrf_token")
	}

	if subtle.ConstantTimeCompare([]byte(csrf), []byte(token)) != 1 {
		return ErrCSRFMismatch
	}

	return nil
}

func (m *Manager) Close() error {
	close(m.stopChan)
	m.wg.Wait()
	return m.store.Close()
}

func (m *Manager) freshSession(r *http.Request) (*Session, error) {
	id, err := utils.RandomString(m.config.TokenLength)
	if err != nil {
		return nil, err
	}

	csrf, err := utils.RandomString(32)
	if err != nil {
		return nil, err
	}

	now := time.Now()

	return &Session{
		id:           id,
		userID:       "",
		payload:      map[string]any{"csrf_token": csrf},
		ipAddress:    m.getClientIP(r).String(),
		userAgent:    r.UserAgent(),
		lastActivity: now,
		createdAt:    now,
		expiresAt:    now.Add(m.config.MaxAge),
	}, nil
}

func (m *Manager) validateSession(r *http.Request, session *Session) error {
	cip := m.getClientIP(r)
	cua := r.UserAgent()
	cfp := ""

	if time.Now().After(session.expiresAt) ||
		time.Now().After(session.lastActivity.Add(m.config.Timeout)) {
		return ErrSessionExpired
	}

	if session.userAgent != cua || session.ipAddress != cip.String() {
		return ErrSessionHijacked
	}

	if subtle.ConstantTimeCompare([]byte(session.fingerprint), []byte(cfp)) != 1 {
		return ErrSessionHijacked
	}

	return nil
}

// func (m *Manager) generateFingerPrint(r *http.Request) string {
// 	return ""
// }

func (m *Manager) getClientIP(r *http.Request) net.IP {
	var (
		xForwardedFor = http.CanonicalHeaderKey("X-Forwarded-For")
		xRealIP       = http.CanonicalHeaderKey("X-Real-IP")
	)

	cloudflareIP := r.Header.Get("CF-Connecting-IP")
	if !utils.IsStringEmpty(cloudflareIP) {
		return net.ParseIP(cloudflareIP)
	}

	if xff := r.Header.Get(xForwardedFor); !utils.IsStringEmpty(xff) {
		i := strings.Index(xff, ", ")

		if i == -1 {
			i = len(xff)
		}

		return net.ParseIP(xff[:i])
	}

	if ip := r.Header.Get(xRealIP); !utils.IsStringEmpty(ip) {
		return net.ParseIP(ip)
	}

	h, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return net.IP{}
	}

	return net.ParseIP(h)
}

func (m *Manager) startCleanupJob() {
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()

		ticker := time.NewTicker(m.config.CleanupInterval)
		defer ticker.Stop()

		for {
			select {
			case <-m.stopChan:
				return
			case <-ticker.C:
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
				defer cancel()

				m.store.DeleteExpired(ctx)
			}
		}
	}()
}
