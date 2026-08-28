package provider

import (
	"fmt"
	"net/http"
	"time"

	"github.com/mertkanakkoc/ranking_search/internal/provider/jsonprovider"
	"github.com/mertkanakkoc/ranking_search/internal/provider/xmlprovider"
)

func Build(cfg ProviderConfig) (Provider, error) {
	client := &http.Client{
		Timeout: time.Duration(cfg.TimeoutMS) * time.Millisecond,
	}

	switch cfg.Format {
	case "json":
		return jsonprovider.New(cfg.BaseURL, client), nil
	case "xml":
		return xmlprovider.New(cfg.BaseURL, client), nil
	default:
		return nil, fmt.Errorf("provider: unknown format %q for provider %q", cfg.Format, cfg.Name)
	}
}
