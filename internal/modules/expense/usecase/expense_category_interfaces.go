package usecase

import (
	"context"

	"github.com/UmedjonQurbonov/CRM/internal/modules/expense/domain"
	"github.com/google/uuid"
)

// ExpenseCategoryRepository defines persistence operations for expense categories.
type ExpenseCategoryRepository interface {
	ListCategories(ctx context.Context) ([]*domain.ExpenseCategory, error)
	CreateCategory(ctx context.Context, name string) (*domain.ExpenseCategory, error)
	DeleteCategory(ctx context.Context, id uuid.UUID) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.ExpenseCategory, error)
}
