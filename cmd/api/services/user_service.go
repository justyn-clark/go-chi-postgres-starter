package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/yourusername/go-chi-postgres-starter/cmd/api/models"
	"github.com/yourusername/go-chi-postgres-starter/cmd/api/repository"
	"golang.org/x/crypto/bcrypt"
)

// UserService handles business logic for users
type UserService struct {
	userRepo  *repository.UserRepository
	jwtSecret []byte
}

// NewUserService creates a new user service
func NewUserService(userRepo *repository.UserRepository, jwtSecret string) *UserService {
	return &UserService{
		userRepo:  userRepo,
		jwtSecret: []byte(jwtSecret),
	}
}

// Register creates a new user account
func (s *UserService) Register(ctx context.Context, req *models.RegisterRequest) (*models.User, error) {
	// Check if user already exists
	_, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err == nil {
		return nil, repository.ErrUserAlreadyExists
	}
	// If error is not "user not found", return it
	if !errors.Is(err, repository.ErrUserNotFound) {
		return nil, fmt.Errorf("failed to check if user exists: %w", err)
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
	user := &models.User{
		ID:       uuid.New(),
		Email:    req.Email,
		FullName: req.FullName,
		Password: string(hashedPassword),
		Role:     models.RoleUser, // Default role
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		// Return duplicate user error directly
		if err == repository.ErrUserAlreadyExists {
			return nil, err
		}
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

// Login authenticates a user and returns a JWT token
func (s *UserService) Login(ctx context.Context, req *models.LoginRequest) (*models.LoginResponse, error) {
	// Find user by email
	user, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("invalid email or password")
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, fmt.Errorf("invalid email or password")
	}

	// Generate JWT token
	token, err := s.generateToken(user.ID, user.Email, user.Role)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	return &models.LoginResponse{
		Token: token,
		User:  *user,
	}, nil
}

// GetUser retrieves a user by ID
func (s *UserService) GetUser(ctx context.Context, id uuid.UUID) (*models.User, error) {
	user, err := s.userRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// ListUsers retrieves a list of users
func (s *UserService) ListUsers(ctx context.Context, limit, offset int) ([]*models.User, error) {
	users, err := s.userRepo.List(ctx, limit, offset)
	if err != nil {
		return nil, err
	}
	return users, nil
}

// UpdateUser updates a user's information
func (s *UserService) UpdateUser(ctx context.Context, id uuid.UUID, email, fullName string) (*models.User, error) {
	user, err := s.userRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	user.Email = email
	user.FullName = fullName

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// DeleteUser deletes a user by ID
func (s *UserService) DeleteUser(ctx context.Context, id uuid.UUID) error {
	return s.userRepo.Delete(ctx, id)
}

// generateToken creates a JWT token for a user
func (s *UserService) generateToken(userID uuid.UUID, email string, role models.UserRole) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID.String(),
		"email":   email,
		"role":    string(role),
		"exp":     time.Now().Add(time.Hour * 24 * 7).Unix(), // 7 days
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// ValidateToken validates a JWT token and returns the user ID, email, and role
func (s *UserService) ValidateToken(tokenString string) (uuid.UUID, string, models.UserRole, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})

	if err != nil {
		return uuid.Nil, "", models.RoleUser, fmt.Errorf("invalid token: %w", err)
	}

	if !token.Valid {
		return uuid.Nil, "", models.RoleUser, fmt.Errorf("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return uuid.Nil, "", models.RoleUser, fmt.Errorf("invalid token claims")
	}

	userIDStr, ok := claims["user_id"].(string)
	if !ok {
		return uuid.Nil, "", models.RoleUser, fmt.Errorf("invalid user_id in token")
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return uuid.Nil, "", models.RoleUser, fmt.Errorf("invalid user_id format: %w", err)
	}

	email, _ := claims["email"].(string)
	roleStr, _ := claims["role"].(string)
	role := models.UserRole(roleStr)
	if role != models.RoleAdmin && role != models.RoleUser {
		role = models.RoleUser // Default to user if invalid
	}

	return userID, email, role, nil
}

// RequestPasswordReset generates a password reset token and stores it
func (s *UserService) RequestPasswordReset(ctx context.Context, email string) error {
	// Find user
	_, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		// Don't reveal if user exists or not (security best practice)
		return nil // Silently succeed
	}

	// Generate reset token (using UUID for security)
	resetToken := uuid.New().String()

	// Set expiration (1 hour from now)
	expiresAt := time.Now().Add(time.Hour * 1)

	// Store token
	if err := s.userRepo.SetPasswordResetToken(ctx, email, resetToken, expiresAt); err != nil {
		return fmt.Errorf("failed to set reset token: %w", err)
	}

	// NOTE: In production, send email with reset link
	// For development, we log it to console
	// Example email service integration:
	//   emailService.SendPasswordResetEmail(email, resetToken)
	log.Info().
		Str("email", email).
		Str("token", resetToken).
		Msg("password reset token generated (send email in production)")
	fmt.Printf("Password reset token for %s: %s (expires in 1 hour)\n", email, resetToken)
	fmt.Printf("Reset link: http://localhost:8080/api/auth/reset-password?token=%s\n", resetToken)

	return nil
}

// ResetPassword resets a user's password using a reset token
func (s *UserService) ResetPassword(ctx context.Context, token, newPassword string) error {
	// Find user by token
	user, err := s.userRepo.FindByPasswordResetToken(ctx, token)
	if err != nil {
		return fmt.Errorf("invalid or expired reset token")
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Update password
	if err := s.userRepo.UpdatePassword(ctx, user.ID, string(hashedPassword)); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	return nil
}

// ChangePassword changes a user's password (for logged-in users)
func (s *UserService) ChangePassword(ctx context.Context, userID uuid.UUID, currentPassword, newPassword string) error {
	// Get user
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("user not found")
	}

	// Verify current password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(currentPassword)); err != nil {
		return fmt.Errorf("current password is incorrect")
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Update password
	if err := s.userRepo.UpdatePassword(ctx, userID, string(hashedPassword)); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	return nil
}

// UpdateUserRole updates a user's role (admin only)
func (s *UserService) UpdateUserRole(ctx context.Context, targetUserID uuid.UUID, newRole models.UserRole) error {
	if newRole != models.RoleAdmin && newRole != models.RoleUser {
		return fmt.Errorf("invalid role")
	}

	return s.userRepo.UpdateRole(ctx, targetUserID, newRole)
}
