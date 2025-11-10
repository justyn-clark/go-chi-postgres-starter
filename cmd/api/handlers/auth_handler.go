package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/rs/zerolog/log"
	"github.com/yourusername/go-api-starter/cmd/api/middleware"
	"github.com/yourusername/go-api-starter/cmd/api/models"
	"github.com/yourusername/go-api-starter/cmd/api/repository"
	"github.com/yourusername/go-api-starter/cmd/api/services"
	"github.com/yourusername/go-api-starter/cmd/api/utils"
)

var GetUserID = middleware.GetUserID

// AuthHandler handles authentication-related HTTP requests
type AuthHandler struct {
	userService *services.UserService
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(userService *services.UserService) *AuthHandler {
	return &AuthHandler{
		userService: userService,
	}
}

// Register handles user registration
// @Summary Register a new user
// @Description Create a new user account
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body models.RegisterRequest true "Registration data"
// @Success 201 {object} models.UserResponse
// @Failure 400 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Router /api/auth/register [post]
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req models.RegisterRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate request using validator
	if err := utils.ValidateStruct(&req); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	user, err := h.userService.Register(r.Context(), &req)
	if err != nil {
		// Check for duplicate user error
		if errors.Is(err, repository.ErrUserAlreadyExists) {
			respondError(w, http.StatusConflict, "user with this email already exists")
			return
		}
		log.Error().Err(err).Str("email", req.Email).Msg("failed to register user")
		respondError(w, http.StatusInternalServerError, "failed to register user")
		return
	}

	respondJSON(w, http.StatusCreated, user.ToResponse())
}

// Login handles user login
// @Summary Login user
// @Description Authenticate user and get JWT token
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body models.LoginRequest true "Login credentials"
// @Success 200 {object} models.LoginResponse
// @Failure 401 {object} ErrorResponse
// @Router /api/auth/login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate request using validator
	if err := utils.ValidateStruct(&req); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	response, err := h.userService.Login(r.Context(), &req)
	if err != nil {
		log.Error().Err(err).Str("email", req.Email).Msg("failed to login")
		respondError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	respondJSON(w, http.StatusOK, response)
}

// RequestPasswordReset handles password reset requests
// @Summary Request password reset
// @Description Request a password reset token via email
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body models.PasswordResetRequest true "Password reset request"
// @Success 200 {object} map[string]string
// @Failure 400 {object} ErrorResponse
// @Router /api/auth/request-password-reset [post]
func (h *AuthHandler) RequestPasswordReset(w http.ResponseWriter, r *http.Request) {
	var req models.PasswordResetRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate request
	if err := utils.ValidateStruct(&req); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Request password reset (always returns success for security)
	if err := h.userService.RequestPasswordReset(r.Context(), req.Email); err != nil {
		log.Error().Err(err).Str("email", req.Email).Msg("failed to request password reset")
		// Still return success to prevent email enumeration
	}

	// Always return success message (security best practice)
	respondJSON(w, http.StatusOK, map[string]string{
		"message": "If an account with that email exists, a password reset link has been sent",
	})
}

// ResetPassword handles password reset confirmation
// @Summary Reset password
// @Description Reset password using a reset token
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body models.PasswordResetConfirm true "Password reset confirmation"
// @Success 200 {object} map[string]string
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /api/auth/reset-password [post]
func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req models.PasswordResetConfirm
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate request
	if err := utils.ValidateStruct(&req); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Reset password
	if err := h.userService.ResetPassword(r.Context(), req.Token, req.Password); err != nil {
		log.Error().Err(err).Msg("failed to reset password")
		respondError(w, http.StatusBadRequest, "invalid or expired reset token")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "Password has been reset successfully",
	})
}

// ChangePassword handles password change for logged-in users
// @Summary Change password
// @Description Change password for the authenticated user
// @Tags Authentication
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.ChangePasswordRequest true "Password change request"
// @Success 200 {object} map[string]string
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /api/auth/change-password [post]
func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := GetUserID(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	var req models.ChangePasswordRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate request
	if err := utils.ValidateStruct(&req); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Change password
	if err := h.userService.ChangePassword(r.Context(), userID, req.CurrentPassword, req.NewPassword); err != nil {
		log.Error().Err(err).Str("user_id", userID.String()).Msg("failed to change password")
		if err.Error() == "current password is incorrect" {
			respondError(w, http.StatusBadRequest, "current password is incorrect")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to change password")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "Password has been changed successfully",
	})
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error"`
}

// respondJSON sends a JSON response
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Error().Err(err).Msg("failed to encode JSON response")
	}
}

// respondError sends an error response
func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, ErrorResponse{Error: message})
}

// decodeJSON decodes JSON request body into the provided value
// Returns an error if decoding fails, allowing explicit error handling
func decodeJSON(r *http.Request, v interface{}) error {
	return json.NewDecoder(r.Body).Decode(v)
}
