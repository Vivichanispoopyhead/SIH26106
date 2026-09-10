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

	"sih26106/backend/internal/ai"
	"sih26106/backend/internal/config"
	"sih26106/backend/internal/email"
	"sih26106/backend/internal/httpapi"
	"sih26106/backend/internal/persistence"
)

func main() {
	cfg := config.Load()
	logger := newLogger(cfg.Environment)
	configuredAnalyzer, err := ai.NewGeminiAnalyzerFromEnv()
	if err != nil {
		logger.Error("Gemini configuration is invalid", "error", err)
		os.Exit(1)
	}
	store, err := persistence.NewPostgresStore(context.Background(), cfg.DatabaseURL)
	if err != nil {
		logger.Error("postgres initialization failed", "error", err)
		os.Exit(1)
	}
	defer store.Close()

	server := &http.Server{
		Addr:              cfg.Address,
		Handler:           httpapi.NewRouter(logger, cfg.AllowedOrigins, email.NewServiceWithAnalyzer(store, configuredAnalyzer)),
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
