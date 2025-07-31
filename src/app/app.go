package app

import (
	"coffee-and-running/src/config"
	"coffee-and-running/src/observability/metrics"
	"coffee-and-running/src/storage"
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"
)

type Application interface {
	Run()
}

type application struct {
	config *config.Config
	logger *zap.Logger
	engine storage.Engine
	server *http.Server
	stats  metrics.Agent
}

func New(config *config.Config, logger *zap.Logger, stats metrics.Agent, engine storage.Engine, server *http.Server) Application {
	return &application{
		config: config,
		logger: logger,
		engine: engine,
		server: server,
		stats:  stats,
	}
}

func (a *application) Run() {
	// Create a channel to receive OS signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start server in a goroutine
	go func() {
		log.Printf("Starting server on %s", a.server.Addr)

		var err error
		if a.config.Server.TLS.Enabled {
			err = a.server.ListenAndServeTLS(a.config.Server.TLS.CertFile, a.config.Server.TLS.KeyFile)
		} else {
			err = a.server.ListenAndServe()
		}

		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()
	// Wait for interrupt signal
	<-sigChan
	log.Println("Shutting down server...")

	// Create a context with timeout for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), a.config.Server.ShutdownTimeout)
	defer cancel()

	// Attempt graceful shutdown
	if err := a.server.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	} else {
		log.Println("Server gracefully stopped")
	}
}
