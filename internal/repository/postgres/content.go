package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mertkanakkoc/ranking_search/internal/domain"
	"github.com/mertkanakkoc/ranking_search/internal/repository"
)

type ContentRepository struct {
	pool *pgxpool.Pool
}

func NewContentRepository(pool *pgxpool.Pool) *ContentRepository {
	return &ContentRepository{pool: pool}
}

func (r *ContentRepository) Upsert(ctx context.Context, c domain.Content) error {
	const query = `
		INSERT INTO contents (
			id, external_id, provider, title, content_type,
			views, likes, reading_time, reactions,
			published_at, tags, raw_metrics, final_score
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9,
			$10, $11, $12, $13
		)
		ON CONFLICT (id) DO UPDATE SET
			title        = EXCLUDED.title,
			content_type = EXCLUDED.content_type,
			views        = EXCLUDED.views,
			likes        = EXCLUDED.likes,
			reading_time = EXCLUDED.reading_time,
			reactions    = EXCLUDED.reactions,
			published_at = EXCLUDED.published_at,
			tags         = EXCLUDED.tags,
			raw_metrics  = EXCLUDED.raw_metrics,
			final_score  = EXCLUDED.final_score,
			fetched_at   = now()
	`

	_, err := r.pool.Exec(ctx, query,
		c.UniqueKey(), c.ExternalID, c.Provider, c.Title, string(c.Type),
		c.Metrics.Views, c.Metrics.Likes, c.Metrics.ReadingTime, c.Metrics.Reactions,
		c.PublishedAt, c.Tags, c.RawMetrics, c.FinalScore,
	)
	if err != nil {
		return fmt.Errorf("postgres: upsert content: %w", err)
	}
	return nil
}

func (r *ContentRepository) Search(ctx context.Context, params repository.SearchParams) (repository.SearchResult, error) {
	page := params.Page
	if page < 1 {
		page = 1
	}
	perPage := params.PerPage
	if perPage < 1 {
		perPage = 20
	}
	offset := (page - 1) * perPage

	var (
		conditions []string
		args       []any
		argPos     = 1
	)

	if params.Query != "" {
		conditions = append(conditions, fmt.Sprintf("search_vector @@ plainto_tsquery('english', $%d)", argPos))
		args = append(args, params.Query)
		argPos++
	}

	if params.Type != "" {
		conditions = append(conditions, fmt.Sprintf("content_type = $%d", argPos))
		args = append(args, string(params.Type))
		argPos++
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	orderBy := "final_score DESC"
	if params.Sort == repository.SortByDate {
		orderBy = "published_at DESC"
	}

	query := fmt.Sprintf(`
		SELECT
			external_id, provider, title, content_type,
			views, likes, reading_time, reactions,
			published_at, tags, raw_metrics, final_score,
			COUNT(*) OVER() AS total_count
		FROM contents
		%s
		ORDER BY %s
		LIMIT $%d OFFSET $%d
	`, where, orderBy, argPos, argPos+1)

	args = append(args, perPage, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return repository.SearchResult{}, fmt.Errorf("postgres: search: %w", err)
	}
	defer rows.Close()

	var (
		items []domain.Content
		total int
	)

	for rows.Next() {
		var (
			c           domain.Content
			contentType string
		)
		if err := rows.Scan(
			&c.ExternalID, &c.Provider, &c.Title, &contentType,
			&c.Metrics.Views, &c.Metrics.Likes, &c.Metrics.ReadingTime, &c.Metrics.Reactions,
			&c.PublishedAt, &c.Tags, &c.RawMetrics, &c.FinalScore,
			&total,
		); err != nil {
			return repository.SearchResult{}, fmt.Errorf("postgres: scan content: %w", err)
		}
		c.Type = domain.ContentType(contentType)
		items = append(items, c)
	}

	if err := rows.Err(); err != nil {
		return repository.SearchResult{}, fmt.Errorf("postgres: rows: %w", err)
	}

	return repository.SearchResult{Items: items, Total: total}, nil
}

func (r *ContentRepository) Get(ctx context.Context, id string) (domain.Content, error) {
	const query = `
		SELECT external_id, provider, title, content_type,
		       views, likes, reading_time, reactions,
		       published_at, tags, raw_metrics, final_score
		FROM contents
		WHERE id = $1
	`

	var (
		c           domain.Content
		contentType string
	)

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&c.ExternalID, &c.Provider, &c.Title, &contentType,
		&c.Metrics.Views, &c.Metrics.Likes, &c.Metrics.ReadingTime, &c.Metrics.Reactions,
		&c.PublishedAt, &c.Tags, &c.RawMetrics, &c.FinalScore,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Content{}, repository.ErrNotFound
		}
		return domain.Content{}, fmt.Errorf("postgres: get content: %w", err)
	}

	c.Type = domain.ContentType(contentType)
	return c, nil
}
