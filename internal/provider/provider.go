package provider

import (
	"context"

	"github.com/mertkanakkoc/ranking_search/internal/domain"
)

type Provider interface {
	Name() string
	Fetch(ctx context.Context) ([]domain.Content, error)
}
