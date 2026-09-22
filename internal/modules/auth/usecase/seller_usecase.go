package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/UmedjonQurbonov/CRM/internal/modules/auth/domain"
	"github.com/UmedjonQurbonov/CRM/internal/platform/hasher"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// CreateSellerDTO contains input data required to register a new seller.
type CreateSellerDTO struct {
	Name           string
	Phone          string
	Password       string
	CommissionRate decimal.Decimal
}

// SellerUsecase provides business workflows for managing sales personnel.
type SellerUsecase struct {
	userRepo UserRepository
	hasher   hasher.PasswordHasher
}

func NewSellerUsecase(userRepo UserRepository, hasher hasher.PasswordHasher) *SellerUsecase {
	return &SellerUsecase{
		userRepo: userRepo,
		hasher:   hasher,
	}
}

// CreateSeller validates input, hashes password, and creates a new seller.
func (u *SellerUsecase) CreateSeller(ctx context.Context, dto CreateSellerDTO) (*domain.User, error) {
	name := strings.TrimSpace(dto.Name)
	phone := strings.TrimSpace(dto.Phone)
	password := strings.TrimSpace(dto.Password)

	if name == "" || phone == "" || password == "" {
		return nil, domain.ErrValidation
	}

	if dto.CommissionRate.IsNegative() || dto.CommissionRate.GreaterThan(decimal.NewFromInt(100)) {
		return nil, domain.ErrInvalidCommissionRate
	}

	passwordHash, err := u.hasher.Hash(password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash seller password: %w", err)
	}

	seller := &domain.User{
		ID:             uuid.New(),
		Name:           name,
		Phone:          phone,
		PasswordHash:   passwordHash,
		Role:           domain.RoleSeller,
		CommissionRate: dto.CommissionRate,
	}

	if err := seller.Validate(); err != nil {
		return nil, err
	}

	if err := u.userRepo.Create(ctx, seller); err != nil {
		return nil, err
	}

	return seller, nil
}

// ListSellers retrieves all active seller accounts.
func (u *SellerUsecase) ListSellers(ctx context.Context) ([]*domain.User, error) {
	return u.userRepo.ListSellers(ctx)
}

// UpdateCommission updates the commission rate percentage for a seller.
func (u *SellerUsecase) UpdateCommission(ctx context.Context, sellerID uuid.UUID, rate decimal.Decimal) error {
	if sellerID == uuid.Nil {
		return domain.ErrValidation
	}

	if rate.IsNegative() || rate.GreaterThan(decimal.NewFromInt(100)) {
		return domain.ErrInvalidCommissionRate
	}

	return u.userRepo.UpdateCommissionRate(ctx, sellerID, rate)
}
