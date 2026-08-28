package xmlprovider

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/mertkanakkoc/ranking_search/internal/domain"
	"github.com/mertkanakkoc/ranking_search/internal/provider/httpfetch"
)

const providerName = "provider2"

type Adapter struct {
	baseURL string
	client  *http.Client
}

type feed struct {
	XMLName xml.Name `xml:"feed"`
	Items   []item   `xml:"items>item"`
}

type item struct {
	ID              string   `xml:"id"`
	Headline        string   `xml:"headline"`
	Type            string   `xml:"type"`
	Stats           stats    `xml:"stats"`
	PublicationDate string   `xml:"publication_date"`
	Categories      []string `xml:"categories>category"`
	InnerXML        []byte   `xml:",innerxml"`
}

type stats struct {
	Views       int64   `xml:"views"`
	Likes       int64   `xml:"likes"`
	ReadingTime float64 `xml:"reading_time"`
	Reactions   int64   `xml:"reactions"`
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
	var f feed
	if err := xml.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("xmlprovider: parse payload: %w", err)
	}

	var contents []domain.Content
	var errs []error

	for _, it := range f.Items {
		c, err := toContent(it)
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

func toContent(it item) (domain.Content, error) {
	contentType, err := mapType(it.Type)
	if err != nil {
		return domain.Content{}, err
	}

	publishedAt, err := time.Parse("2006-01-02", it.PublicationDate)
	if err != nil {
		return domain.Content{}, fmt.Errorf("xmlprovider: parse date: %w", err)
	}

	return domain.Content{
		ExternalID: it.ID,
		Provider:   providerName,
		Title:      it.Headline,
		Type:       contentType,
		Metrics: domain.Metrics{
			Views:       it.Stats.Views,
			Likes:       it.Stats.Likes,
			ReadingTime: it.Stats.ReadingTime,
			Reactions:   it.Stats.Reactions,
		},
		PublishedAt: publishedAt,
		Tags:        it.Categories,
		RawMetrics:  it.InnerXML,
	}, nil
}

func mapType(raw string) (domain.ContentType, error) {
	switch raw {
	case "video":
		return domain.ContentTypeVideo, nil
	case "article":
		return domain.ContentTypeText, nil
	default:
		return "", fmt.Errorf("xmlprovider: unknown content type %q", raw)
	}
}
