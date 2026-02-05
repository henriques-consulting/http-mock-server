package app

import (
	"context"
	"errors"
	"fmt"
	"http-mock-server/pkg/version"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"http-mock-server/internal/config"
	"http-mock-server/internal/handler"
)

// App represents the application
type App struct {
	config *config.Config
	server *http.Server
}

// New creates a new application instance
func New() *App {
	return &App{}
}

// Run starts the application
func (a *App) Run() error {
	log.Printf("Starting HTTP mock server %s\n", version.Version)
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	a.config = cfg

	// Setup HTTP server
	a.setupServer()

	// Start server in goroutine
	serverErr := make(chan error, 1)
	go func() {
		log.Printf("Listening on port %d\n", cfg.Server.Port)
		if err := a.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- fmt.Errorf("server failed: %w", err)
		}
		close(serverErr)
	}()

	// Wait for shutdown signal or server error
	return a.waitForShutdown(serverErr)
}

func (a *App) setupServer() {
	mux := http.NewServeMux()

	// Add health check endpoint
	mux.HandleFunc(
		"/health", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("OK"))
		},
	)

	// Add mock handler
	mockHandler := handler.NewMockHandler(a.config)
	mux.Handle("/", handler.LoggingMiddleware(mockHandler))

	a.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", a.config.Server.Port),
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
}

func (a *App) waitForShutdown(serverErr <-chan error) error {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		return err
	case sig := <-sigChan:
		log.Printf("Received signal %v, shutting down...", sig)
	}

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := a.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	log.Println("Server stopped gracefully")
	return nil
}
