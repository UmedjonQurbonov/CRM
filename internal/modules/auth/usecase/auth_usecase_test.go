package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/UmedjonQurbonov/CRM/internal/modules/auth/domain"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type mockPasswordHasher struct {
	hashErr    error
	compareErr error
}

func (m *mockPasswordHasher) Hash(password string) (string, error) {
	if m.hashErr != nil {
		return "", m.hashErr
	}
	return "hashed_" + password, nil
}

func (m *mockPasswordHasher) Compare(hash, password string) error {
	if m.compareErr != nil {
		return m.compareErr
	}
	if hash != "hashed_"+password {
		return domain.ErrInvalidCredentials
	}
	return nil
}

type mockUserRepo struct {
	byPhone map[string]*domain.User
	byID    map[uuid.UUID]*domain.User
	count   int64
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{
		byPhone: make(map[string]*domain.User),
		byID:    make(map[uuid.UUID]*domain.User),
	}
}

func (m *mockUserRepo) Create(ctx context.Context, u *domain.User) error {
	if _, exists := m.byPhone[u.Phone]; exists {
		return domain.ErrPhoneAlreadyExists
	}
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	m.byPhone[u.Phone] = u
	m.byID[u.ID] = u
	m.count++
	return nil
}

func (m *mockUserRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	u, ok := m.byID[id]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	return u, nil
}

func (m *mockUserRepo) GetByPhone(ctx context.Context, phone string) (*domain.User, error) {
	u, ok := m.byPhone[phone]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	return u, nil
}

func (m *mockUserRepo) ListSellers(ctx context.Context) ([]*domain.User, error) {
	var result []*domain.User
	for _, u := range m.byID {
		if u.Role == domain.RoleSeller {
			result = append(result, u)
		}
	}
	return result, nil
}

func (m *mockUserRepo) UpdateCommissionRate(ctx context.Context, id uuid.UUID, rate decimal.Decimal) error {
	u, ok := m.byID[id]
	if !ok || u.Role != domain.RoleSeller {
		return domain.ErrUserNotFound
	}
	u.CommissionRate = rate
	return nil
}

func (m *mockUserRepo) Count(ctx context.Context) (int64, error) {
	return m.count, nil
}

type mockSessionRepo struct {
	sessions map[string]*domain.RefreshSession
}

func newMockSessionRepo() *mockSessionRepo {
	return &mockSessionRepo{
		sessions: make(map[string]*domain.RefreshSession),
	}
}

func (m *mockSessionRepo) CreateSession(ctx context.Context, s *domain.RefreshSession) error {
	m.sessions[s.TokenHash] = s
	return nil
}

func (m *mockSessionRepo) GetSessionByTokenHash(ctx context.Context, tokenHash string) (*domain.RefreshSession, error) {
	s, ok := m.sessions[tokenHash]
	if !ok {
		return nil, domain.ErrSessionNotFound
	}
	return s, nil
}

func (m *mockSessionRepo) DeleteSession(ctx context.Context, tokenHash string) error {
	delete(m.sessions, tokenHash)
	return nil
}

func (m *mockSessionRepo) DeleteUserSessions(ctx context.Context, userID uuid.UUID) error {
	for k, v := range m.sessions {
		if v.UserID == userID {
			delete(m.sessions, k)
		}
	}
	return nil
}

func TestAuthUsecase_Login_Success(t *testing.T) {
	ctx := context.Background()
	userRepo := newMockUserRepo()
	sessionRepo := newMockSessionRepo()
	hasher := &mockPasswordHasher{}
	tokenService := NewTokenService("test_secret_key_1234567890123456")

	// Seed user
	owner := &domain.User{
		ID:             uuid.New(),
		Name:           "Owner",
		Phone:          "+992900000000",
		PasswordHash:   "hashed_Secret123!",
		Role:           domain.RoleOwner,
		CommissionRate: decimal.Zero,
	}
	_ = userRepo.Create(ctx, owner)

	usecase := NewAuthUsecase(userRepo, sessionRepo, hasher, tokenService)

	tokens, user, err := usecase.Login(ctx, "+992900000000", "Secret123!", "Chrome/Mac")
	if err != nil {
		t.Fatalf("expected login success, got error: %v", err)
	}

	if tokens.AccessToken == "" {
		t.Error("expected non-empty access token")
	}
	if tokens.RefreshToken == "" {
		t.Error("expected non-empty refresh token")
	}
	if user.Phone != "+992900000000" {
		t.Errorf("expected user phone +992900000000, got %s", user.Phone)
	}
	if len(sessionRepo.sessions) != 1 {
		t.Errorf("expected 1 session created, got %d", len(sessionRepo.sessions))
	}
}

