package jsonprovider

import (
	"testing"
	"time"

	"github.com/mertkanakkoc/ranking_search/internal/domain"
)

func TestParse(t *testing.T) {
	data := []byte(`{
		"contents": [
			{
				"id": "v1",
				"title": "Go Programming Tutorial",
				"type": "video",
				"metrics": { "views": 15000, "likes": 1200, "duration": "15:30" },
				"published_at": "2024-03-15T10:00:00Z",
				"tags": ["programming", "tutorial"]
			}
		],
		"pagination": { "total": 150, "page": 1, "per_page": 10 }
	}`)

	contents, err := Parse(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(contents) != 1 {
		t.Fatalf("expected 1 content, got %d", len(contents))
	}

	c := contents[0]

	if c.ExternalID != "v1" {
		t.Errorf("ExternalID = %q, want %q", c.ExternalID, "v1")
	}
	if c.Provider != "provider1" {
		t.Errorf("Provider = %q, want %q", c.Provider, "provider1")
	}
	if c.Type != domain.ContentTypeVideo {
		t.Errorf("Type = %q, want %q", c.Type, domain.ContentTypeVideo)
	}
	if c.Metrics.Views != 15000 || c.Metrics.Likes != 1200 {
		t.Errorf("Metrics = %+v, want views=15000 likes=1200", c.Metrics)
	}

	wantTime := time.Date(2024, 3, 15, 10, 0, 0, 0, time.UTC)
	if !c.PublishedAt.Equal(wantTime) {
		t.Errorf("PublishedAt = %v, want %v", c.PublishedAt, wantTime)
	}
	if len(c.Tags) != 2 || c.Tags[0] != "programming" {
		t.Errorf("Tags = %v", c.Tags)
	}
	if len(c.RawMetrics) == 0 {
		t.Error("RawMetrics should not be empty")
	}
}

func TestParse_PartialFailure(t *testing.T) {
	data := []byte(`{
		"contents": [
			{ "id": "x1", "title": "Mystery", "type": "podcast",
			  "metrics": {}, "published_at": "2024-01-01T00:00:00Z", "tags": [] },
			{ "id": "v9", "title": "Valid Video", "type": "video",
			  "metrics": {"views": 100, "likes": 10},
			  "published_at": "2024-01-01T00:00:00Z", "tags": [] }
		]
	}`)

	contents, err := Parse(data)
	if err == nil {
		t.Fatal("expected error for unknown content type")
	}
	if len(contents) != 1 {
		t.Fatalf("expected 1 surviving content, got %d", len(contents))
	}
	if contents[0].ExternalID != "v9" {
		t.Errorf("expected surviving content to be v9, got %s", contents[0].ExternalID)
	}
}
