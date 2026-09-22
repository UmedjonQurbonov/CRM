package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/UmedjonQurbonov/CRM/internal/modules/order/domain"
	"github.com/UmedjonQurbonov/CRM/internal/modules/order/usecase"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

type PostgresOrderRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresOrderRepository(pool *pgxpool.Pool) *PostgresOrderRepository {
	return &PostgresOrderRepository{pool: pool}
}

func (r *PostgresOrderRepository) CreateOrderTx(
	ctx context.Context,
	sellerID uuid.UUID,
	paymentMethod string,
	commissionRate decimal.Decimal,
	cartItems []usecase.CartItemInput,
) (*domain.Order, error) {
	if len(cartItems) == 0 {
		return nil, domain.ErrEmptyCart
	}

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return nil, fmt.Errorf("failed to begin checkout transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	orderID := uuid.New()
	var orderItems []domain.OrderItem
	totalAmount := decimal.Zero
	totalCost := decimal.Zero

	// 1. Atomic decrement of each product stock with quantity verification
	updateStockQuery := `
		UPDATE products
		SET stock_quantity = stock_quantity - $1, updated_at = NOW()
		WHERE id = $2 AND stock_quantity >= $1 AND is_active = TRUE
		RETURNING id, name, cost_price, selling_price
	`

	for _, item := range cartItems {
		var (
			prodID       uuid.UUID
			prodName     string
			costPrice    decimal.Decimal
			sellingPrice decimal.Decimal
		)

		err := tx.QueryRow(ctx, updateStockQuery, item.Quantity, item.ProductID).Scan(
			&prodID,
			&prodName,
			&costPrice,
			&sellingPrice,
		)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, domain.ErrInsufficientStock
			}
			return nil, fmt.Errorf("failed to deduct stock for product %s: %w", item.ProductID, err)
		}

		qtyDec := decimal.NewFromInt(int64(item.Quantity))
		subtotal := sellingPrice.Mul(qtyDec)
		totalAmount = totalAmount.Add(subtotal)
		totalCost = totalCost.Add(costPrice.Mul(qtyDec))

		orderItems = append(orderItems, domain.OrderItem{
			ID:            uuid.New(),
			OrderID:       orderID,
			ProductID:     prodID,
			ProductName:   prodName,
			Quantity:      item.Quantity,
			UnitPrice:     sellingPrice,
			UnitCostPrice: costPrice,
			Subtotal:      subtotal,
		})
	}

	// 2. Calculate commission earned
	commissionEarned := totalAmount.Mul(commissionRate).Div(decimal.NewFromInt(100)).Round(2)

	now := time.Now().UTC()
	order := &domain.Order{
		ID:                     orderID,
		SellerID:               sellerID,
		TotalAmount:            totalAmount,
		TotalCost:              totalCost,
		CommissionRateSnapshot: commissionRate,
		CommissionEarned:       commissionEarned,
		PaymentMethod:          paymentMethod,
		Status:                 domain.StatusCompleted,
		Items:                  orderItems,
		CreatedAt:              now,
		UpdatedAt:              now,
	}

	// 3. Insert order record
	insertOrderQuery := `
		INSERT INTO orders (
			id, seller_id, total_amount, total_cost,
			commission_rate_snapshot, commission_earned,
			payment_method, status, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING created_at, updated_at
	`

	err = tx.QueryRow(ctx, insertOrderQuery,
		order.ID,
		order.SellerID,
		order.TotalAmount,
		order.TotalCost,
		order.CommissionRateSnapshot,
		order.CommissionEarned,
		order.PaymentMethod,
		order.Status,
		order.CreatedAt,
		order.UpdatedAt,
	).Scan(&order.CreatedAt, &order.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to insert order record: %w", err)
	}

	// 4. Batch insert order items
	insertItemQuery := `
		INSERT INTO order_items (
			id, order_id, product_id, quantity, unit_price, unit_cost_price, subtotal
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	for _, oi := range order.Items {
		_, err := tx.Exec(ctx, insertItemQuery,
			oi.ID,
			oi.OrderID,
			oi.ProductID,
			oi.Quantity,
			oi.UnitPrice,
			oi.UnitCostPrice,
			oi.Subtotal,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to insert order item: %w", err)
		}
	}

	// 5. Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit checkout transaction: %w", err)
	}

	return order, nil
}

func (r *PostgresOrderRepository) RefundOrderTx(ctx context.Context, orderID uuid.UUID) (*domain.Order, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return nil, fmt.Errorf("failed to begin refund transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	// 1. Lock order for update
	lockOrderQuery := `
		SELECT id, seller_id, total_amount, total_cost,
		       commission_rate_snapshot, commission_earned,
		       payment_method, status, created_at, updated_at
		FROM orders
		WHERE id = $1
		FOR UPDATE
	`

	var o domain.Order
	err = tx.QueryRow(ctx, lockOrderQuery, orderID).Scan(
		&o.ID,
		&o.SellerID,
		&o.TotalAmount,
		&o.TotalCost,
		&o.CommissionRateSnapshot,
		&o.CommissionEarned,
		&o.PaymentMethod,
		&o.Status,
		&o.CreatedAt,
		&o.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrOrderNotFound
		}
		return nil, fmt.Errorf("failed to lock order for refund: %w", err)
	}

	if o.Status == domain.StatusRefunded {
		return nil, domain.ErrOrderAlreadyRefunded
	}

	// 2. Fetch order items
	itemsQuery := `
		SELECT oi.id, oi.order_id, oi.product_id, COALESCE(p.name, ''),
		       oi.quantity, oi.unit_price, oi.unit_cost_price, oi.subtotal
		FROM order_items oi
		LEFT JOIN products p ON p.id = oi.product_id
		WHERE oi.order_id = $1
	`

	rows, err := tx.Query(ctx, itemsQuery, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve order items for refund: %w", err)
	}
	defer rows.Close()

	var items []domain.OrderItem
	for rows.Next() {
		var item domain.OrderItem
		if err := rows.Scan(
			&item.ID,
			&item.OrderID,
			&item.ProductID,
			&item.ProductName,
			&item.Quantity,
			&item.UnitPrice,
			&item.UnitCostPrice,
			&item.Subtotal,
		); err != nil {
			return nil, fmt.Errorf("failed to scan item for refund: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during items iteration for refund: %w", err)
	}
	rows.Close()

	// 3. Restore product stock quantities
	restoreStockQuery := `
		UPDATE products
		SET stock_quantity = stock_quantity + $1, updated_at = NOW()
		WHERE id = $2
	`

	for _, item := range items {
		_, err := tx.Exec(ctx, restoreStockQuery, item.Quantity, item.ProductID)
		if err != nil {
			return nil, fmt.Errorf("failed to restore stock for product %s: %w", item.ProductID, err)
		}
	}

	// 4. Update order status to refunded
	updateStatusQuery := `
		UPDATE orders
		SET status = $1, updated_at = NOW()
		WHERE id = $2
		RETURNING updated_at
	`

	err = tx.QueryRow(ctx, updateStatusQuery, domain.StatusRefunded, orderID).Scan(&o.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to mark order as refunded: %w", err)
	}

	o.Status = domain.StatusRefunded
	o.Items = items

	// 5. Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit refund transaction: %w", err)
	}

	return &o, nil
}

func (r *PostgresOrderRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Order, error) {
	orderQuery := `
		SELECT id, seller_id, total_amount, total_cost,
		       commission_rate_snapshot, commission_earned,
		       payment_method, status, created_at, updated_at
		FROM orders
		WHERE id = $1
	`

	var o domain.Order
	err := r.pool.QueryRow(ctx, orderQuery, id).Scan(
		&o.ID,
		&o.SellerID,
		&o.TotalAmount,
		&o.TotalCost,
		&o.CommissionRateSnapshot,
		&o.CommissionEarned,
		&o.PaymentMethod,
		&o.Status,
		&o.CreatedAt,
		&o.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrOrderNotFound
		}
		return nil, fmt.Errorf("failed to query order by id: %w", err)
	}

	// Fetch items
	itemsQuery := `
		SELECT oi.id, oi.order_id, oi.product_id, COALESCE(p.name, ''),
		       oi.quantity, oi.unit_price, oi.unit_cost_price, oi.subtotal
		FROM order_items oi
		LEFT JOIN products p ON p.id = oi.product_id
		WHERE oi.order_id = $1
	`

	rows, err := r.pool.Query(ctx, itemsQuery, id)
	if err != nil {
		return nil, fmt.Errorf("failed to query items for order: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var item domain.OrderItem
		if err := rows.Scan(
			&item.ID,
			&item.OrderID,
			&item.ProductID,
			&item.ProductName,
			&item.Quantity,
			&item.UnitPrice,
			&item.UnitCostPrice,
			&item.Subtotal,
		); err != nil {
			return nil, fmt.Errorf("failed to scan order item: %w", err)
		}
		o.Items = append(o.Items, item)
	}

	return &o, nil
}

func (r *PostgresOrderRepository) List(ctx context.Context, filter usecase.OrderFilter) ([]*domain.Order, int, error) {
	var whereClauses []string
	var args []interface{}
	argIdx := 1

	if filter.SellerID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("seller_id = $%d", argIdx))
		args = append(args, *filter.SellerID)
		argIdx++
	}

	if filter.Status != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, filter.Status)
		argIdx++
	}

	if filter.StartDate != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("created_at >= $%d", argIdx))
		args = append(args, *filter.StartDate)
		argIdx++
	}

	if filter.EndDate != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("created_at <= $%d", argIdx))
		args = append(args, *filter.EndDate)
		argIdx++
	}

	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	// Count query
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM orders %s", whereSQL)
	var totalCount int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&totalCount); err != nil {
		return nil, 0, fmt.Errorf("failed to count orders: %w", err)
	}

	// Data query
	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	} else if limit > 100 {
		limit = 100
	}

	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	dataQuery := fmt.Sprintf(`
		SELECT id, seller_id, total_amount, total_cost,
		       commission_rate_snapshot, commission_earned,
		       payment_method, status, created_at, updated_at
		FROM orders
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereSQL, argIdx, argIdx+1)

	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list orders: %w", err)
	}
	defer rows.Close()

	var orders []*domain.Order
	for rows.Next() {
		var o domain.Order
		if err := rows.Scan(
			&o.ID,
			&o.SellerID,
			&o.TotalAmount,
			&o.TotalCost,
			&o.CommissionRateSnapshot,
			&o.CommissionEarned,
			&o.PaymentMethod,
			&o.Status,
			&o.CreatedAt,
			&o.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan order row: %w", err)
		}
		orders = append(orders, &o)
	}
	rows.Close()

	// Hydrate items for each order in list
	for _, o := range orders {
		itemsQuery := `
			SELECT oi.id, oi.order_id, oi.product_id, COALESCE(p.name, ''),
			       oi.quantity, oi.unit_price, oi.unit_cost_price, oi.subtotal
			FROM order_items oi
			LEFT JOIN products p ON p.id = oi.product_id
			WHERE oi.order_id = $1
		`
		iRows, err := r.pool.Query(ctx, itemsQuery, o.ID)
		if err == nil {
			for iRows.Next() {
				var item domain.OrderItem
				if err := iRows.Scan(
					&item.ID,
					&item.OrderID,
					&item.ProductID,
					&item.ProductName,
					&item.Quantity,
					&item.UnitPrice,
					&item.UnitCostPrice,
					&item.Subtotal,
				); err == nil {
					o.Items = append(o.Items, item)
				}
			}
			iRows.Close()
		}
	}

	return orders, totalCount, nil
}
