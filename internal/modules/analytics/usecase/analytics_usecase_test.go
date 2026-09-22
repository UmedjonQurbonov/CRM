package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/UmedjonQurbonov/CRM/internal/modules/analytics/domain"
	"github.com/UmedjonQurbonov/CRM/internal/modules/analytics/usecase"
)

type mockAnalyticsRepo struct {
	getPLSummaryFunc      func(ctx context.Context, fromDate, toDate time.Time) (*domain.AnalyticsSummary, error)
	getSellersRankingFunc func(ctx context.Context, fromDate, toDate time.Time, limit int) ([]*domain.SellerRanking, error)
	getTopProductsFunc    func(ctx context.Context, fromDate, toDate time.Time, limit int) ([]*domain.TopProduct, error)
	getSellerEarningsFunc func(ctx context.Context, sellerID uuid.UUID, fromDate, toDate time.Time) (*domain.SellerEarningsSummary, error)
}

func (m *mockAnalyticsRepo) GetPLSummary(ctx context.Context, fromDate, toDate time.Time) (*domain.AnalyticsSummary, error) {
	if m.getPLSummaryFunc != nil {
		return m.getPLSummaryFunc(ctx, fromDate, toDate)
	}
	return nil, nil
}

func (m *mockAnalyticsRepo) GetSellersRanking(ctx context.Context, fromDate, toDate time.Time, limit int) ([]*domain.SellerRanking, error) {
	if m.getSellersRankingFunc != nil {
		return m.getSellersRankingFunc(ctx, fromDate, toDate, limit)
	}
	return nil, nil
}

func (m *mockAnalyticsRepo) GetTopProducts(ctx context.Context, fromDate, toDate time.Time, limit int) ([]*domain.TopProduct, error) {
	if m.getTopProductsFunc != nil {
		return m.getTopProductsFunc(ctx, fromDate, toDate, limit)
	}
	return nil, nil
}

func (m *mockAnalyticsRepo) GetSellerEarnings(ctx context.Context, sellerID uuid.UUID, fromDate, toDate time.Time) (*domain.SellerEarningsSummary, error) {
	if m.getSellerEarningsFunc != nil {
		return m.getSellerEarningsFunc(ctx, sellerID, fromDate, toDate)
	}
	return nil, nil
}

func TestAnalytics_PLFormulas(t *testing.T) {
	summary := domain.AnalyticsSummary{
		Revenue:          decimal.NewFromFloat(10000.00),
		CostOfGoodsSold:  decimal.NewFromFloat(6000.00),
		TotalExpenses:    decimal.NewFromFloat(1500.00),
		TotalCommissions: decimal.NewFromFloat(500.00),
		TotalOrders:      25,
	}

	summary.CalculatePL()

	expectedGross := decimal.NewFromFloat(4000.00)
	if !summary.GrossProfit.Equal(expectedGross) {
		t.Fatalf("expected GrossProfit %s, got %s", expectedGross, summary.GrossProfit)
	}

	expectedNet := decimal.NewFromFloat(2000.00)
	if !summary.NetProfit.Equal(expectedNet) {
		t.Fatalf("expected NetProfit %s, got %s", expectedNet, summary.NetProfit)
	}
}

func TestAnalytics_DateRangeValidation(t *testing.T) {
	repo := &mockAnalyticsRepo{}
	uc := usecase.NewAnalyticsUsecase(repo)
	ctx := context.Background()

	// 1. from > to should return ErrInvalidDateRange
	_, err := uc.GetSummary(ctx, "owner", "2026-09-30", "2026-09-01")
	if !errors.Is(err, domain.ErrInvalidDateRange) {
		t.Fatalf("expected ErrInvalidDateRange, got %v", err)
	}

	// 2. malformed date should return ErrValidation
	_, err = uc.GetSummary(ctx, "owner", "invalid-date", "2026-09-30")
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}

	// 3. empty dates should default gracefully to current month
	_, _, fFrom, fTo, err := usecase.ParseDateRange("", "")
	if err != nil {
		t.Fatalf("expected no error on empty dates, got %v", err)
	}
	now := time.Now().UTC()
	expectedMonthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
	if fFrom != expectedMonthStart {
		t.Errorf("expected from date %q, got %q", expectedMonthStart, fFrom)
	}
	if fTo != now.Format("2006-01-02") {
		t.Errorf("expected to date %q, got %q", now.Format("2006-01-02"), fTo)
	}
}

