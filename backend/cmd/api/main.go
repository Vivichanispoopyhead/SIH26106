package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"sih26106/backend/internal/config"
	"sih26106/backend/internal/httpapi"
)

func main() {
	cfg := config.Load()
	logger := newLogger(cfg.Environment)

	server := &http.Server{
		Addr:              cfg.Address,
		Handler:           httpapi.NewRouter(logger, cfg.AllowedOrigins),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logger.Info("HTTP server starting", "address", cfg.Address, "environment", cfg.Environment)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("HTTP server failed", "error", err)
			os.Exit(1)
		}
	}()

	shutdownSignal := make(chan os.Signal, 1)
	signal.Notify(shutdownSignal, os.Interrupt, syscall.SIGTERM)
	<-shutdownSignal

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
		return
	}

	logger.Info("HTTP server stopped")
}

func newLogger(environment string) *slog.Logger {
	options := &slog.HandlerOptions{}
	if environment == "development" {
		options.Level = slog.LevelDebug
	}

	return slog.New(slog.NewJSONHandler(os.Stdout, options))
}
