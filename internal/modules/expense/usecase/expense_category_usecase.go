package usecase

import (
	"context"
	"time"

	"github.com/UmedjonQurbonov/CRM/internal/modules/expense/domain"
	"github.com/google/uuid"
)

// ExpenseCategoryResponseDTO represents an expense category in API responses.
type ExpenseCategoryResponseDTO struct {
	ID        string    `json:"id" example:"a1b2c3d4-e5f6-7890-abcd-ef1234567890"`
	Name      string    `json:"name" example:"Аренда помещения"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ExpenseCategoryUsecase encapsulates business logic for expense categories.
type ExpenseCategoryUsecase struct {
	repo ExpenseCategoryRepository
}

// NewExpenseCategoryUsecase initializes the expense category usecase.
func NewExpenseCategoryUsecase(repo ExpenseCategoryRepository) *ExpenseCategoryUsecase {
	return &ExpenseCategoryUsecase{repo: repo}
}

// ListCategories returns all expense categories.
func (u *ExpenseCategoryUsecase) ListCategories(ctx context.Context, callerRole string) ([]ExpenseCategoryResponseDTO, error) {
	if callerRole != "owner" {
		return nil, domain.ErrForbidden
	}

	categories, err := u.repo.ListCategories(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]ExpenseCategoryResponseDTO, 0, len(categories))
	for _, cat := range categories {
		result = append(result, ExpenseCategoryResponseDTO{
			ID:        cat.ID.String(),
			Name:      cat.Name,
			CreatedAt: cat.CreatedAt,
			UpdatedAt: cat.UpdatedAt,
		})
	}

	return result, nil
}

// CreateCategory creates a new expense category.
func (u *ExpenseCategoryUsecase) CreateCategory(ctx context.Context, callerRole string, name string) (*ExpenseCategoryResponseDTO, error) {
	if callerRole != "owner" {
		return nil, domain.ErrForbidden
	}

	cat, err := u.repo.CreateCategory(ctx, name)
	if err != nil {
		return nil, err
	}

	return &ExpenseCategoryResponseDTO{
		ID:        cat.ID.String(),
		Name:      cat.Name,
		CreatedAt: cat.CreatedAt,
		UpdatedAt: cat.UpdatedAt,
	}, nil
}

// DeleteCategory deletes an expense category by ID.
func (u *ExpenseCategoryUsecase) DeleteCategory(ctx context.Context, callerRole string, id uuid.UUID) error {
	if callerRole != "owner" {
		return domain.ErrForbidden
	}

	return u.repo.DeleteCategory(ctx, id)
}
