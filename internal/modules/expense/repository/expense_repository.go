package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/UmedjonQurbonov/CRM/internal/modules/expense/domain"
	"github.com/UmedjonQurbonov/CRM/internal/modules/expense/usecase"
)

// PostgresExpenseRepository implements usecase.ExpenseRepository using PostgreSQL pgxpool.
type PostgresExpenseRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresExpenseRepository initializes a new PostgresExpenseRepository.
func NewPostgresExpenseRepository(pool *pgxpool.Pool) *PostgresExpenseRepository {
	return &PostgresExpenseRepository{pool: pool}
}

// Create inserts a new expense record into PostgreSQL.
func (r *PostgresExpenseRepository) Create(ctx context.Context, expense *domain.Expense) (*domain.Expense, error) {
	if expense.ID == uuid.Nil {
		expense.ID = uuid.New()
	}
	now := time.Now().UTC()
	if expense.CreatedAt.IsZero() {
		expense.CreatedAt = now
	}
	if expense.UpdatedAt.IsZero() {
		expense.UpdatedAt = now
	}

	query := `
		INSERT INTO expenses (
			id, created_by, category, amount, comment, expense_date, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at
	`

	err := r.pool.QueryRow(ctx, query,
		expense.ID,
		expense.CreatedBy,
		expense.Category,
		expense.Amount,
		expense.Comment,
		expense.ExpenseDate,
		expense.CreatedAt,
		expense.UpdatedAt,
	).Scan(&expense.ID, &expense.CreatedAt, &expense.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to insert expense: %w", err)
	}

	return expense, nil
}

// GetByID retrieves a single expense record by UUID.
func (r *PostgresExpenseRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Expense, error) {
	query := `
		SELECT id, created_by, category, amount, comment, expense_date, created_at, updated_at
		FROM expenses
		WHERE id = $1
	`

	var exp domain.Expense
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&exp.ID,
		&exp.CreatedBy,
		&exp.Category,
		&exp.Amount,
		&exp.Comment,
		&exp.ExpenseDate,
		&exp.CreatedAt,
		&exp.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrExpenseNotFound
		}
		return nil, fmt.Errorf("failed to fetch expense by id: %w", err)
	}

	return &exp, nil
}

// List queries expenses based on filters and pagination.
func (r *PostgresExpenseRepository) List(ctx context.Context, filter usecase.ExpenseFilter) ([]*domain.Expense, int, error) {
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

	conditions := []string{"1=1"}
	args := make([]any, 0)
	argIdx := 1

	if filter.Category != "" {
		conditions = append(conditions, fmt.Sprintf("LOWER(category) LIKE LOWER($%d)", argIdx))
		args = append(args, "%"+strings.TrimSpace(filter.Category)+"%")
		argIdx++
	}

	if filter.FromDate != nil {
		conditions = append(conditions, fmt.Sprintf("expense_date >= $%d", argIdx))
		args = append(args, *filter.FromDate)
		argIdx++
	}

	if filter.ToDate != nil {
		conditions = append(conditions, fmt.Sprintf("expense_date <= $%d", argIdx))
		args = append(args, *filter.ToDate)
		argIdx++
	}

	whereClause := strings.Join(conditions, " AND ")

	query := fmt.Sprintf(`
		SELECT id, created_by, category, amount, comment, expense_date, created_at, updated_at,
		       COUNT(*) OVER() AS total_count
		FROM expenses
		WHERE %s
		ORDER BY expense_date DESC, created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIdx, argIdx+1)

	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list expenses: %w", err)
	}
	defer rows.Close()

	expenses := make([]*domain.Expense, 0)
	total := 0

	for rows.Next() {
		var exp domain.Expense
		err := rows.Scan(
			&exp.ID,
			&exp.CreatedBy,
			&exp.Category,
			&exp.Amount,
			&exp.Comment,
			&exp.ExpenseDate,
			&exp.CreatedAt,
			&exp.UpdatedAt,
			&total,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan expense row: %w", err)
		}
		expenses = append(expenses, &exp)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error reading expense rows: %w", err)
	}

	return expenses, total, nil
}

// Delete physically deletes an expense record.
func (r *PostgresExpenseRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM expenses WHERE id = $1`

	tag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete expense: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return domain.ErrExpenseNotFound
	}

	return nil
}
