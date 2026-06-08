package stats

import (
	"context"

	"github.com/google/uuid"
	"github.com/touchgal/developer/backend/internal/model"
)

type Store interface {
	Summary(ctx context.Context, userID uuid.UUID, days int, tokenID *uuid.UUID) (model.StatsSummary, error)
	Trend(ctx context.Context, userID uuid.UUID, days int, tokenID *uuid.UUID) ([]model.StatsTrend, error)
	Sources(ctx context.Context, userID uuid.UUID, days int, tokenID *uuid.UUID) ([]model.StatsSource, error)
	Endpoints(ctx context.Context, userID uuid.UUID, days int, tokenID *uuid.UUID) ([]model.StatsEndpoint, error)
}

type Service struct{ store Store }

func NewService(store Store) *Service { return &Service{store: store} }

func NormalizeDays(days int) int {
	if days <= 0 {
		return 30
	}
	if days > 90 {
		return 90
	}
	return days
}

func (s *Service) Summary(ctx context.Context, userID uuid.UUID, days int, tokenID *uuid.UUID) (model.StatsSummary, error) {
	return s.store.Summary(ctx, userID, NormalizeDays(days), tokenID)
}
func (s *Service) Trend(ctx context.Context, userID uuid.UUID, days int, tokenID *uuid.UUID) ([]model.StatsTrend, error) {
	return s.store.Trend(ctx, userID, NormalizeDays(days), tokenID)
}
func (s *Service) Sources(ctx context.Context, userID uuid.UUID, days int, tokenID *uuid.UUID) ([]model.StatsSource, error) {
	return s.store.Sources(ctx, userID, NormalizeDays(days), tokenID)
}
func (s *Service) Endpoints(ctx context.Context, userID uuid.UUID, days int, tokenID *uuid.UUID) ([]model.StatsEndpoint, error) {
	return s.store.Endpoints(ctx, userID, NormalizeDays(days), tokenID)
}
