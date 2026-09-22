package usecase

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/UmedjonQurbonov/CRM/internal/modules/auth/domain"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// TokenPair holds the access and refresh tokens.
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"` // in seconds
	TokenType    string `json:"token_type"`
}

// UserClaims defines JWT custom claims for access tokens.
type UserClaims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	Name   string `json:"name"`
	jwt.RegisteredClaims
}

// TokenService manages generation and validation of tokens.
type TokenService struct {
	jwtSecret      []byte
	accessDuration time.Duration
}

// NewTokenService creates a new TokenService.
func NewTokenService(jwtSecret string) *TokenService {
	return &TokenService{
		jwtSecret:      []byte(jwtSecret),
		accessDuration: 15 * time.Minute,
	}
}

// GenerateAccessToken signs a new HMAC JWT for the given user.
func (s *TokenService) GenerateAccessToken(user *domain.User) (string, error) {
	now := time.Now().UTC()
	claims := UserClaims{
		UserID: user.ID.String(),
		Role:   user.Role,
		Name:   user.Name,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.accessDuration)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("failed to sign access token: %w", err)
	}

	return tokenString, nil
}

// GenerateRefreshToken creates a cryptographically secure random token string.
func (s *TokenService) GenerateRefreshToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random token: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// HashToken computes the SHA-256 hash of an opaque token string.
func (s *TokenService) HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// ValidateAccessToken parses and verifies a JWT access token string.
func (s *TokenService) ValidateAccessToken(tokenStr string) (*UserClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &UserClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})

	if err != nil {
		return nil, domain.ErrUnauthorized
	}

	claims, ok := token.Claims.(*UserClaims)
	if !ok || !token.Valid {
		return nil, domain.ErrUnauthorized
	}

	// Verify valid UUID
	if _, err := uuid.Parse(claims.UserID); err != nil {
		return nil, errors.New("invalid user_id claim")
	}

	return claims, nil
}

func (s *TokenService) AccessDurationSeconds() int64 {
	return int64(s.accessDuration.Seconds())
}
