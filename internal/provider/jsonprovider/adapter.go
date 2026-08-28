package jsonprovider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/mertkanakkoc/ranking_search/internal/domain"
	"github.com/mertkanakkoc/ranking_search/internal/provider/httpfetch"
)

const providerName = "provider1"

type payload struct {
	Contents []json.RawMessage `json:"contents"`
}

type item struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Type        string    `json:"type"`
	Metrics     metrics   `json:"metrics"`
	PublishedAt time.Time `json:"published_at"`
	Tags        []string  `json:"tags"`
}

type metrics struct {
	Views int64 `json:"views"`
	Likes int64 `json:"likes"`
}

type Adapter struct {
	baseURL string
	client  *http.Client
}

func New(baseURL string, client *http.Client) *Adapter {
	return &Adapter{baseURL: baseURL, client: client}
}

func (a *Adapter) Name() string { return providerName }

func (a *Adapter) Fetch(ctx context.Context) ([]domain.Content, error) {
	body, err := httpfetch.Get(ctx, a.client, a.baseURL)
	if err != nil {
		return nil, err
	}
	return Parse(body)
}

func Parse(data []byte) ([]domain.Content, error) {
	var p payload
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("jsonprovider: parse payload: %w", err)
	}

	var contents []domain.Content
	var errs []error

	for _, raw := range p.Contents {
		var it item
		if err := json.Unmarshal(raw, &it); err != nil {
			errs = append(errs, fmt.Errorf("jsonprovider: parse item: %w", err))
			continue
		}

		c, err := toContent(it, raw)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		contents = append(contents, c)
	}

	if len(errs) > 0 {
		return contents, errors.Join(errs...)
	}
	return contents, nil
}

func toContent(it item, raw json.RawMessage) (domain.Content, error) {
	contentType, err := mapType(it.Type)
	if err != nil {
		return domain.Content{}, err
	}

	return domain.Content{
		ExternalID: it.ID,
		Provider:   providerName,
		Title:      it.Title,
		Type:       contentType,
		Metrics: domain.Metrics{
			Views: it.Metrics.Views,
			Likes: it.Metrics.Likes,
		},
		PublishedAt: it.PublishedAt,
		Tags:        it.Tags,
		RawMetrics:  raw,
	}, nil
}

func mapType(raw string) (domain.ContentType, error) {
	switch raw {
	case "video":
		return domain.ContentTypeVideo, nil
	case "text":
		return domain.ContentTypeText, nil
	default:
		return "", fmt.Errorf("jsonprovider: unknown content type %q", raw)
	}
}