func TestAnalytics_RBAC_SellerForbidden(t *testing.T) {
	repo := &mockAnalyticsRepo{}
	uc := usecase.NewAnalyticsUsecase(repo)
	ctx := context.Background()

	// 1. GetSummary attempt by seller
	_, err := uc.GetSummary(ctx, "seller", "2026-09-01", "2026-09-30")
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden on GetSummary for seller, got %v", err)
	}

	// 2. GetSellersRanking attempt by seller
	_, err = uc.GetSellersRanking(ctx, "seller", "2026-09-01", "2026-09-30", 10)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden on GetSellersRanking for seller, got %v", err)
	}

	// 3. GetTopProducts attempt by seller
	_, err = uc.GetTopProducts(ctx, "seller", "2026-09-01", "2026-09-30", 10)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden on GetTopProducts for seller, got %v", err)
	}
}

func TestAnalytics_GetSummarySuccess(t *testing.T) {
	repo := &mockAnalyticsRepo{
		getPLSummaryFunc: func(ctx context.Context, fromDate, toDate time.Time) (*domain.AnalyticsSummary, error) {
			s := &domain.AnalyticsSummary{
				Revenue:          decimal.NewFromFloat(50000.00),
				CostOfGoodsSold:  decimal.NewFromFloat(30000.00),
				TotalExpenses:    decimal.NewFromFloat(5000.00),
				TotalCommissions: decimal.NewFromFloat(3500.00),
				TotalOrders:      150,
			}
			s.CalculatePL()
			return s, nil
		},
	}

	uc := usecase.NewAnalyticsUsecase(repo)
	ctx := context.Background()

	summary, err := uc.GetSummary(ctx, "owner", "2026-09-01", "2026-09-30")
	if err != nil {
		t.Fatalf("unexpected error on GetSummary: %v", err)
	}

	if summary.Revenue != "50000.00" {
		t.Errorf("expected revenue 50000.00, got %s", summary.Revenue)
	}
	if summary.CostOfGoodsSold != "30000.00" {
		t.Errorf("expected cogs 30000.00, got %s", summary.CostOfGoodsSold)
	}
	if summary.GrossProfit != "20000.00" {
		t.Errorf("expected gross profit 20000.00, got %s", summary.GrossProfit)
	}
	if summary.NetProfit != "11500.00" {
		t.Errorf("expected net profit 11500.00, got %s", summary.NetProfit)
	}
	if summary.TotalOrders != 150 {
		t.Errorf("expected 150 orders, got %d", summary.TotalOrders)
	}
}

func TestAnalytics_GetMyEarningsSuccess(t *testing.T) {
	sellerID := uuid.New()
	repo := &mockAnalyticsRepo{
		getSellerEarningsFunc: func(ctx context.Context, id uuid.UUID, fromDate, toDate time.Time) (*domain.SellerEarningsSummary, error) {
			return &domain.SellerEarningsSummary{
				SellerID:              id,
				SellerName:            "Test Cashier",
				CommissionRate:        decimal.NewFromFloat(8.50),
				TotalSalesAmount:      decimal.NewFromFloat(10000.00),
				TotalCommissionEarned: decimal.NewFromFloat(850.00),
				OrdersCount:           20,
			}, nil
		},
	}

	uc := usecase.NewAnalyticsUsecase(repo)
	ctx := context.Background()

	earnings, err := uc.GetMyEarnings(ctx, sellerID, "2026-09-01", "2026-09-30")
	if err != nil {
		t.Fatalf("unexpected error on GetMyEarnings: %v", err)
	}

	if earnings.SellerName != "Test Cashier" {
		t.Errorf("expected seller name 'Test Cashier', got %q", earnings.SellerName)
	}
	if earnings.CommissionRate != "8.50%" {
		t.Errorf("expected commission rate '8.50%%', got %q", earnings.CommissionRate)
	}
	if earnings.TotalSalesAmount != "10000.00" {
		t.Errorf("expected total sales '10000.00', got %q", earnings.TotalSalesAmount)
	}
	if earnings.TotalCommissionEarned != "850.00" {
		t.Errorf("expected commission earned '850.00', got %q", earnings.TotalCommissionEarned)
	}
	if earnings.OrdersCount != 20 {
		t.Errorf("expected orders count 20, got %d", earnings.OrdersCount)
	}
}
