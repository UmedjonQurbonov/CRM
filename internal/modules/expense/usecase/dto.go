package usecase

import (
	"github.com/UmedjonQurbonov/CRM/internal/modules/expense/domain"
)

// CreateExpenseDTO represents the payload for registering a new expense.
type CreateExpenseDTO struct {
	Category    string  `json:"category" example:"utilities"`
	Amount      string  `json:"amount" example:"450.00"`
	Comment     *string `json:"comment,omitempty" example:"Monthly electricity bill for store"`
	ExpenseDate string  `json:"expense_date,omitempty" example:"2026-09-22"`
}

// ExpenseResponseDTO represents formatted expense data for API responses.
type ExpenseResponseDTO struct {
	ID          string  `json:"id" example:"497f6eca-6276-4993-bfeb-53cbbbba6f08"`
	Category    string  `json:"category" example:"utilities"`
	Amount      string  `json:"amount" example:"450.00"`
	Comment     *string `json:"comment,omitempty" example:"Monthly electricity bill for store"`
	ExpenseDate string  `json:"expense_date" example:"2026-09-22"`
	CreatedBy   string  `json:"created_by" example:"9b1deb4d-3b7d-4bad-9bdd-2b0d7b3dcb6d"`
	CreatedAt   string  `json:"created_at" example:"2026-09-22T10:00:00Z"`
}

// ExpenseListResponseDTO represents paginated expense list response.
type ExpenseListResponseDTO struct {
	Items  []ExpenseResponseDTO `json:"items"`
	Total  int                  `json:"total" example:"42"`
	Limit  int                  `json:"limit" example:"20"`
	Offset int                  `json:"offset" example:"0"`
}

// ToExpenseResponseDTO converts a domain Expense into ExpenseResponseDTO.
func ToExpenseResponseDTO(e *domain.Expense) ExpenseResponseDTO {
	return ExpenseResponseDTO{
		ID:          e.ID.String(),
		Category:    e.Category,
		Amount:      e.Amount.StringFixed(2),
		Comment:     e.Comment,
		ExpenseDate: e.ExpenseDate.Format("2006-01-02"),
		CreatedBy:   e.CreatedBy.String(),
		CreatedAt:   e.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
