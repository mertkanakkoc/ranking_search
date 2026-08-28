package httpapi

import "time"

type ContentResponse struct {
	ExternalID  string    `json:"external_id"`
	Provider    string    `json:"provider"`
	Title       string    `json:"title"`
	Type        string    `json:"type"`
	Score       float64   `json:"score"`
	PublishedAt time.Time `json:"published_at"`
	Tags        []string  `json:"tags"`
}

type SearchResponse struct {
	Data []ContentResponse `json:"data"`
	Meta MetaResponse      `json:"meta"`
}

type MetaResponse struct {
	Total   int `json:"total"`
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}
