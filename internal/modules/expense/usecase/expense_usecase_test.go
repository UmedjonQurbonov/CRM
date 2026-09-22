package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/UmedjonQurbonov/CRM/internal/modules/expense/domain"
	"github.com/UmedjonQurbonov/CRM/internal/modules/expense/usecase"
)

type mockExpenseRepository struct {
	createFunc  func(ctx context.Context, expense *domain.Expense) (*domain.Expense, error)
	getByIDFunc func(ctx context.Context, id uuid.UUID) (*domain.Expense, error)
	listFunc    func(ctx context.Context, filter usecase.ExpenseFilter) ([]*domain.Expense, int, error)
	deleteFunc  func(ctx context.Context, id uuid.UUID) error
}

func (m *mockExpenseRepository) Create(ctx context.Context, expense *domain.Expense) (*domain.Expense, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, expense)
	}
	return expense, nil
}

func (m *mockExpenseRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Expense, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return nil, domain.ErrExpenseNotFound
}

func (m *mockExpenseRepository) List(ctx context.Context, filter usecase.ExpenseFilter) ([]*domain.Expense, int, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, filter)
	}
	return nil, 0, nil
}

func (m *mockExpenseRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return nil
}

func TestExpenseUsecase_RBACGuard_SellerForbidden(t *testing.T) {
	repo := &mockExpenseRepository{}
	uc := usecase.NewExpenseUsecase(repo)
	ctx := context.Background()
	sellerID := uuid.New()
	sellerRole := "seller"

	// 1. CreateExpense attempt by seller
	_, err := uc.CreateExpense(ctx, sellerID, sellerRole, usecase.CreateExpenseDTO{
		Category: "rent",
		Amount:   "500.00",
	})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden on CreateExpense for seller, got %v", err)
	}

	// 2. ListExpenses attempt by seller
	_, _, err = uc.ListExpenses(ctx, sellerRole, usecase.ExpenseFilter{})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden on ListExpenses for seller, got %v", err)
	}

	// 3. DeleteExpense attempt by seller
	err = uc.DeleteExpense(ctx, sellerRole, uuid.New())
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden on DeleteExpense for seller, got %v", err)
	}
}

func TestExpenseUsecase_Validation_Amount(t *testing.T) {
	repo := &mockExpenseRepository{}
	uc := usecase.NewExpenseUsecase(repo)
	ctx := context.Background()
	ownerID := uuid.New()

	testCases := []struct {
		name   string
		amount string
	}{
		{"negative amount", "-100.00"},
		{"zero amount", "0.00"},
		{"empty amount", ""},
		{"alphabetic amount", "invalid"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := uc.CreateExpense(ctx, ownerID, "owner", usecase.CreateExpenseDTO{
				Category: "utilities",
				Amount:   tc.amount,
			})
			if !errors.Is(err, domain.ErrInvalidExpenseAmount) {
				t.Fatalf("expected ErrInvalidExpenseAmount for amount %q, got %v", tc.amount, err)
			}
		})
	}
}

func TestExpenseUsecase_Validation_Category(t *testing.T) {
	repo := &mockExpenseRepository{}
	uc := usecase.NewExpenseUsecase(repo)
	ctx := context.Background()
	ownerID := uuid.New()

	testCases := []struct {
		name     string
		category string
	}{
		{"empty category", ""},
		{"whitespace category", "    "},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := uc.CreateExpense(ctx, ownerID, "owner", usecase.CreateExpenseDTO{
				Category: tc.category,
				Amount:   "150.00",
			})
			if !errors.Is(err, domain.ErrEmptyCategory) {
				t.Fatalf("expected ErrEmptyCategory for category %q, got %v", tc.category, err)
			}
		})
	}
}

