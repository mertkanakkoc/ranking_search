package scoring

import (
	"math"
	"testing"
	"time"

	"github.com/mertkanakkoc/ranking_search/internal/domain"
)

const epsilon = 1e-9

func almostEqual(a, b float64) bool {
	return math.Abs(a-b) < epsilon
}

func TestBaseScore(t *testing.T) {
	tests := []struct {
		name string
		c    domain.Content
		want float64
	}{
		{
			name: "video",
			c: domain.Content{
				Type:    domain.ContentTypeVideo,
				Metrics: domain.Metrics{Views: 15000, Likes: 1200},
			},
			want: 27,
		},
		{
			name: "text",
			c: domain.Content{
				Type:    domain.ContentTypeText,
				Metrics: domain.Metrics{ReadingTime: 8, Reactions: 450},
			},
			want: 17,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := baseScore(tt.c)
			if !almostEqual(got, tt.want) {
				t.Errorf("baseScore() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTypeCoefficient(t *testing.T) {
	tests := []struct {
		name string
		ct   domain.ContentType
		want float64
	}{
		{"video", domain.ContentTypeVideo, 1.5},
		{"text", domain.ContentTypeText, 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := typeCoefficient(tt.ct)
			if !almostEqual(got, tt.want) {
				t.Errorf("typeCoefficient() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFreshnessScore(t *testing.T) {
	now := time.Date(2024, 3, 20, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name        string
		publishedAt time.Time
		want        float64
	}{
		{"within a week", now.Add(-3 * 24 * time.Hour), 5},
		{"exactly a week", now.Add(-7 * 24 * time.Hour), 5},
		{"within a month", now.Add(-20 * 24 * time.Hour), 3},
		{"within three months", now.Add(-60 * 24 * time.Hour), 1},
		{"older than three months", now.Add(-120 * 24 * time.Hour), 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := freshnessScore(tt.publishedAt, now)
			if !almostEqual(got, tt.want) {
				t.Errorf("freshnessScore() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEngagementScore(t *testing.T) {
	tests := []struct {
		name string
		c    domain.Content
		want float64
	}{
		{
			name: "video normal",
			c: domain.Content{
				Type:    domain.ContentTypeVideo,
				Metrics: domain.Metrics{Views: 15000, Likes: 1200},
			},
			want: 0.8,
		},
		{
			name: "video zero views",
			c: domain.Content{
				Type:    domain.ContentTypeVideo,
				Metrics: domain.Metrics{Views: 0, Likes: 0},
			},
			want: 0,
		},
		{
			name: "text normal",
			c: domain.Content{
				Type:    domain.ContentTypeText,
				Metrics: domain.Metrics{ReadingTime: 8, Reactions: 450},
			},
			want: 281.25,
		},
		{
			name: "text zero reading time",
			c: domain.Content{
				Type:    domain.ContentTypeText,
				Metrics: domain.Metrics{ReadingTime: 0, Reactions: 10},
			},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := engagementScore(tt.c)
			if !almostEqual(got, tt.want) {
				t.Errorf("engagementScore() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCalculate(t *testing.T) {
	now := time.Date(2024, 3, 20, 0, 0, 0, 0, time.UTC)

	t.Run("video within a week", func(t *testing.T) {
		c := domain.Content{
			Type:        domain.ContentTypeVideo,
			Metrics:     domain.Metrics{Views: 15000, Likes: 1200},
			PublishedAt: now.Add(-3 * 24 * time.Hour),
		}

		got, err := Calculate(c, now)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		want := 46.3 // (27 * 1.5) + 5 + 0.8
		if !almostEqual(got, want) {
			t.Errorf("Calculate() = %v, want %v", got, want)
		}
	})

	t.Run("text within a month", func(t *testing.T) {
		c := domain.Content{
			Type:        domain.ContentTypeText,
			Metrics:     domain.Metrics{ReadingTime: 8, Reactions: 450},
			PublishedAt: now.Add(-20 * 24 * time.Hour),
		}

		got, err := Calculate(c, now)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		want := 301.25 // (17 * 1.0) + 3 + 281.25
		if !almostEqual(got, want) {
			t.Errorf("Calculate() = %v, want %v", got, want)
		}
	})

	t.Run("unknown content type returns error", func(t *testing.T) {
		c := domain.Content{
			Type: domain.ContentType("unknown"),
		}

		_, err := Calculate(c, now)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
