package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/oxiginedev/go-session/internal/session"
	"github.com/oxiginedev/go-session/internal/session/memory"
)

func main() {
	sm := session.NewManager(memory.NewMemoryStore(), nil)
	defer func() {
		_ = sm.Close()
	}()

	mux := http.NewServeMux()

	mux.Handle("/", sm.VerifyCSRFToken(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("Server is active!"))
		},
	)))

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", 5000),
		Handler: sm.Handle(mux),
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGQUIT, syscall.SIGTERM)

	log.Printf("Listening at %s\n", "http://localhost:5000")

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("failed to listen")
		}
	}()

	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("failed to shutdown")
	}

	log.Println("server has shutdown")
}