func TestAuthUsecase_Login_InvalidPassword(t *testing.T) {
	ctx := context.Background()
	userRepo := newMockUserRepo()
	sessionRepo := newMockSessionRepo()
	hasher := &mockPasswordHasher{}
	tokenService := NewTokenService("test_secret_key_1234567890123456")

	owner := &domain.User{
		ID:           uuid.New(),
		Name:         "Owner",
		Phone:        "+992900000000",
		PasswordHash: "hashed_CorrectPassword",
		Role:         domain.RoleOwner,
	}
	_ = userRepo.Create(ctx, owner)

	usecase := NewAuthUsecase(userRepo, sessionRepo, hasher, tokenService)

	_, _, err := usecase.Login(ctx, "+992900000000", "WrongPassword", "Chrome")
	if err != domain.ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestAuthUsecase_RefreshToken_Rotation(t *testing.T) {
	ctx := context.Background()
	userRepo := newMockUserRepo()
	sessionRepo := newMockSessionRepo()
	hasher := &mockPasswordHasher{}
	tokenService := NewTokenService("test_secret_key_1234567890123456")

	user := &domain.User{
		ID:           uuid.New(),
		Name:         "Seller",
		Phone:        "+992901111111",
		PasswordHash: "hashed_pass",
		Role:         domain.RoleSeller,
	}
	_ = userRepo.Create(ctx, user)

	rawRefreshToken := "my_refresh_token_123"
	tokenHash := tokenService.HashToken(rawRefreshToken)

	session := &domain.RefreshSession{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	_ = sessionRepo.CreateSession(ctx, session)

	usecase := NewAuthUsecase(userRepo, sessionRepo, hasher, tokenService)

	newTokens, err := usecase.RefreshToken(ctx, rawRefreshToken, "Mobile")
	if err != nil {
		t.Fatalf("expected refresh success, got: %v", err)
	}

	if newTokens.RefreshToken == rawRefreshToken {
		t.Error("expected new refresh token to be different (rotation)")
	}

	// Verify old session was deleted
	if _, exists := sessionRepo.sessions[tokenHash]; exists {
		t.Error("expected old session to be deleted after rotation")
	}

	// Verify new session exists
	newHash := tokenService.HashToken(newTokens.RefreshToken)
	if _, exists := sessionRepo.sessions[newHash]; !exists {
		t.Error("expected new session to be persisted")
	}
}

func TestAuthUsecase_RefreshToken_Expired(t *testing.T) {
	ctx := context.Background()
	userRepo := newMockUserRepo()
	sessionRepo := newMockSessionRepo()
	hasher := &mockPasswordHasher{}
	tokenService := NewTokenService("test_secret_key_1234567890123456")

	rawRefreshToken := "expired_refresh_token"
	tokenHash := tokenService.HashToken(rawRefreshToken)

	session := &domain.RefreshSession{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(-1 * time.Hour), // expired in past
	}
	_ = sessionRepo.CreateSession(ctx, session)

	usecase := NewAuthUsecase(userRepo, sessionRepo, hasher, tokenService)

	_, err := usecase.RefreshToken(ctx, rawRefreshToken, "Mobile")
	if err != domain.ErrSessionExpired {
		t.Errorf("expected ErrSessionExpired, got %v", err)
	}
}

func TestAuthUsecase_Logout(t *testing.T) {
	ctx := context.Background()
	userRepo := newMockUserRepo()
	sessionRepo := newMockSessionRepo()
	hasher := &mockPasswordHasher{}
	tokenService := NewTokenService("test_secret_key_1234567890123456")

	rawRefreshToken := "token_to_logout"
	tokenHash := tokenService.HashToken(rawRefreshToken)

	session := &domain.RefreshSession{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	_ = sessionRepo.CreateSession(ctx, session)

	usecase := NewAuthUsecase(userRepo, sessionRepo, hasher, tokenService)

	err := usecase.Logout(ctx, rawRefreshToken)
	if err != nil {
		t.Fatalf("expected logout success, got: %v", err)
	}

	if _, exists := sessionRepo.sessions[tokenHash]; exists {
		t.Error("expected session to be deleted on logout")
	}
}

func TestSeedDefaultOwner(t *testing.T) {
	ctx := context.Background()
	userRepo := newMockUserRepo()
	hasher := &mockPasswordHasher{}

	// When empty
	err := SeedDefaultOwner(ctx, userRepo, hasher, "+992900000000", "AdminPass123!")
	if err != nil {
		t.Fatalf("expected seed success, got: %v", err)
	}

	owner, err := userRepo.GetByPhone(ctx, "+992900000000")
	if err != nil {
		t.Fatalf("expected owner created, got error: %v", err)
	}

	if owner.Role != domain.RoleOwner {
		t.Errorf("expected role 'owner', got %s", owner.Role)
	}

	// Calling again should be no-op
	err = SeedDefaultOwner(ctx, userRepo, hasher, "+992900000000", "AdminPass123!")
	if err != nil {
		t.Fatalf("expected idempotent seed, got: %v", err)
	}
	if userRepo.count != 1 {
		t.Errorf("expected user count to stay 1, got %d", userRepo.count)
	}
}
