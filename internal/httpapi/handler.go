package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/mertkanakkoc/ranking_search/internal/domain"
	"github.com/mertkanakkoc/ranking_search/internal/repository"
)

type SearchHandler struct {
	contentRepo repository.ContentRepository
}

func NewSearchHandler(contentRepo repository.ContentRepository) *SearchHandler {
	return &SearchHandler{contentRepo: contentRepo}
}

func (h *SearchHandler) Search(w http.ResponseWriter, r *http.Request) {
	params, err := parseSearchParams(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := h.contentRepo.Search(r.Context(), params)
	if err != nil {
		slog.Error("httpapi: search", "error", err)
		writeError(w, http.StatusInternalServerError, "search failed")
		return
	}

	resp := SearchResponse{
		Data: toContentResponses(result.Items),
		Meta: MetaResponse{
			Total:   result.Total,
			Page:    params.Page,
			PerPage: params.PerPage,
		},
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *SearchHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	c, err := h.contentRepo.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "content not found")
			return
		}
		slog.Error("httpapi: get content", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to get content")
		return
	}

	writeJSON(w, http.StatusOK, toContentResponse(c))
}

func parseSearchParams(r *http.Request) (repository.SearchParams, error) {
	q := r.URL.Query()

	params := repository.SearchParams{
		Query:   q.Get("q"),
		Page:    1,
		PerPage: 20,
	}

	if typeParam := q.Get("type"); typeParam != "" {
		ct := domain.ContentType(typeParam)
		if ct != domain.ContentTypeVideo && ct != domain.ContentTypeText {
			return repository.SearchParams{}, fmt.Errorf("invalid type: %q (expected video or text)", typeParam)
		}
		params.Type = ct
	}

	switch sortParam := q.Get("sort"); sortParam {
	case "", "score":
		params.Sort = repository.SortByScore
	case "date":
		params.Sort = repository.SortByDate
	default:
		return repository.SearchParams{}, fmt.Errorf("invalid sort: %q (expected score or date)", sortParam)
	}

	if pageParam := q.Get("page"); pageParam != "" {
		page, err := strconv.Atoi(pageParam)
		if err != nil || page < 1 {
			return repository.SearchParams{}, fmt.Errorf("invalid page: %q", pageParam)
		}
		params.Page = page
	}

	if perPageParam := q.Get("per_page"); perPageParam != "" {
		perPage, err := strconv.Atoi(perPageParam)
		if err != nil || perPage < 1 || perPage > 100 {
			return repository.SearchParams{}, fmt.Errorf("invalid per_page: %q (must be 1-100)", perPageParam)
		}
		params.PerPage = perPage
	}

	return params, nil
}

func toContentResponse(c domain.Content) ContentResponse {
	return ContentResponse{
		ExternalID:  c.ExternalID,
		Provider:    c.Provider,
		Title:       c.Title,
		Type:        string(c.Type),
		Score:       c.FinalScore,
		PublishedAt: c.PublishedAt,
		Tags:        c.Tags,
	}
}

func toContentResponses(items []domain.Content) []ContentResponse {
	responses := make([]ContentResponse, 0, len(items))
	for _, c := range items {
		responses = append(responses, toContentResponse(c))
	}
	return responses
}

func Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		slog.Error("httpapi: encode response", "error", err)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, ErrorResponse{Error: message})
}
