package provider

import (
	"context"

	"github.com/mertkanakkoc/ranking_search/internal/domain"
)

type Provider interface {
	Name() string
	Fetch(ctx context.Context) ([]domain.Content, error)
}

type ProviderConfig struct {
	Name         string
	Format       string
	BaseURL      string
	RateLimitRPS float64
	TimeoutMS    int
}
