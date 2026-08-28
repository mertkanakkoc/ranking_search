package xmlprovider

import (
	"testing"
	"time"

	"github.com/mertkanakkoc/ranking_search/internal/domain"
)

func TestParse(t *testing.T) {
	data := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<feed>
  <items>
    <item>
      <id>v1</id>
      <headline>Introduction to Docker</headline>
      <type>video</type>
      <stats>
        <views>22000</views>
        <likes>1800</likes>
        <duration>25:15</duration>
      </stats>
      <publication_date>2024-03-15</publication_date>
      <categories>
        <category>devops</category>
        <category>containers</category>
      </categories>
    </item>
    <item>
      <id>a1</id>
      <headline>Clean Architecture in Go</headline>
      <type>article</type>
      <stats>
        <reading_time>8</reading_time>
        <reactions>450</reactions>
        <comments>25</comments>
      </stats>
      <publication_date>2024-03-14</publication_date>
      <categories>
        <category>programming</category>
        <category>architecture</category>
      </categories>
    </item>
  </items>
  <meta>
    <total_count>75</total_count>
    <current_page>1</current_page>
    <items_per_page>10</items_per_page>
  </meta>
</feed>`)

	contents, err := Parse(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(contents) != 2 {
		t.Fatalf("expected 2 contents, got %d", len(contents))
	}

	video := contents[0]
	if video.ExternalID != "v1" {
		t.Errorf("ExternalID = %q, want %q", video.ExternalID, "v1")
	}
	if video.Provider != "provider2" {
		t.Errorf("Provider = %q, want %q", video.Provider, "provider2")
	}
	if video.Type != domain.ContentTypeVideo {
		t.Errorf("Type = %q, want %q", video.Type, domain.ContentTypeVideo)
	}
	if video.Metrics.Views != 22000 || video.Metrics.Likes != 1800 {
		t.Errorf("Metrics = %+v, want views=22000 likes=1800", video.Metrics)
	}

	wantDate := time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)
	if !video.PublishedAt.Equal(wantDate) {
		t.Errorf("PublishedAt = %v, want %v", video.PublishedAt, wantDate)
	}
	if len(video.Tags) != 2 || video.Tags[0] != "devops" {
		t.Errorf("Tags = %v", video.Tags)
	}
	if len(video.RawMetrics) == 0 {
		t.Error("RawMetrics should not be empty")
	}

	article := contents[1]
	if article.Type != domain.ContentTypeText {
		t.Errorf("Type = %q, want %q", article.Type, domain.ContentTypeText)
	}
	if article.Metrics.ReadingTime != 8 || article.Metrics.Reactions != 450 {
		t.Errorf("Metrics = %+v, want reading_time=8 reactions=450", article.Metrics)
	}
}

func TestParse_PartialFailure(t *testing.T) {
	data := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<feed>
  <items>
    <item>
      <id>x1</id>
      <headline>Mystery</headline>
      <type>podcast</type>
      <stats></stats>
      <publication_date>2024-01-01</publication_date>
      <categories></categories>
    </item>
    <item>
      <id>v9</id>
      <headline>Valid Video</headline>
      <type>video</type>
      <stats>
        <views>100</views>
        <likes>10</likes>
      </stats>
      <publication_date>2024-01-01</publication_date>
      <categories></categories>
    </item>
  </items>
</feed>`)

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
