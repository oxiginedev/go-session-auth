package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/oxiginedev/go-session/handlers"
	"github.com/oxiginedev/go-session/internal/session"
	"github.com/oxiginedev/go-session/internal/session/memory"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatalf("failed to load env variables - %v", err)
	}

	sm := session.NewManager(memory.NewMemoryStore(), nil)
	defer func() {
		_ = sm.Close()
	}()

	// db, err := sqlite.New(context.Background())
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// defer func() {
	// 	_ = db.Close()
	// }()
	port, err := strconv.Atoi(os.Getenv("APP_PORT"))
	if err != nil {
		log.Fatal(err)
	}

	handler := handlers.New(sm)

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           handler.InitRoutes(),
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       time.Minute,
		ReadHeaderTimeout: 2 * time.Second,
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
