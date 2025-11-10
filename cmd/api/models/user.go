package models

import (
	"time"

	"github.com/google/uuid"
)

// UserRole represents user roles
type UserRole string

const (
	RoleUser  UserRole = "user"
	RoleAdmin UserRole = "admin"
)

// User represents a user in the system
type User struct {
	ID                     uuid.UUID  `json:"id" db:"id"`
	Email                  string     `json:"email" db:"email"`
	FullName               string     `json:"full_name" db:"full_name"`
	Password               string     `json:"-" db:"password"` // Never return password in JSON
	Role                   UserRole   `json:"role" db:"role"`
	PasswordResetToken     *string    `json:"-" db:"password_reset_token"`
	PasswordResetExpiresAt *time.Time `json:"-" db:"password_reset_expires_at"`
	CreatedAt              time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at" db:"updated_at"`
}

// IsAdmin checks if the user is an admin
func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin
}

// RegisterRequest represents a user registration request
type RegisterRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
	FullName string `json:"full_name" validate:"required,min=1"`
}

// LoginRequest represents a user login request
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// LoginResponse represents the response after successful login
type LoginResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

// UpdateUserRequest represents a user update request
type UpdateUserRequest struct {
	Email    string `json:"email" validate:"required,email"`
	FullName string `json:"full_name" validate:"required,min=1"`
}

// PasswordResetRequest represents a password reset request
type PasswordResetRequest struct {
	Email string `json:"email" validate:"required,email"`
}

// PasswordResetConfirm represents a password reset confirmation
type PasswordResetConfirm struct {
	Token    string `json:"token" validate:"required"`
	Password string `json:"password" validate:"required,min=8"`
}

// ChangePasswordRequest represents a password change request (for logged-in users)
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" validate:"required"`
	NewPassword     string `json:"new_password" validate:"required,min=8"`
}

// UserResponse represents a user in API responses (without sensitive data)
type UserResponse struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	FullName  string    `json:"full_name"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ToResponse converts a User to UserResponse (removes sensitive fields)
func (u *User) ToResponse() *UserResponse {
	return &UserResponse{
		ID:        u.ID,
		Email:     u.Email,
		FullName:  u.FullName,
		Role:      string(u.Role),
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string          `json:"status"`
	Service   string          `json:"service"`
	Timestamp string          `json:"timestamp"`
	Database  *DatabaseHealth `json:"database,omitempty"`
	Version   string          `json:"version,omitempty"`
	Uptime    string          `json:"uptime,omitempty"`
}

// DatabaseHealth represents database health status
type DatabaseHealth struct {
	Status       string `json:"status"`
	ResponseTime string `json:"response_time,omitempty"`
	Error        string `json:"error,omitempty"`
}
