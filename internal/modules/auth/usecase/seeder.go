package usecase

import (
	"context"
	"fmt"
	"log"

	"github.com/UmedjonQurbonov/CRM/internal/modules/auth/domain"
	"github.com/UmedjonQurbonov/CRM/internal/platform/hasher"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// SeedDefaultOwner checks if the users table is empty; if so, provisions the initial Owner account.
func SeedDefaultOwner(
	ctx context.Context,
	userRepo UserRepository,
	pwdHasher hasher.PasswordHasher,
	adminPhone string,
	adminPassword string,
) error {
	count, err := userRepo.Count(ctx)
	if err != nil {
		return fmt.Errorf("failed to check existing user count: %w", err)
	}

	if count > 0 {
		return nil
	}

	log.Printf("No existing users found. Seeding initial Owner account (%s)...", adminPhone)

	passwordHash, err := pwdHasher.Hash(adminPassword)
	if err != nil {
		return fmt.Errorf("failed to hash default admin password: %w", err)
	}

	owner := &domain.User{
		ID:             uuid.New(),
		Name:           "Default Owner",
		Phone:          adminPhone,
		PasswordHash:   passwordHash,
		Role:           domain.RoleOwner,
		CommissionRate: decimal.Zero,
	}

	if err := userRepo.Create(ctx, owner); err != nil {
		return fmt.Errorf("failed to seed default owner: %w", err)
	}

	log.Printf("Initial Owner account seeded successfully [Phone: %s].", adminPhone)
	return nil
}
