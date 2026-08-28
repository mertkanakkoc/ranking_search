package cached

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"time"

	"github.com/mertkanakkoc/ranking_search/internal/cache"
	"github.com/mertkanakkoc/ranking_search/internal/domain"
	"github.com/mertkanakkoc/ranking_search/internal/repository"
)

const (
	searchTTL = 60 * time.Second
	getTTL    = 5 * time.Minute
)

type ContentRepository struct {
	inner repository.ContentRepository
	cache cache.Cache
}

func NewContentRepository(inner repository.ContentRepository, c cache.Cache) *ContentRepository {
	return &ContentRepository{inner: inner, cache: c}
}

func (r *ContentRepository) Upsert(ctx context.Context, c domain.Content) error {
	if err := r.inner.Upsert(ctx, c); err != nil {
		return err
	}

	if err := r.cache.Delete(ctx, getCacheKey(c.UniqueKey())); err != nil {
		slog.Warn("cached: invalidate get cache", "id", c.UniqueKey(), "error", err)
	}

	return nil
}

func (r *ContentRepository) Search(ctx context.Context, params repository.SearchParams) (repository.SearchResult, error) {
	key := searchCacheKey(params)

	var cached repository.SearchResult
	hit, err := r.cache.Get(ctx, key, &cached)
	if err != nil {
		slog.Warn("cached: search cache get failed", "error", err)
	}
	if hit {
		return cached, nil
	}

	result, err := r.inner.Search(ctx, params)
	if err != nil {
		return repository.SearchResult{}, err
	}

	if err := r.cache.Set(ctx, key, result, searchTTL); err != nil {
		slog.Warn("cached: search cache set failed", "error", err)
	}

	return result, nil
}

func (r *ContentRepository) Get(ctx context.Context, id string) (domain.Content, error) {
	key := getCacheKey(id)

	var cached domain.Content
	hit, err := r.cache.Get(ctx, key, &cached)
	if err != nil {
		slog.Warn("cached: get cache get failed", "error", err)
	}
	if hit {
		return cached, nil
	}

	c, err := r.inner.Get(ctx, id)
	if err != nil {
		return domain.Content{}, err
	}

	if err := r.cache.Set(ctx, key, c, getTTL); err != nil {
		slog.Warn("cached: get cache set failed", "error", err)
	}

	return c, nil
}

func searchCacheKey(p repository.SearchParams) string {
	raw := fmt.Sprintf("q=%s&type=%s&sort=%s&page=%d&per_page=%d", p.Query, p.Type, p.Sort, p.Page, p.PerPage)
	sum := sha256.Sum256([]byte(raw))
	return "search:" + hex.EncodeToString(sum[:])
}

func getCacheKey(id string) string {
	return "content:" + id
}
