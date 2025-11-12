package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/yourusername/go-chi-postgres-starter/cmd/api/middleware"
	"github.com/yourusername/go-chi-postgres-starter/cmd/api/models"
	"github.com/yourusername/go-chi-postgres-starter/cmd/api/repository"
	"github.com/yourusername/go-chi-postgres-starter/cmd/api/services"
	"github.com/yourusername/go-chi-postgres-starter/cmd/api/utils"
)

// UserHandler handles user-related HTTP requests
type UserHandler struct {
	userService *services.UserService
}

// NewUserHandler creates a new user handler
func NewUserHandler(userService *services.UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
	}
}

// GetMe retrieves the current authenticated user's profile
// @Summary Get current user
// @Description Get the authenticated user's own profile
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} models.UserResponse
// @Failure 401 {object} ErrorResponse
// @Router /api/users/me [get]
func (h *UserHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	// Get current user ID from context
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	user, err := h.userService.GetUser(r.Context(), userID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			respondError(w, http.StatusNotFound, repository.ErrUserNotFoundMsg)
			return
		}
		log.Error().Err(err).Str("id", userID.String()).Msg("failed to get user")
		respondError(w, http.StatusInternalServerError, "failed to get user")
		return
	}

	respondJSON(w, http.StatusOK, user.ToResponse())
}

// ListUsers retrieves a list of users (admin only)
// @Summary List users
// @Description Get a paginated list of all users (admin only)
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param limit query int false "Limit" default(10)
// @Param offset query int false "Offset" default(0)
// @Success 200 {array} models.UserResponse
// @Failure 403 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /api/users [get]
func (h *UserHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	limit := 10
	offset := 0

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	users, err := h.userService.ListUsers(r.Context(), limit, offset)
	if err != nil {
		log.Error().Err(err).Msg("failed to list users")
		respondError(w, http.StatusInternalServerError, "failed to list users")
		return
	}

	responses := make([]*models.UserResponse, len(users))
	for i, user := range users {
		responses[i] = user.ToResponse()
	}

	respondJSON(w, http.StatusOK, responses)
}

// GetUser retrieves a user by ID (users can only get themselves, admins can get any)
// @Summary Get user by ID
// @Description Get a specific user by their ID. Users can only access their own profile, admins can access any.
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID"
// @Success 200 {object} models.UserResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /api/users/{id} [get]
func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid user ID")
		return
	}

	user, err := h.userService.GetUser(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			respondError(w, http.StatusNotFound, repository.ErrUserNotFoundMsg)
			return
		}
		log.Error().Err(err).Str("id", id.String()).Msg("failed to get user")
		respondError(w, http.StatusInternalServerError, "failed to get user")
		return
	}

	respondJSON(w, http.StatusOK, user.ToResponse())
}

// CreateUser creates a new user
// @Summary Create a new user
// @Description Create a new user account (admin only)
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.RegisterRequest true "User data"
// @Success 201 {object} models.UserResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /api/users [post]
func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
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
		// Check if it's a duplicate email error
		if errors.Is(err, repository.ErrUserAlreadyExists) {
			respondError(w, http.StatusConflict, repository.ErrUserExistsMsg)
			return
		}
		log.Error().Err(err).Str("email", req.Email).Msg("failed to create user")
		respondError(w, http.StatusInternalServerError, "failed to create user")
		return
	}

	respondJSON(w, http.StatusCreated, user.ToResponse())
}

// UpdateUser updates a user's information (users can only update themselves, admins can update any)
// @Summary Update user
// @Description Update a user's information. Users can only update their own profile, admins can update any.
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID"
// @Param request body models.UpdateUserRequest true "User data"
// @Success 200 {object} models.UserResponse
// @Failure 400 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /api/users/{id} [put]
func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid user ID")
		return
	}

	var req models.UpdateUserRequest

	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate request using validator
	if err := utils.ValidateStruct(&req); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	user, err := h.userService.UpdateUser(r.Context(), id, req.Email, req.FullName)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			respondError(w, http.StatusNotFound, repository.ErrUserNotFoundMsg)
			return
		}
		log.Error().Err(err).Str("id", id.String()).Msg("failed to update user")
		respondError(w, http.StatusInternalServerError, "failed to update user")
		return
	}

	respondJSON(w, http.StatusOK, user.ToResponse())
}

// DeleteUser deletes a user (admin only)
// @Summary Delete user
// @Description Delete a user by ID (admin only)
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID"
// @Success 204 "No Content"
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /api/users/{id} [delete]
func (h *UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid user ID")
		return
	}

	if err := h.userService.DeleteUser(r.Context(), id); err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			respondError(w, http.StatusNotFound, repository.ErrUserNotFoundMsg)
			return
		}
		log.Error().Err(err).Str("id", id.String()).Msg("failed to delete user")
		respondError(w, http.StatusInternalServerError, "failed to delete user")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// UpdateUserRole updates a user's role (admin only)
// @Summary Update user role
// @Description Update a user's role (admin only)
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID"
// @Param request body map[string]string true "Role update" example({"role": "admin"})
// @Success 200 {object} models.UserResponse
// @Failure 400 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/users/{id}/role [put]
func (h *UserHandler) UpdateUserRole(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid user ID")
		return
	}

	var req struct {
		Role string `json:"role" validate:"required"`
	}

	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate request
	if err := utils.ValidateStruct(&req); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Validate role value
	if req.Role != string(models.RoleAdmin) && req.Role != string(models.RoleUser) {
		respondError(w, http.StatusBadRequest, "role must be 'user' or 'admin'")
		return
	}

	// Update role
	newRole := models.UserRole(req.Role)
	if err := h.userService.UpdateUserRole(r.Context(), id, newRole); err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			respondError(w, http.StatusNotFound, repository.ErrUserNotFoundMsg)
			return
		}
		log.Error().Err(err).Str("id", id.String()).Msg("failed to update user role")
		respondError(w, http.StatusInternalServerError, "failed to update user role")
		return
	}

	// Get updated user
	user, err := h.userService.GetUser(r.Context(), id)
	if err != nil {
		log.Error().Err(err).Str("id", id.String()).Msg("failed to get updated user")
		respondError(w, http.StatusInternalServerError, "failed to get updated user")
		return
	}

	respondJSON(w, http.StatusOK, user.ToResponse())
}