func TestExpenseUsecase_CreateSuccess(t *testing.T) {
	ownerID := uuid.New()
	comment := "Office electricity"
	repo := &mockExpenseRepository{
		createFunc: func(ctx context.Context, exp *domain.Expense) (*domain.Expense, error) {
			if exp.CreatedBy != ownerID {
				t.Errorf("expected created_by %v, got %v", ownerID, exp.CreatedBy)
			}
			if exp.Category != "utilities" {
				t.Errorf("expected category 'utilities', got %v", exp.Category)
			}
			if !exp.Amount.Equal(decimal.NewFromFloat(250.50)) {
				t.Errorf("expected amount 250.50, got %v", exp.Amount)
			}
			if exp.Comment == nil || *exp.Comment != comment {
				t.Errorf("expected comment %q, got %v", comment, exp.Comment)
			}
			if exp.ExpenseDate.Format("2006-01-02") != "2026-09-20" {
				t.Errorf("expected date 2026-09-20, got %v", exp.ExpenseDate)
			}
			exp.CreatedAt = time.Now().UTC()
			return exp, nil
		},
	}

	uc := usecase.NewExpenseUsecase(repo)
	ctx := context.Background()

	res, err := uc.CreateExpense(ctx, ownerID, "owner", usecase.CreateExpenseDTO{
		Category:    "  utilities  ",
		Amount:      "250.50",
		Comment:     &comment,
		ExpenseDate: "2026-09-20",
	})
	if err != nil {
		t.Fatalf("unexpected error on CreateExpense: %v", err)
	}
	if res == nil {
		t.Fatal("expected non-nil response")
	}
	if res.Category != "utilities" {
		t.Errorf("expected category 'utilities', got %q", res.Category)
	}
	if res.Amount != "250.50" {
		t.Errorf("expected amount '250.50', got %q", res.Amount)
	}
	if res.ExpenseDate != "2026-09-20" {
		t.Errorf("expected date '2026-09-20', got %q", res.ExpenseDate)
	}
	if res.CreatedBy != ownerID.String() {
		t.Errorf("expected created_by %q, got %q", ownerID.String(), res.CreatedBy)
	}
}

func TestExpenseUsecase_ListSuccess(t *testing.T) {
	ownerID := uuid.New()
	fromDate, _ := time.Parse("2006-01-02", "2026-09-01")
	toDate, _ := time.Parse("2006-01-02", "2026-09-30")

	repo := &mockExpenseRepository{
		listFunc: func(ctx context.Context, filter usecase.ExpenseFilter) ([]*domain.Expense, int, error) {
			if filter.Category != "rent" {
				t.Errorf("expected filter category 'rent', got %q", filter.Category)
			}
			if filter.FromDate == nil || *filter.FromDate != fromDate {
				t.Errorf("expected fromDate %v, got %v", fromDate, filter.FromDate)
			}
			if filter.ToDate == nil || *filter.ToDate != toDate {
				t.Errorf("expected toDate %v, got %v", toDate, filter.ToDate)
			}
			if filter.Limit != 10 {
				t.Errorf("expected limit 10, got %d", filter.Limit)
			}
			if filter.Offset != 0 {
				t.Errorf("expected offset 0, got %d", filter.Offset)
			}

			return []*domain.Expense{
				{
					ID:          uuid.New(),
					CreatedBy:   ownerID,
					Category:    "rent",
					Amount:      decimal.NewFromFloat(1500.00),
					ExpenseDate: fromDate,
					CreatedAt:   time.Now().UTC(),
				},
			}, 1, nil
		},
	}

	uc := usecase.NewExpenseUsecase(repo)
	ctx := context.Background()

	items, total, err := uc.ListExpenses(ctx, "owner", usecase.ExpenseFilter{
		Category: "rent",
		FromDate: &fromDate,
		ToDate:   &toDate,
		Limit:    10,
		Offset:   0,
	})
	if err != nil {
		t.Fatalf("unexpected error on ListExpenses: %v", err)
	}
	if total != 1 {
		t.Errorf("expected total 1, got %d", total)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].Category != "rent" {
		t.Errorf("expected category 'rent', got %q", items[0].Category)
	}
	if items[0].Amount != "1500.00" {
		t.Errorf("expected amount '1500.00', got %q", items[0].Amount)
	}
}

func TestExpenseUsecase_DeleteSuccessAndNotFound(t *testing.T) {
	existingID := uuid.New()
	missingID := uuid.New()

	repo := &mockExpenseRepository{
		deleteFunc: func(ctx context.Context, id uuid.UUID) error {
			if id == existingID {
				return nil
			}
			return domain.ErrExpenseNotFound
		},
	}

	uc := usecase.NewExpenseUsecase(repo)
	ctx := context.Background()

	// 1. Success delete
	err := uc.DeleteExpense(ctx, "owner", existingID)
	if err != nil {
		t.Fatalf("unexpected error on DeleteExpense: %v", err)
	}

	// 2. Not found delete
	err = uc.DeleteExpense(ctx, "owner", missingID)
	if !errors.Is(err, domain.ErrExpenseNotFound) {
		t.Fatalf("expected ErrExpenseNotFound, got %v", err)
	}
}
