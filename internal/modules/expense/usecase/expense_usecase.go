package usecase

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/UmedjonQurbonov/CRM/internal/modules/expense/domain"
)

// ExpenseUsecase coordinates operational expense business workflows.
type ExpenseUsecase struct {
	expenseRepo ExpenseRepository
}

// NewExpenseUsecase creates a new ExpenseUsecase instance.
func NewExpenseUsecase(expenseRepo ExpenseRepository) *ExpenseUsecase {
	return &ExpenseUsecase{
		expenseRepo: expenseRepo,
	}
}

// CreateExpense records a new operational expense. Strictly owner-only.
func (uc *ExpenseUsecase) CreateExpense(
	ctx context.Context,
	callerID uuid.UUID,
	callerRole string,
	dto CreateExpenseDTO,
) (*ExpenseResponseDTO, error) {
	if callerRole != "owner" {
		return nil, domain.ErrForbidden
	}

	trimmedCategory := strings.TrimSpace(dto.Category)
	if trimmedCategory == "" {
		return nil, domain.ErrEmptyCategory
	}

	amount, err := decimal.NewFromString(strings.TrimSpace(dto.Amount))
	if err != nil || !amount.GreaterThan(decimal.Zero) {
		return nil, domain.ErrInvalidExpenseAmount
	}

	expenseDate := time.Now().UTC().Truncate(24 * time.Hour)
	if strings.TrimSpace(dto.ExpenseDate) != "" {
		parsedDate, err := time.Parse("2006-01-02", strings.TrimSpace(dto.ExpenseDate))
		if err != nil {
			return nil, domain.ErrValidation
		}
		expenseDate = parsedDate
	}

	expense := &domain.Expense{
		ID:          uuid.New(),
		CreatedBy:   callerID,
		Category:    trimmedCategory,
		Amount:      amount,
		Comment:     dto.Comment,
		ExpenseDate: expenseDate,
	}

	if err := expense.Validate(); err != nil {
		return nil, err
	}

	created, err := uc.expenseRepo.Create(ctx, expense)
	if err != nil {
		return nil, err
	}

	resp := ToExpenseResponseDTO(created)
	return &resp, nil
}

// ListExpenses retrieves operational expenses with filtering. Strictly owner-only.
func (uc *ExpenseUsecase) ListExpenses(
	ctx context.Context,
	callerRole string,
	filter ExpenseFilter,
) ([]ExpenseResponseDTO, int, error) {
	if callerRole != "owner" {
		return nil, 0, domain.ErrForbidden
	}

	expenses, total, err := uc.expenseRepo.List(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	dtos := make([]ExpenseResponseDTO, 0, len(expenses))
	for _, exp := range expenses {
		dtos = append(dtos, ToExpenseResponseDTO(exp))
	}

	return dtos, total, nil
}

// DeleteExpense removes an erroneously registered expense. Strictly owner-only.
func (uc *ExpenseUsecase) DeleteExpense(
	ctx context.Context,
	callerRole string,
	id uuid.UUID,
) error {
	if callerRole != "owner" {
		return domain.ErrForbidden
	}

	return uc.expenseRepo.Delete(ctx, id)
}
