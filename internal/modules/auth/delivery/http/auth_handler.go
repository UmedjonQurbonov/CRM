package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/UmedjonQurbonov/CRM/internal/delivery/http/response"
	"github.com/UmedjonQurbonov/CRM/internal/modules/auth/domain"
	"github.com/UmedjonQurbonov/CRM/internal/modules/auth/usecase"
)

type AuthHandler struct {
	authUsecase *usecase.AuthUsecase
}

func NewAuthHandler(authUsecase *usecase.AuthUsecase) *AuthHandler {
	return &AuthHandler{authUsecase: authUsecase}
}

// LoginRequest defines credentials payload for login.
type LoginRequest struct {
	Phone    string `json:"phone" example:"+992900000000"`
	Password string `json:"password" example:"AdminPass123!"`
}

// UserDTO defines user presentation details.
type UserDTO struct {
	ID             string `json:"id" example:"a43c2c77-4cf7-4f81-9b16-560ef71c9b68"`
	Name           string `json:"name" example:"Admin Owner"`
	Phone          string `json:"phone" example:"+992900000000"`
	Role           string `json:"role" example:"owner"`
	CommissionRate string `json:"commission_rate" example:"0.00"`
}

// AuthResponse returns token pair along with user profile.
type AuthResponse struct {
	Tokens usecase.TokenPair `json:"tokens"`
	User   UserDTO           `json:"user"`
}

// RefreshRequest defines payload for token rotation.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" example:"5e884898da28047151d0e56f8dc6292773603d0d6aabbdd62a11ef721d1542d8"`
}

// RefreshResponse returns rotated token pair.
type RefreshResponse struct {
	Tokens usecase.TokenPair `json:"tokens"`
}

// LogoutRequest defines payload for logout.
type LogoutRequest struct {
	RefreshToken string `json:"refresh_token" example:"5e884898da28047151d0e56f8dc6292773603d0d6aabbdd62a11ef721d1542d8"`
}

// MessageResponse represents simple message output.
type MessageResponse struct {
	Message string `json:"message" example:"Success"`
}

// Login godoc
// @Summary User Login
// @Description Authenticates user by phone and password, issuing access and refresh tokens.
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Login credentials"
// @Success 200 {object} AuthResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/v1/auth/login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "Malformed JSON body", nil)
		return
	}

	deviceInfo := r.UserAgent()
	tokens, user, err := h.authUsecase.Login(r.Context(), req.Phone, req.Password, deviceInfo)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidCredentials) {
			response.Error(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid phone or password", nil)
			return
		}
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Authentication failed", nil)
		return
	}

	resp := AuthResponse{
		Tokens: *tokens,
		User: UserDTO{
			ID:             user.ID.String(),
			Name:           user.Name,
			Phone:          user.Phone,
			Role:           user.Role,
			CommissionRate: user.CommissionRate.StringFixed(2),
		},
	}

	response.JSON(w, http.StatusOK, resp)
}

// Refresh godoc
// @Summary Refresh Access Token
// @Description Validates refresh token, invalidates it (rotation), and issues a new token pair.
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body RefreshRequest true "Refresh token"
// @Success 200 {object} RefreshResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/v1/auth/refresh [post]
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "Malformed JSON body", nil)
		return
	}

	deviceInfo := r.UserAgent()
	tokens, err := h.authUsecase.RefreshToken(r.Context(), req.RefreshToken, deviceInfo)
	if err != nil {
		if errors.Is(err, domain.ErrSessionExpired) {
			response.Error(w, http.StatusUnauthorized, "SESSION_EXPIRED", "Refresh session has expired", nil)
			return
		}
		if errors.Is(err, domain.ErrUnauthorized) {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid or revoked refresh token", nil)
			return
		}
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Token refresh failed", nil)
		return
	}

	response.JSON(w, http.StatusOK, RefreshResponse{Tokens: *tokens})
}

// Logout godoc
// @Summary Logout
// @Description Invalidates user refresh token session.
// @Tags Auth
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body LogoutRequest true "Refresh token to revoke"
// @Success 200 {object} MessageResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/v1/auth/logout [post]
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	var req LogoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "Malformed JSON body", nil)
		return
	}

	if err := h.authUsecase.Logout(r.Context(), req.RefreshToken); err != nil {
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Logout failed", nil)
		return
	}

	response.JSON(w, http.StatusOK, MessageResponse{Message: "Logged out successfully"})
}
