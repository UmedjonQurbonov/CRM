package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/UmedjonQurbonov/CRM/internal/modules/product/domain"
	"github.com/UmedjonQurbonov/CRM/internal/modules/product/usecase"
	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

type PostgresProductRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresProductRepository(pool *pgxpool.Pool) *PostgresProductRepository {
	return &PostgresProductRepository{pool: pool}
}

func (r *PostgresProductRepository) Create(ctx context.Context, p *domain.Product) (*domain.Product, error) {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	now := time.Now().UTC()
	if p.CreatedAt.IsZero() {
		p.CreatedAt = now
	}
	if p.UpdatedAt.IsZero() {
		p.UpdatedAt = now
	}
	p.IsActive = true

	query := `
		INSERT INTO products (
			id, name, sku, qr_code, category, image_url, cost_price, selling_price,
			stock_quantity, min_stock_alert, is_active, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id, created_at, updated_at
	`

	err := r.pool.QueryRow(ctx, query,
		p.ID,
		p.Name,
		p.SKU,
		p.QRCode,
		p.Category,
		p.ImageURL,
		p.CostPrice,
		p.SellingPrice,
		p.StockQuantity,
		p.MinStockAlert,
		p.IsActive,
		p.CreatedAt,
		p.UpdatedAt,
	).Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			if strings.Contains(pgErr.ConstraintName, "sku") {
				return nil, domain.ErrDuplicateSKU
			}
			if strings.Contains(pgErr.ConstraintName, "qr_code") {
				return nil, domain.ErrDuplicateQRCode
			}
		}
		return nil, fmt.Errorf("failed to create product: %w", err)
	}

	return p, nil
}

func (r *PostgresProductRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Product, error) {
	query := `
		SELECT id, name, sku, qr_code, COALESCE(category, ''), COALESCE(image_url, ''), cost_price, selling_price,
		       stock_quantity, min_stock_alert, is_active, created_at, updated_at
		FROM products
		WHERE id = $1 AND is_active = TRUE
	`

	var p domain.Product
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&p.ID,
		&p.Name,
		&p.SKU,
		&p.QRCode,
		&p.Category,
		&p.ImageURL,
		&p.CostPrice,
		&p.SellingPrice,
		&p.StockQuantity,
		&p.MinStockAlert,
		&p.IsActive,
		&p.CreatedAt,
		&p.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrProductNotFound
		}
		return nil, fmt.Errorf("failed to get product by id: %w", err)
	}

	return &p, nil
}

func (r *PostgresProductRepository) GetByQRCode(ctx context.Context, qrCode string) (*domain.Product, error) {
	query := `
		SELECT id, name, sku, qr_code, COALESCE(category, ''), COALESCE(image_url, ''), cost_price, selling_price,
		       stock_quantity, min_stock_alert, is_active, created_at, updated_at
		FROM products
		WHERE qr_code = $1 AND is_active = TRUE
	`

	var p domain.Product
	err := r.pool.QueryRow(ctx, query, qrCode).Scan(
		&p.ID,
		&p.Name,
		&p.SKU,
		&p.QRCode,
		&p.Category,
		&p.ImageURL,
		&p.CostPrice,
		&p.SellingPrice,
		&p.StockQuantity,
		&p.MinStockAlert,
		&p.IsActive,
		&p.CreatedAt,
		&p.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrProductNotFound
		}
		return nil, fmt.Errorf("failed to get product by qr code: %w", err)
	}

	return &p, nil
}

func (r *PostgresProductRepository) List(ctx context.Context, filter usecase.ProductFilter) ([]*domain.Product, int, error) {
	var whereClauses []string
	var args []interface{}
	argIdx := 1

	whereClauses = append(whereClauses, "is_active = TRUE")

	if filter.Search != "" {
		searchTerm := "%" + strings.TrimSpace(filter.Search) + "%"
		whereClauses = append(whereClauses, fmt.Sprintf("(name ILIKE $%d OR sku ILIKE $%d)", argIdx, argIdx))
		args = append(args, searchTerm)
		argIdx++
	}

	if filter.Category != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("category = $%d", argIdx))
		args = append(args, strings.TrimSpace(filter.Category))
		argIdx++
	}

	if filter.LowStock {
		whereClauses = append(whereClauses, "stock_quantity <= min_stock_alert")
	}

	if filter.InStock != nil {
		if *filter.InStock {
			whereClauses = append(whereClauses, "stock_quantity > 0")
		} else {
			whereClauses = append(whereClauses, "stock_quantity = 0")
		}
	}

	whereSQL := strings.Join(whereClauses, " AND ")

	// 1. Total count query
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM products WHERE %s", whereSQL)
	var totalCount int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&totalCount); err != nil {
		return nil, 0, fmt.Errorf("failed to count products: %w", err)
	}

	// 2. Data query with pagination
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
		SELECT id, name, sku, qr_code, COALESCE(category, ''), COALESCE(image_url, ''), cost_price, selling_price,
		       stock_quantity, min_stock_alert, is_active, created_at, updated_at
		FROM products
		WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereSQL, argIdx, argIdx+1)

	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list products: %w", err)
	}
	defer rows.Close()

	var products []*domain.Product
	for rows.Next() {
		var p domain.Product
		if err := rows.Scan(
			&p.ID,
			&p.Name,
			&p.SKU,
			&p.QRCode,
			&p.Category,
			&p.ImageURL,
			&p.CostPrice,
			&p.SellingPrice,
			&p.StockQuantity,
			&p.MinStockAlert,
			&p.IsActive,
			&p.CreatedAt,
			&p.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan product row: %w", err)
		}
		products = append(products, &p)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error during products scan iteration: %w", err)
	}

	return products, totalCount, nil
}

