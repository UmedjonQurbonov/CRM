package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/UmedjonQurbonov/CRM/internal/modules/expense/domain"
)

// PostgresExpenseCategoryRepository implements expense category CRUD via PostgreSQL.
type PostgresExpenseCategoryRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresExpenseCategoryRepository initializes a new expense category repository.
func NewPostgresExpenseCategoryRepository(pool *pgxpool.Pool) *PostgresExpenseCategoryRepository {
	return &PostgresExpenseCategoryRepository{pool: pool}
}

// ListCategories returns all expense categories ordered by name.
func (r *PostgresExpenseCategoryRepository) ListCategories(ctx context.Context) ([]*domain.ExpenseCategory, error) {
	query := `SELECT id, name, created_at, updated_at FROM expense_categories ORDER BY name ASC`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list expense categories: %w", err)
	}
	defer rows.Close()

	categories := make([]*domain.ExpenseCategory, 0)
	for rows.Next() {
		var cat domain.ExpenseCategory
		if err := rows.Scan(&cat.ID, &cat.Name, &cat.CreatedAt, &cat.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan expense category: %w", err)
		}
		categories = append(categories, &cat)
	}

	return categories, rows.Err()
}

// CreateCategory inserts a new expense category.
func (r *PostgresExpenseCategoryRepository) CreateCategory(ctx context.Context, name string) (*domain.ExpenseCategory, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return nil, domain.ErrEmptyCategory
	}

	id := uuid.New()
	now := time.Now().UTC()

	query := `
		INSERT INTO expense_categories (id, name, created_at, updated_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id, name, created_at, updated_at
	`

	var cat domain.ExpenseCategory
	err := r.pool.QueryRow(ctx, query, id, trimmed, now, now).Scan(
		&cat.ID, &cat.Name, &cat.CreatedAt, &cat.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, domain.ErrDuplicateCategory
		}
		return nil, fmt.Errorf("failed to create expense category: %w", err)
	}

	return &cat, nil
}

// DeleteCategory removes an expense category by UUID.
func (r *PostgresExpenseCategoryRepository) DeleteCategory(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM expense_categories WHERE id = $1`

	tag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete expense category: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return domain.ErrCategoryNotFound
	}

	return nil
}

// GetByID retrieves an expense category by UUID.
func (r *PostgresExpenseCategoryRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.ExpenseCategory, error) {
	query := `SELECT id, name, created_at, updated_at FROM expense_categories WHERE id = $1`

	var cat domain.ExpenseCategory
	err := r.pool.QueryRow(ctx, query, id).Scan(&cat.ID, &cat.Name, &cat.CreatedAt, &cat.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrCategoryNotFound
		}
		return nil, fmt.Errorf("failed to get expense category by id: %w", err)
	}

	return &cat, nil
}
