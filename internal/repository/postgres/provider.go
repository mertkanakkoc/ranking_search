package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mertkanakkoc/ranking_search/internal/provider"
	"github.com/mertkanakkoc/ranking_search/internal/repository"
)

type ProviderRepository struct {
	pool *pgxpool.Pool
}

func NewProviderRepository(pool *pgxpool.Pool) *ProviderRepository {
	return &ProviderRepository{pool: pool}
}

func (r *ProviderRepository) ListActive(ctx context.Context) ([]provider.ProviderConfig, error) {
	const query = `
		SELECT name, format, base_url, rate_limit_rps, timeout_ms
		FROM providers
		WHERE enabled = true
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("postgres: list active providers: %w", err)
	}
	defer rows.Close()

	var configs []provider.ProviderConfig
	for rows.Next() {
		var cfg provider.ProviderConfig
		if err := rows.Scan(&cfg.Name, &cfg.Format, &cfg.BaseURL, &cfg.RateLimitRPS, &cfg.TimeoutMS); err != nil {
			return nil, fmt.Errorf("postgres: scan provider config: %w", err)
		}
		configs = append(configs, cfg)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: rows: %w", err)
	}

	return configs, nil
}

func (r *ProviderRepository) UpdateStatus(ctx context.Context, update repository.ProviderStatusUpdate) error {
	const query = `
		UPDATE providers
		SET last_fetched_at = now(),
			last_status = $2,
			last_error = NULLIF($3, '')
		WHERE name = $1
	`

	_, err := r.pool.Exec(ctx, query, update.Name, update.Status, update.Error)
	if err != nil {
		return fmt.Errorf("postgres: update provider status: %w", err)
	}

	return nil
}
