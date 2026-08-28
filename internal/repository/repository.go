package repository

import (
	"context"

	"github.com/mertkanakkoc/ranking_search/internal/domain"
)

type Sortfield string

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

type ContentRepository interface {
	Upsert(ctx context.Context, c domain.Content) error
	Search(ctx context.Context, params SearchParams) (SearchResult, error)
}
