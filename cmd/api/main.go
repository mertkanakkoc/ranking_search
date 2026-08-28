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

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mertkanakkoc/ranking_search/internal/httpapi"
	"github.com/mertkanakkoc/ranking_search/internal/ingest"
	"github.com/mertkanakkoc/ranking_search/internal/repository/postgres"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		slog.Error("DATABASE_URL is not set")
		os.Exit(1)
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		slog.Error("connect to postgres", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		slog.Error("ping postgres", "error", err)
		os.Exit(1)
	}

	providerRepo := postgres.NewProviderRepository(pool)
	contentRepo := postgres.NewContentRepository(pool)

	interval := 5 * time.Minute
	if v := os.Getenv("INGEST_INTERVAL"); v != "" {
		if parsed, err := time.ParseDuration(v); err == nil {
			interval = parsed
		} else {
			slog.Warn("invalid INGEST_INTERVAL, using default", "value", v, "default", interval)
		}
	}

	ingestService := ingest.NewService(providerRepo, contentRepo, interval)

	go func() {
		slog.Info("starting ingest service", "interval", interval)
		ingestService.Run(ctx)
	}()

	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	server := &http.Server{
		Addr:         addr,
		Handler:      httpapi.NewRouter(contentRepo),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	serverErrCh := make(chan error, 1)
	go func() {
		slog.Info("starting http server", "addr", addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrCh <- err
			return
		}
		serverErrCh <- nil
	}()

	select {
	case <-ctx.Done():
		slog.Info("shutdown signal received")
	case err := <-serverErrCh:
		if err != nil {
			slog.Error("http server error", "error", err)
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("http server shutdown", "error", err)
	}

	slog.Info("shutting down")
}
