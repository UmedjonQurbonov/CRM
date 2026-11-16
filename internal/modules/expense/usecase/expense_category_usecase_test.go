package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/UmedjonQurbonov/CRM/internal/modules/expense/domain"
	"github.com/google/uuid"
)

type mockExpenseCategoryRepo struct {
	categories map[uuid.UUID]*domain.ExpenseCategory
}

func newMockExpenseCategoryRepo() *mockExpenseCategoryRepo {
	return &mockExpenseCategoryRepo{
		categories: make(map[uuid.UUID]*domain.ExpenseCategory),
	}
}

func (m *mockExpenseCategoryRepo) ListCategories(ctx context.Context) ([]*domain.ExpenseCategory, error) {
	result := make([]*domain.ExpenseCategory, 0, len(m.categories))
	for _, c := range m.categories {
		result = append(result, c)
	}
	return result, nil
}

func (m *mockExpenseCategoryRepo) CreateCategory(ctx context.Context, name string) (*domain.ExpenseCategory, error) {
	for _, c := range m.categories {
		if c.Name == name {
			return nil, domain.ErrDuplicateCategory
		}
	}
	id := uuid.New()
	cat := &domain.ExpenseCategory{
		ID:        id,
		Name:      name,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	m.categories[id] = cat
	return cat, nil
}

func (m *mockExpenseCategoryRepo) DeleteCategory(ctx context.Context, id uuid.UUID) error {
	if _, ok := m.categories[id]; !ok {
		return domain.ErrCategoryNotFound
	}
	delete(m.categories, id)
	return nil
}

func (m *mockExpenseCategoryRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.ExpenseCategory, error) {
	cat, ok := m.categories[id]
	if !ok {
		return nil, domain.ErrCategoryNotFound
	}
	return cat, nil
}

func TestExpenseCategoryUsecase_CRUD(t *testing.T) {
	ctx := context.Background()
	repo := newMockExpenseCategoryRepo()
	uc := NewExpenseCategoryUsecase(repo)

	// Seller cannot access categories
	_, err := uc.ListCategories(ctx, "seller")
	if err != domain.ErrForbidden {
		t.Errorf("expected ErrForbidden for seller, got %v", err)
	}
	_, err = uc.CreateCategory(ctx, "seller", "Test Cat")
	if err != domain.ErrForbidden {
		t.Errorf("expected ErrForbidden for seller, got %v", err)
	}

	// Owner can create category
	cat, err := uc.CreateCategory(ctx, "owner", "Реклама в Instagram")
	if err != nil {
		t.Fatalf("failed to create category: %v", err)
	}
	if cat.Name != "Реклама в Instagram" {
		t.Errorf("expected name 'Реклама в Instagram', got %s", cat.Name)
	}

	// Owner can list categories
	list, err := uc.ListCategories(ctx, "owner")
	if err != nil {
		t.Fatalf("failed to list categories: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("expected 1 category, got %d", len(list))
	}

	// Duplicate category error
	_, err = uc.CreateCategory(ctx, "owner", "Реклама в Instagram")
	if err != domain.ErrDuplicateCategory {
		t.Errorf("expected ErrDuplicateCategory, got %v", err)
	}

	// Owner can delete category
	catID, _ := uuid.Parse(cat.ID)
	err = uc.DeleteCategory(ctx, "owner", catID)
	if err != nil {
		t.Fatalf("failed to delete category: %v", err)
	}

	// Deleting again should return ErrCategoryNotFound
	err = uc.DeleteCategory(ctx, "owner", catID)
	if err != domain.ErrCategoryNotFound {
		t.Errorf("expected ErrCategoryNotFound, got %v", err)
	}
}
