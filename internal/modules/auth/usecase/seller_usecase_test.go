package usecase

import (
	"context"
	"testing"

	"github.com/UmedjonQurbonov/CRM/internal/modules/auth/domain"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestSellerUsecase_CreateSeller_Success(t *testing.T) {
	ctx := context.Background()
	userRepo := newMockUserRepo()
	hasher := &mockPasswordHasher{}

	usecase := NewSellerUsecase(userRepo, hasher)

	dto := CreateSellerDTO{
		Name:           "Ali Valiyev",
		Phone:          "+992901234567",
		Password:       "Pass123!",
		CommissionRate: decimal.NewFromFloat(5.50),
	}

	seller, err := usecase.CreateSeller(ctx, dto)
	if err != nil {
		t.Fatalf("expected seller created, got error: %v", err)
	}

	if seller.Role != domain.RoleSeller {
		t.Errorf("expected role 'seller', got %s", seller.Role)
	}
	if !seller.CommissionRate.Equal(decimal.NewFromFloat(5.50)) {
		t.Errorf("expected commission 5.50, got %s", seller.CommissionRate)
	}
}

func TestSellerUsecase_CreateSeller_InvalidCommission(t *testing.T) {
	ctx := context.Background()
	userRepo := newMockUserRepo()
	hasher := &mockPasswordHasher{}

	usecase := NewSellerUsecase(userRepo, hasher)

	dto := CreateSellerDTO{
		Name:           "Ali Valiyev",
		Phone:          "+992901234567",
		Password:       "Pass123!",
		CommissionRate: decimal.NewFromFloat(-1.0),
	}

	_, err := usecase.CreateSeller(ctx, dto)
	if err != domain.ErrInvalidCommissionRate {
		t.Errorf("expected ErrInvalidCommissionRate, got %v", err)
	}
}

func TestSellerUsecase_UpdateCommission_Success(t *testing.T) {
	ctx := context.Background()
	userRepo := newMockUserRepo()
	hasher := &mockPasswordHasher{}

	seller := &domain.User{
		ID:             uuid.New(),
		Name:           "Seller",
		Phone:          "+992901111111",
		Role:           domain.RoleSeller,
		CommissionRate: decimal.NewFromFloat(5.00),
	}
	_ = userRepo.Create(ctx, seller)

	usecase := NewSellerUsecase(userRepo, hasher)

	newRate := decimal.NewFromFloat(8.50)
	err := usecase.UpdateCommission(ctx, seller.ID, newRate)
	if err != nil {
		t.Fatalf("expected commission update success, got: %v", err)
	}

	updated, _ := userRepo.GetByID(ctx, seller.ID)
	if !updated.CommissionRate.Equal(newRate) {
		t.Errorf("expected rate %s, got %s", newRate, updated.CommissionRate)
	}
}

func TestSellerUsecase_UpdateCommission_NotFound(t *testing.T) {
	ctx := context.Background()
	userRepo := newMockUserRepo()
	hasher := &mockPasswordHasher{}

	usecase := NewSellerUsecase(userRepo, hasher)

	err := usecase.UpdateCommission(ctx, uuid.New(), decimal.NewFromFloat(10.0))
	if err != domain.ErrUserNotFound {
		t.Errorf("expected ErrUserNotFound, got %v", err)
	}
}
