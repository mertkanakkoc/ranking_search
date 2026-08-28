package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

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

	slog.Info("starting ingest service", "interval", interval)
	ingestService.Run(ctx)

	slog.Info("shutting down")
}
