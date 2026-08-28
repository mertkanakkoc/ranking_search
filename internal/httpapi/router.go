package httpapi

import (
	"net/http"

	"github.com/mertkanakkoc/ranking_search/internal/repository"
)

func NewRouter(contentRepo repository.ContentRepository) http.Handler {
	mux := http.NewServeMux()

	searchHandler := NewSearchHandler(contentRepo)

	mux.HandleFunc("GET /api/v1/contents", searchHandler.Search)
	mux.HandleFunc("GET /api/v1/health", Health)
	mux.HandleFunc("GET /api/v1/contents/{id}", searchHandler.GetByID)

	return mux
}