func (r *PostgresProductRepository) Update(ctx context.Context, p *domain.Product) (*domain.Product, error) {
	query := `
		UPDATE products
		SET name = $1, sku = $2, qr_code = $3, category = $4, image_url = $5,
		    cost_price = $6, selling_price = $7, stock_quantity = $8,
		    min_stock_alert = $9, updated_at = CURRENT_TIMESTAMP
		WHERE id = $10 AND is_active = TRUE
		RETURNING updated_at
	`

	err := r.pool.QueryRow(ctx, query,
		p.Name,
		p.SKU,
		p.QRCode,
		p.Category,
		p.ImageURL,
		p.CostPrice,
		p.SellingPrice,
		p.StockQuantity,
		p.MinStockAlert,
		p.ID,
	).Scan(&p.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrProductNotFound
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			if strings.Contains(pgErr.ConstraintName, "sku") {
				return nil, domain.ErrDuplicateSKU
			}
			if strings.Contains(pgErr.ConstraintName, "qr_code") {
				return nil, domain.ErrDuplicateQRCode
			}
		}
		return nil, fmt.Errorf("failed to update product: %w", err)
	}

	return p, nil
}

func (r *PostgresProductRepository) Deactivate(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE products
		SET is_active = FALSE, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND is_active = TRUE
	`

	cmdTag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to deactivate product: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return domain.ErrProductNotFound
	}

	return nil
}

// Restock increments stock quantity and optionally updates cost price.
func (r *PostgresProductRepository) Restock(ctx context.Context, id uuid.UUID, addedQty int, newCostPrice *decimal.Decimal) (*domain.Product, error) {
	var query string
	var args []any

	if newCostPrice != nil && newCostPrice.IsPositive() {
		query = `
			UPDATE products
			SET stock_quantity = stock_quantity + $1,
			    cost_price = $2,
			    updated_at = CURRENT_TIMESTAMP
			WHERE id = $3 AND is_active = TRUE
			RETURNING id, name, sku, qr_code, COALESCE(category, ''), COALESCE(image_url, ''), cost_price, selling_price,
			          stock_quantity, min_stock_alert, is_active, created_at, updated_at
		`
		args = []any{addedQty, *newCostPrice, id}
	} else {
		query = `
			UPDATE products
			SET stock_quantity = stock_quantity + $1,
			    updated_at = CURRENT_TIMESTAMP
			WHERE id = $2 AND is_active = TRUE
			RETURNING id, name, sku, qr_code, COALESCE(category, ''), COALESCE(image_url, ''), cost_price, selling_price,
			          stock_quantity, min_stock_alert, is_active, created_at, updated_at
		`
		args = []any{addedQty, id}
	}

	var p domain.Product
	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&p.ID,
		&p.Name,
		&p.SKU,
		&p.QRCode,
		&p.Category,
		&p.ImageURL,
		&p.CostPrice,
		&p.SellingPrice,
		&p.StockQuantity,
		&p.MinStockAlert,
		&p.IsActive,
		&p.CreatedAt,
		&p.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrProductNotFound
		}
		return nil, fmt.Errorf("failed to restock product: %w", err)
	}

	return &p, nil
}

// UpdateImage updates the image URL of an existing product.
func (r *PostgresProductRepository) UpdateImage(ctx context.Context, id uuid.UUID, imageURL string) error {
	query := `
		UPDATE products
		SET image_url = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2 AND is_active = TRUE
	`
	cmdTag, err := r.pool.Exec(ctx, query, imageURL, id)
	if err != nil {
		return fmt.Errorf("failed to update product image: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return domain.ErrProductNotFound
	}
	return nil
}

// GetTotalStockProfit calculates SUM(stock_quantity * (selling_price - cost_price)) for all active products.
func (r *PostgresProductRepository) GetTotalStockProfit(ctx context.Context) (decimal.Decimal, error) {
	query := `
		SELECT COALESCE(SUM(
			CASE 
				WHEN (selling_price - cost_price) > 0 AND stock_quantity > 0 
				THEN stock_quantity * (selling_price - cost_price) 
				ELSE 0 
			END
		), 0)
		FROM products
		WHERE is_active = TRUE
	`
	var total decimal.Decimal
	err := r.pool.QueryRow(ctx, query).Scan(&total)
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to calculate total stock profit: %w", err)
	}
	return total, nil
}
