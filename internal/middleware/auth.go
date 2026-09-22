package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/UmedjonQurbonov/CRM/internal/delivery/http/response"
	"github.com/UmedjonQurbonov/CRM/internal/modules/auth/usecase"
	"github.com/google/uuid"
)

type contextKey string

const (
	UserIDContextKey   contextKey = "user_id"
	UserRoleContextKey contextKey = "user_role"
	UserNameContextKey contextKey = "user_name"
)

// AuthMiddleware validates JWT Bearer token and places user identity into the request context.
func AuthMiddleware(tokenService *usecase.TokenService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authorization header is required", nil)
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid authorization header format. Expected 'Bearer <token>'", nil)
				return
			}

			tokenStr := strings.TrimSpace(parts[1])
			claims, err := tokenService.ValidateAccessToken(tokenStr)
			if err != nil {
				response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid or expired access token", nil)
				return
			}

			parsedID, err := uuid.Parse(claims.UserID)
			if err != nil {
				response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Corrupted identity claims in token", nil)
				return
			}

			ctx := context.WithValue(r.Context(), UserIDContextKey, parsedID)
			ctx = context.WithValue(ctx, UserRoleContextKey, claims.Role)
			ctx = context.WithValue(ctx, UserNameContextKey, claims.Name)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole checks if the authenticated user possesses one of the allowed roles.
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	allowedMap := make(map[string]bool, len(roles))
	for _, r := range roles {
		allowedMap[r] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role, ok := GetUserRole(r.Context())
			if !ok || !allowedMap[role] {
				response.Error(w, http.StatusForbidden, "FORBIDDEN", "You do not have permission to access this resource", nil)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// GetUserID retrieves the authenticated user's UUID from the context.
func GetUserID(ctx context.Context) (uuid.UUID, bool) {
	val := ctx.Value(UserIDContextKey)
	if val == nil {
		return uuid.Nil, false
	}
	id, ok := val.(uuid.UUID)
	return id, ok
}

// GetUserRole retrieves the authenticated user's role from the context.
func GetUserRole(ctx context.Context) (string, bool) {
	val := ctx.Value(UserRoleContextKey)
	if val == nil {
		return "", false
	}
	role, ok := val.(string)
	return role, ok
}
