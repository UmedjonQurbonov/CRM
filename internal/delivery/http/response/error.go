package response

import (
	"encoding/json"
	"net/http"
)

// ErrorResponse adheres to RFC 7807 problem details specification.
type ErrorResponse struct {
	Code    string         `json:"code" example:"INVALID_CREDENTIALS"`
	Message string         `json:"message" example:"Invalid phone or password"`
	Details map[string]any `json:"details,omitempty"`
}

// JSON sends a JSON response with status code.
func JSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		_ = json.NewEncoder(w).Encode(data)
	}
}

// Error writes a standardized RFC 7807 error payload.
func Error(w http.ResponseWriter, status int, code, message string, details map[string]any) {
	JSON(w, status, ErrorResponse{
		Code:    code,
		Message: message,
		Details: details,
	})
}
