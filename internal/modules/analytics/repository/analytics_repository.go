package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/UmedjonQurbonov/CRM/internal/modules/analytics/domain"
)

// PostgresAnalyticsRepository implements usecase.AnalyticsRepository using PostgreSQL pgxpool.
type PostgresAnalyticsRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresAnalyticsRepository initializes a new PostgresAnalyticsRepository.
func NewPostgresAnalyticsRepository(pool *pgxpool.Pool) *PostgresAnalyticsRepository {
	return &PostgresAnalyticsRepository{pool: pool}
}

// GetPLSummary computes the Profit & Loss statement for completed orders and expenses in the given range.
func (r *PostgresAnalyticsRepository) GetPLSummary(ctx context.Context, fromDate, toDate time.Time) (*domain.AnalyticsSummary, error) {
	ordersQuery := `
		SELECT 
			COALESCE(SUM(total_amount), 0.00) AS revenue,
			COALESCE(SUM(total_cost), 0.00) AS cost_of_goods_sold,
			COALESCE(SUM(commission_earned), 0.00) AS total_commissions,
			COUNT(*) AS total_orders
		FROM orders
		WHERE status = 'completed' AND created_at >= $1 AND created_at <= $2
	`

	var summary domain.AnalyticsSummary
	err := r.pool.QueryRow(ctx, ordersQuery, fromDate, toDate).Scan(
		&summary.Revenue,
		&summary.CostOfGoodsSold,
		&summary.TotalCommissions,
		&summary.TotalOrders,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate orders for P&L: %w", err)
	}

	expensesQuery := `
		SELECT COALESCE(SUM(amount), 0.00) AS total_expenses
		FROM expenses
		WHERE expense_date >= $1::date AND expense_date <= $2::date
	`

	err = r.pool.QueryRow(ctx, expensesQuery, fromDate, toDate).Scan(&summary.TotalExpenses)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate expenses for P&L: %w", err)
	}

	summary.CalculatePL()
	return &summary, nil
}

// GetSellersRanking returns seller performance metrics with safe revenue share percentage calculation.
func (r *PostgresAnalyticsRepository) GetSellersRanking(ctx context.Context, fromDate, toDate time.Time, limit int) ([]*domain.SellerRanking, error) {
	if limit <= 0 {
		limit = 10
	} else if limit > 100 {
		limit = 100
	}

	// 1. Calculate total store revenue for the period
	totalRevQuery := `
		SELECT COALESCE(SUM(total_amount), 0.00)
		FROM orders
		WHERE status = 'completed' AND created_at >= $1 AND created_at <= $2
	`
	var totalStoreRevenue decimal.Decimal
	err := r.pool.QueryRow(ctx, totalRevQuery, fromDate, toDate).Scan(&totalStoreRevenue)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate total store revenue: %w", err)
	}

	// 2. Query grouped sellers ranking
	rankingQuery := `
		SELECT 
			u.id,
			u.name,
			COUNT(o.id) AS total_orders,
			COALESCE(SUM(o.total_amount), 0.00) AS total_revenue,
			COALESCE(SUM(o.commission_earned), 0.00) AS commission_earned
		FROM orders o
		JOIN users u ON o.seller_id = u.id
		WHERE o.status = 'completed' AND o.created_at >= $1 AND o.created_at <= $2
		GROUP BY u.id, u.name
		ORDER BY total_revenue DESC
		LIMIT $3
	`

	rows, err := r.pool.Query(ctx, rankingQuery, fromDate, toDate, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query sellers ranking: %w", err)
	}
	defer rows.Close()

	rankings := make([]*domain.SellerRanking, 0)
	hundred := decimal.NewFromInt(100)

	for rows.Next() {
		var rank domain.SellerRanking
		err := rows.Scan(
			&rank.SellerID,
			&rank.SellerName,
			&rank.TotalOrders,
			&rank.TotalRevenue,
			&rank.CommissionEarned,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan seller ranking row: %w", err)
		}

		// Safe revenue share calculation (avoid division by zero)
		if totalStoreRevenue.GreaterThan(decimal.Zero) {
			rank.RevenueSharePercentage = rank.TotalRevenue.Div(totalStoreRevenue).Mul(hundred)
		} else {
			rank.RevenueSharePercentage = decimal.Zero
		}

		rankings = append(rankings, &rank)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error reading sellers ranking rows: %w", err)
	}

	return rankings, nil
}

// GetTopProducts returns products ranked by total revenue generated.
func (r *PostgresAnalyticsRepository) GetTopProducts(ctx context.Context, fromDate, toDate time.Time, limit int) ([]*domain.TopProduct, error) {
	if limit <= 0 {
		limit = 10
	} else if limit > 100 {
		limit = 100
	}

	query := `
		SELECT 
			p.id,
			p.name,
			p.sku,
			COALESCE(SUM(oi.quantity), 0) AS total_quantity_sold,
			COALESCE(SUM(oi.subtotal), 0.00) AS total_revenue
		FROM order_items oi
		JOIN orders o ON oi.order_id = o.id
		JOIN products p ON oi.product_id = p.id
		WHERE o.status = 'completed' AND o.created_at >= $1 AND o.created_at <= $2
		GROUP BY p.id, p.name, p.sku
		ORDER BY total_revenue DESC
		LIMIT $3
	`

	rows, err := r.pool.Query(ctx, query, fromDate, toDate, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query top products: %w", err)
	}
	defer rows.Close()

	topProducts := make([]*domain.TopProduct, 0)
	for rows.Next() {
		var prod domain.TopProduct
		err := rows.Scan(
			&prod.ProductID,
			&prod.ProductName,
			&prod.SKU,
			&prod.TotalQuantitySold,
			&prod.TotalRevenue,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan top product row: %w", err)
		}
		topProducts = append(topProducts, &prod)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error reading top products rows: %w", err)
	}

	return topProducts, nil
}

// GetSellerEarnings calculates individual seller earnings for completed sales in the period.
func (r *PostgresAnalyticsRepository) GetSellerEarnings(ctx context.Context, sellerID uuid.UUID, fromDate, toDate time.Time) (*domain.SellerEarningsSummary, error) {
	userQuery := `SELECT name, commission_rate FROM users WHERE id = $1`
	var summary domain.SellerEarningsSummary
	summary.SellerID = sellerID

	err := r.pool.QueryRow(ctx, userQuery, sellerID).Scan(&summary.SellerName, &summary.CommissionRate)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch seller profile: %w", err)
	}

	ordersQuery := `
		SELECT 
			COALESCE(SUM(total_amount), 0.00) AS total_sales_amount,
			COALESCE(SUM(commission_earned), 0.00) AS total_commission_earned,
			COUNT(*) AS orders_count
		FROM orders
		WHERE seller_id = $1 AND status = 'completed' AND created_at >= $2 AND created_at <= $3
	`

	err = r.pool.QueryRow(ctx, ordersQuery, sellerID, fromDate, toDate).Scan(
		&summary.TotalSalesAmount,
		&summary.TotalCommissionEarned,
		&summary.OrdersCount,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate seller earnings: %w", err)
	}

	return &summary, nil
}
