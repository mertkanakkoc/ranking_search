package ingest

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"github.com/mertkanakkoc/ranking_search/internal/provider"
	"github.com/mertkanakkoc/ranking_search/internal/repository"
	"github.com/mertkanakkoc/ranking_search/internal/scoring"
)

type Service struct {
	providerRepo repository.ProviderRepository
	contentRepo  repository.ContentRepository
	interval     time.Duration
	now          func() time.Time

	mu       sync.Mutex
	limiters map[string]*rate.Limiter
}

func NewService(providerRepo repository.ProviderRepository, contentRepo repository.ContentRepository, interval time.Duration) *Service {
	return &Service{
		providerRepo: providerRepo,
		contentRepo:  contentRepo,
		interval:     interval,
		now:          time.Now,
		limiters:     make(map[string]*rate.Limiter),
	}
}

func (s *Service) Run(ctx context.Context) {
	s.runOnce(ctx)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.runOnce(ctx)
		}
	}
}

func (s *Service) runOnce(ctx context.Context) {
	configs, err := s.providerRepo.ListActive(ctx)
	if err != nil {
		slog.Error("ingest: list active providers", "error", err)
		return
	}

	var wg sync.WaitGroup
	for _, cfg := range configs {
		wg.Add(1)
		go func(cfg provider.ProviderConfig) {
			defer wg.Done()
			s.ingestProvider(ctx, cfg)
		}(cfg)
	}
	wg.Wait()
}

func (s *Service) ingestProvider(ctx context.Context, cfg provider.ProviderConfig) {
	limiter := s.limiterFor(cfg)
	if err := limiter.Wait(ctx); err != nil {
		slog.Warn("ingest: rate limiter wait", "provider", cfg.Name, "error", err)
		return
	}

	p, err := provider.Build(cfg)
	if err != nil {
		slog.Error("ingest: build provider", "provider", cfg.Name, "error", err)
		s.reportStatus(ctx, cfg.Name, err)
		return
	}

	contents, fetchErr := p.Fetch(ctx)
	if fetchErr != nil {
		slog.Warn("ingest: fetch returned errors", "provider", cfg.Name, "error", fetchErr)
	}

	now := s.now()
	for _, c := range contents {
		score, err := scoring.Calculate(c, now)
		if err != nil {
			slog.Error("ingest: scoring content", "provider", cfg.Name, "content", c.UniqueKey(), "error", err)
			continue
		}
		c.FinalScore = score

		if err := s.contentRepo.Upsert(ctx, c); err != nil {
			slog.Error("ingest: upsert content", "provider", cfg.Name, "content", c.UniqueKey(), "error", err)
		}
	}

	s.reportStatus(ctx, cfg.Name, fetchErr)
}

func (s *Service) limiterFor(cfg provider.ProviderConfig) *rate.Limiter {
	s.mu.Lock()
	defer s.mu.Unlock()

	l, ok := s.limiters[cfg.Name]
	if !ok {
		l = rate.NewLimiter(rate.Limit(cfg.RateLimitRPS), 1)
		s.limiters[cfg.Name] = l
	}
	return l
}

func (s *Service) reportStatus(ctx context.Context, name string, fetchErr error) {
	update := repository.ProviderStatusUpdate{Name: name, Status: "ok"}
	if fetchErr != nil {
		update.Status = "error"
		update.Error = fetchErr.Error()
	}
	if err := s.providerRepo.UpdateStatus(ctx, update); err != nil {
		slog.Error("ingest: update provider status", "provider", name, "error", err)
	}
}
