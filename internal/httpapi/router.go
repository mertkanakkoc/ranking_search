package httpapi

import (
	"net/http"

	"github.com/mertkanakkoc/ranking_search/internal/repository"
	"github.com/mertkanakkoc/ranking_search/web/dashboard"
)

func NewRouter(contentRepo repository.ContentRepository) http.Handler {
	mux := http.NewServeMux()

	searchHandler := NewSearchHandler(contentRepo)

	mux.Handle("/", dashboard.Handler())
	mux.HandleFunc("GET /api/v1/contents", searchHandler.Search)
	mux.HandleFunc("GET /api/v1/health", Health)
	mux.HandleFunc("GET /api/v1/contents/{id}", searchHandler.GetByID)

	return mux
}
