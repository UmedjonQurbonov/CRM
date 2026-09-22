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
			id, name, sku, qr_code, cost_price, selling_price,
			stock_quantity, min_stock_alert, is_active, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, created_at, updated_at
	`

	err := r.pool.QueryRow(ctx, query,
		p.ID,
		p.Name,
		p.SKU,
		p.QRCode,
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
		SELECT id, name, sku, qr_code, cost_price, selling_price,
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
		SELECT id, name, sku, qr_code, cost_price, selling_price,
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

	if filter.LowStock {
		whereClauses = append(whereClauses, "stock_quantity <= min_stock_alert")
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
		SELECT id, name, sku, qr_code, cost_price, selling_price,
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
		SET name = $1, sku = $2, qr_code = $3, cost_price = $4, selling_price = $5,
		    stock_quantity = $6, min_stock_alert = $7, updated_at = CURRENT_TIMESTAMP
		WHERE id = $8 AND is_active = TRUE
		RETURNING updated_at
	`

	err := r.pool.QueryRow(ctx, query,
		p.Name,
		p.SKU,
		p.QRCode,
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
