package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/UmedjonQurbonov/CRM/internal/modules/analytics/domain"
)

// AnalyticsRepository defines aggregation queries for store performance and finances.
type AnalyticsRepository interface {
	GetPLSummary(ctx context.Context, fromDate, toDate time.Time) (*domain.AnalyticsSummary, error)
	GetSellersRanking(ctx context.Context, fromDate, toDate time.Time, limit int) ([]*domain.SellerRanking, error)
	GetTopProducts(ctx context.Context, fromDate, toDate time.Time, limit int) ([]*domain.TopProduct, error)
	GetSellerEarnings(ctx context.Context, sellerID uuid.UUID, fromDate, toDate time.Time) (*domain.SellerEarningsSummary, error)
}
