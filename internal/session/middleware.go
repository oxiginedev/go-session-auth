package session

import (
	"context"
	"net/http"

	"github.com/oxiginedev/go-session/utils"
)

type SessionContextKey struct{}

type sessionResponseWriter struct {
	http.ResponseWriter
	manager *Manager
	r       *http.Request
	done    bool
}

func (w *sessionResponseWriter) Write(b []byte) (int, error) {
	maybeSetCookie(w)
	return w.ResponseWriter.Write(b)
}

func (w *sessionResponseWriter) WriteHeader(code int) {
	maybeSetCookie(w)
	w.ResponseWriter.WriteHeader(code)
}

func (w *sessionResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func (m *Manager) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, err := m.StartSession(r.Context(), r)
		if err != nil {
			// Then something bad really went wrong
			panic(err)
		}

		ctx := context.WithValue(r.Context(), SessionContextKey{}, session)
		r = r.WithContext(ctx)

		sw := &sessionResponseWriter{
			ResponseWriter: w,
			manager:        m,
			r:              r,
			done:           false,
		}

		w.Header().Add("Vary", "Cookie")
		w.Header().Add("Cache-Control", `no-cache="Set-Cookie"`)

		next.ServeHTTP(sw, r)

		if err := m.SaveSession(r.Context(), session); err != nil {
			panic(err)
		}

		maybeSetCookie(sw)
	})
}

func (m *Manager) VerifyCSRFToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method := r.Method

		methods := []string{"POST", "PUT", "PATCH", "DELETE"}

		if utils.SliceContains(methods, method) {
			session := GetSessionFromContext(r.Context())
			if session == nil {
				http.Error(w, "session required", http.StatusUnauthorized)
				return
			}

			if err := m.ValidateCSRFToken(r, session); err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

func GetSessionFromContext(ctx context.Context) *Session {
	if session, ok := ctx.Value(SessionContextKey{}).(*Session); ok {
		return session
	}

	return nil
}

func maybeSetCookie(sw *sessionResponseWriter) {
	if sw.done {
		return
	}

	session := GetSessionFromContext(sw.r.Context())

	http.SetCookie(sw.ResponseWriter, &http.Cookie{
		Name:     sw.manager.config.Name,
		Value:    session.id,
		Domain:   sw.manager.config.Domain,
		Path:     "/",
		Expires:  session.expiresAt,
		HttpOnly: sw.manager.config.HTTPOnly,
		Secure:   sw.manager.config.Secure,
		SameSite: sw.manager.config.SameSite,
	})

	csrfToken := session.Get("csrf_token").(string)

	http.SetCookie(sw.ResponseWriter, &http.Cookie{
		Name:     "XSRF-TOKEN",
		Value:    csrfToken,
		Domain:   sw.manager.config.Domain,
		Path:     "/",
		HttpOnly: sw.manager.config.HTTPOnly,
		Secure:   sw.manager.config.Secure,
		SameSite: sw.manager.config.SameSite,
	})

	sw.done = true
}
