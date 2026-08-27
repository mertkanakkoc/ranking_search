package scoring

import (
	"fmt"
	"time"

	"github.com/mertkanakkoc/ranking_search/internal/domain"
)

func Calculate(c domain.Content, now time.Time) (float64, error) {
	if c.Type != domain.ContentTypeVideo && c.Type != domain.ContentTypeText {
		return 0, fmt.Errorf("scoring: unknown content type %q", c.Type)
	}

	base := baseScore(c)
	coeff := typeCoefficient(c.Type)
	freshness := freshnessScore(c.PublishedAt, now)
	engagement := engagementScore(c)

	return base*coeff + freshness + engagement, nil
}

func baseScore(c domain.Content) float64 {
	if c.Type == domain.ContentTypeVideo {
		return float64(c.Metrics.Views)/1000 + float64(c.Metrics.Likes)/100
	}
	return c.Metrics.ReadingTime + float64(c.Metrics.Reactions)/50
}

func typeCoefficient(t domain.ContentType) float64 {
	if t == domain.ContentTypeVideo {
		return 1.5
	}
	return 1.0
}

func freshnessScore(publishedAt, now time.Time) float64 {
	age := now.Sub(publishedAt)
	switch {
	case age <= 7*24*time.Hour:
		return 5
	case age <= 30*24*time.Hour:
		return 3
	case age <= 90*24*time.Hour:
		return 1
	default:
		return 0
	}
}

func engagementScore(c domain.Content) float64 {
	if c.Type == domain.ContentTypeVideo {
		if c.Metrics.Views == 0 {
			return 0
		}
		return (float64(c.Metrics.Likes) / float64(c.Metrics.Views)) * 10
	}
	if c.Metrics.ReadingTime == 0 {
		return 0
	}
	return (float64(c.Metrics.Reactions) / c.Metrics.ReadingTime) * 5
}
