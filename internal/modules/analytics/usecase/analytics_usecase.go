package usecase

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/UmedjonQurbonov/CRM/internal/modules/analytics/domain"
)

// AnalyticsUsecase coordinates financial reporting, product rankings, and seller earnings.
type AnalyticsUsecase struct {
	analyticsRepo AnalyticsRepository
}

// NewAnalyticsUsecase initializes an AnalyticsUsecase instance.
func NewAnalyticsUsecase(analyticsRepo AnalyticsRepository) *AnalyticsUsecase {
	return &AnalyticsUsecase{
		analyticsRepo: analyticsRepo,
	}
}

// ParseDateRange parses or sets defaults for a reporting period.
// Defaults to the current month: from 1st day (00:00:00) to end of current day (23:59:59).
func ParseDateRange(fromStr, toStr string) (time.Time, time.Time, string, string, error) {
	now := time.Now().UTC()
	var fromDate, toDate time.Time

	if strings.TrimSpace(fromStr) == "" {
		fromDate = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	} else {
		parsed, err := time.Parse("2006-01-02", strings.TrimSpace(fromStr))
		if err != nil {
			return time.Time{}, time.Time{}, "", "", domain.ErrValidation
		}
		fromDate = time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 0, 0, 0, 0, time.UTC)
	}

	if strings.TrimSpace(toStr) == "" {
		toDate = time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 999999999, time.UTC)
	} else {
		parsed, err := time.Parse("2006-01-02", strings.TrimSpace(toStr))
		if err != nil {
			return time.Time{}, time.Time{}, "", "", domain.ErrValidation
		}
		toDate = time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 23, 59, 59, 999999999, time.UTC)
	}

	if fromDate.After(toDate) {
		return time.Time{}, time.Time{}, "", "", domain.ErrInvalidDateRange
	}

	formattedFrom := fromDate.Format("2006-01-02")
	formattedTo := toDate.Format("2006-01-02")

	return fromDate, toDate, formattedFrom, formattedTo, nil
}

// GetSummary retrieves the Profit & Loss summary for the requested period. Strictly owner-only.
func (uc *AnalyticsUsecase) GetSummary(
	ctx context.Context,
	callerRole string,
	fromStr, toStr string,
) (*AnalyticsSummaryDTO, error) {
	if callerRole != "owner" {
		return nil, domain.ErrForbidden
	}

	fromDate, toDate, fFrom, fTo, err := ParseDateRange(fromStr, toStr)
	if err != nil {
		return nil, err
	}

	summary, err := uc.analyticsRepo.GetPLSummary(ctx, fromDate, toDate)
	if err != nil {
		return nil, err
	}

	dto := ToAnalyticsSummaryDTO(summary, fFrom, fTo)
	return &dto, nil
}

// GetSellersRanking retrieves seller performance rankings. Strictly owner-only.
func (uc *AnalyticsUsecase) GetSellersRanking(
	ctx context.Context,
	callerRole string,
	fromStr, toStr string,
	limit int,
) ([]SellerRankingDTO, error) {
	if callerRole != "owner" {
		return nil, domain.ErrForbidden
	}

	fromDate, toDate, _, _, err := ParseDateRange(fromStr, toStr)
	if err != nil {
		return nil, err
	}

	rankings, err := uc.analyticsRepo.GetSellersRanking(ctx, fromDate, toDate, limit)
	if err != nil {
		return nil, err
	}

	dtos := make([]SellerRankingDTO, 0, len(rankings))
	for _, r := range rankings {
		dtos = append(dtos, ToSellerRankingDTO(r))
	}

	return dtos, nil
}

// GetTopProducts retrieves top-selling products by revenue. Strictly owner-only.
func (uc *AnalyticsUsecase) GetTopProducts(
	ctx context.Context,
	callerRole string,
	fromStr, toStr string,
	limit int,
) ([]TopProductDTO, error) {
	if callerRole != "owner" {
		return nil, domain.ErrForbidden
	}

	fromDate, toDate, _, _, err := ParseDateRange(fromStr, toStr)
	if err != nil {
		return nil, err
	}

	topProds, err := uc.analyticsRepo.GetTopProducts(ctx, fromDate, toDate, limit)
	if err != nil {
		return nil, err
	}

	dtos := make([]TopProductDTO, 0, len(topProds))
	for _, p := range topProds {
		dtos = append(dtos, ToTopProductDTO(p))
	}

	return dtos, nil
}

// GetMyEarnings retrieves individual earnings statistics for the calling seller.
func (uc *AnalyticsUsecase) GetMyEarnings(
	ctx context.Context,
	callerID uuid.UUID,
	fromStr, toStr string,
) (*SellerEarningsDTO, error) {
	fromDate, toDate, fFrom, fTo, err := ParseDateRange(fromStr, toStr)
	if err != nil {
		return nil, err
	}

	earnings, err := uc.analyticsRepo.GetSellerEarnings(ctx, callerID, fromDate, toDate)
	if err != nil {
		return nil, err
	}

	dto := ToSellerEarningsDTO(earnings, fFrom, fTo)
	return &dto, nil
}
