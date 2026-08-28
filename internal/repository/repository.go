package repository

import (
	"context"
	"errors"

	"github.com/mertkanakkoc/ranking_search/internal/domain"
	"github.com/mertkanakkoc/ranking_search/internal/provider"
)

type Sortfield string

var ErrNotFound = errors.New("repository: not found")

const (
	SortByScore Sortfield = "score"
	SortByDate  Sortfield = "date"
)

type SearchParams struct {
	Query   string
	Type    domain.ContentType
	Sort    Sortfield
	Page    int
	PerPage int
}

type SearchResult struct {
	Items []domain.Content
	Total int
}

type ProviderStatusUpdate struct {
	Name   string
	Status string
	Error  string
}

type ContentRepository interface {
	Upsert(ctx context.Context, c domain.Content) error
	Search(ctx context.Context, params SearchParams) (SearchResult, error)
	Get(ctx context.Context, id string) (domain.Content, error)
}

type ProviderRepository interface {
	ListActive(ctx context.Context) ([]provider.ProviderConfig, error)
	UpdateStatus(ctx context.Context, update ProviderStatusUpdate) error
}
