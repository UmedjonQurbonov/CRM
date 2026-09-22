package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/UmedjonQurbonov/CRM/internal/modules/auth/domain"
	"github.com/UmedjonQurbonov/CRM/internal/platform/hasher"
	"github.com/google/uuid"
)

// AuthUsecase handles user authentication, token refresh, and logout operations.
type AuthUsecase struct {
	userRepo     UserRepository
	sessionRepo  SessionRepository
	hasher       hasher.PasswordHasher
	tokenService *TokenService
}

func NewAuthUsecase(
	userRepo UserRepository,
	sessionRepo SessionRepository,
	hasher hasher.PasswordHasher,
	tokenService *TokenService,
) *AuthUsecase {
	return &AuthUsecase{
		userRepo:     userRepo,
		sessionRepo:  sessionRepo,
		hasher:       hasher,
		tokenService: tokenService,
	}
}

// Login authenticates user with phone & password and creates a new session.
func (u *AuthUsecase) Login(ctx context.Context, phone, password, deviceInfo string) (*TokenPair, *domain.User, error) {
	phone = strings.TrimSpace(phone)
	if phone == "" || strings.TrimSpace(password) == "" {
		return nil, nil, domain.ErrInvalidCredentials
	}

	user, err := u.userRepo.GetByPhone(ctx, phone)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, nil, domain.ErrInvalidCredentials
		}
		return nil, nil, err
	}

	if err := u.hasher.Compare(user.PasswordHash, password); err != nil {
		return nil, nil, domain.ErrInvalidCredentials
	}

	accessToken, err := u.tokenService.GenerateAccessToken(user)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := u.tokenService.GenerateRefreshToken()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	tokenHash := u.tokenService.HashToken(refreshToken)
	session := &domain.RefreshSession{
		ID:         uuid.New(),
		UserID:     user.ID,
		TokenHash:  tokenHash,
		DeviceInfo: deviceInfo,
		ExpiresAt:  time.Now().UTC().Add(7 * 24 * time.Hour), // 7 days
		CreatedAt:  time.Now().UTC(),
	}

	if err := u.sessionRepo.CreateSession(ctx, session); err != nil {
		return nil, nil, fmt.Errorf("failed to persist refresh session: %w", err)
	}

	tokens := &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    u.tokenService.AccessDurationSeconds(),
		TokenType:    "Bearer",
	}

	return tokens, user, nil
}

// RefreshToken validates an existing refresh token, rotates it, and issues new tokens.
func (u *AuthUsecase) RefreshToken(ctx context.Context, refreshToken, deviceInfo string) (*TokenPair, error) {
	refreshToken = strings.TrimSpace(refreshToken)
	if refreshToken == "" {
		return nil, domain.ErrUnauthorized
	}

	tokenHash := u.tokenService.HashToken(refreshToken)
	session, err := u.sessionRepo.GetSessionByTokenHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, domain.ErrSessionNotFound) {
			return nil, domain.ErrUnauthorized
		}
		return nil, err
	}

	if session.IsExpired() {
		_ = u.sessionRepo.DeleteSession(ctx, tokenHash)
		return nil, domain.ErrSessionExpired
	}

	user, err := u.userRepo.GetByID(ctx, session.UserID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			_ = u.sessionRepo.DeleteSession(ctx, tokenHash)
			return nil, domain.ErrUnauthorized
		}
		return nil, err
	}

	// Token rotation: delete old refresh token session
	if err := u.sessionRepo.DeleteSession(ctx, tokenHash); err != nil {
		return nil, fmt.Errorf("failed to revoke old session: %w", err)
	}

	// Generate new token pair
	newAccessToken, err := u.tokenService.GenerateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate new access token: %w", err)
	}

	newRefreshToken, err := u.tokenService.GenerateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate new refresh token: %w", err)
	}

	newSession := &domain.RefreshSession{
		ID:         uuid.New(),
		UserID:     user.ID,
		TokenHash:  u.tokenService.HashToken(newRefreshToken),
		DeviceInfo: deviceInfo,
		ExpiresAt:  time.Now().UTC().Add(7 * 24 * time.Hour),
		CreatedAt:  time.Now().UTC(),
	}

	if err := u.sessionRepo.CreateSession(ctx, newSession); err != nil {
		return nil, fmt.Errorf("failed to persist new refresh session: %w", err)
	}

	tokens := &TokenPair{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    u.tokenService.AccessDurationSeconds(),
		TokenType:    "Bearer",
	}

	return tokens, nil
}

// Logout invalidates the provided refresh token session.
func (u *AuthUsecase) Logout(ctx context.Context, refreshToken string) error {
	refreshToken = strings.TrimSpace(refreshToken)
	if refreshToken == "" {
		return nil
	}

	tokenHash := u.tokenService.HashToken(refreshToken)
	return u.sessionRepo.DeleteSession(ctx, tokenHash)
}
