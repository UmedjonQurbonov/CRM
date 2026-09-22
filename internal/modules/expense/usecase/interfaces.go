package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/UmedjonQurbonov/CRM/internal/modules/expense/domain"
)

// ExpenseFilter encapsulates parameters for filtering and paginating operational expenses.
type ExpenseFilter struct {
	Category string
	FromDate *time.Time
	ToDate   *time.Time
	Limit    int
	Offset   int
}

// ExpenseRepository defines persistence operations for operational expenses.
type ExpenseRepository interface {
	Create(ctx context.Context, expense *domain.Expense) (*domain.Expense, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Expense, error)
	List(ctx context.Context, filter ExpenseFilter) ([]*domain.Expense, int, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
