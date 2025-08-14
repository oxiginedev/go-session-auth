package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/oxiginedev/go-session/internal/session"
)

type Handler struct {
	sm *session.Manager
}

func New(sessionManager *session.Manager) *Handler {
	return &Handler{
		sm: sessionManager,
	}
}

func (h *Handler) InitRoutes() http.Handler {
	router := chi.NewRouter()

	router.Use(h.sm.Handle)
	router.Use(h.sm.VerifyCSRFToken)
	router.Use(middleware.Recoverer)

	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	router.Route("/auth", func(r chi.Router) {
		r.Post("/register", h.register)
	})

	return router
}
