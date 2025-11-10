package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
	"github.com/yourusername/go-api-starter/cmd/api/models"
)

// Error messages
const (
	ErrUserNotFoundMsg = "user not found"
	ErrUserExistsMsg   = "user with this email already exists"
)

// Sentinel errors for user operations
var (
	ErrUserNotFound      = errors.New(ErrUserNotFoundMsg)
	ErrUserAlreadyExists = errors.New(ErrUserExistsMsg)
)

// UserRepository handles database operations for users
type UserRepository struct {
	db *pgxpool.Pool
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

// Create creates a new user in the database
func (r *UserRepository) Create(ctx context.Context, user *models.User) error {
	query := `
		INSERT INTO users (id, email, full_name, password, role, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, email, full_name, role, created_at, updated_at
	`

	now := time.Now()
	role := string(user.Role)
	if role == "" {
		role = string(models.RoleUser)
	}

	err := r.db.QueryRow(
		ctx,
		query,
		user.ID,
		user.Email,
		user.FullName,
		user.Password,
		role,
		now,
		now,
	).Scan(
		&user.ID,
		&user.Email,
		&user.FullName,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		// Check for unique constraint violation (duplicate email)
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrUserAlreadyExists
		}
		log.Error().Err(err).Str("email", user.Email).Msg("failed to create user")
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// FindByEmail finds a user by email address
func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	query := `
		SELECT id, email, full_name, password, role, password_reset_token, password_reset_expires_at, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	user := &models.User{}
	err := r.db.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.FullName,
		&user.Password,
		&user.Role,
		&user.PasswordResetToken,
		&user.PasswordResetExpiresAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrUserNotFound
		}
		log.Error().Err(err).Str("email", email).Msg("failed to find user by email")
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	return user, nil
}

// FindByID finds a user by ID
func (r *UserRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	query := `
		SELECT id, email, full_name, password, role, password_reset_token, password_reset_expires_at, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	user := &models.User{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.FullName,
		&user.Password,
		&user.Role,
		&user.PasswordResetToken,
		&user.PasswordResetExpiresAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrUserNotFound
		}
		log.Error().Err(err).Str("id", id.String()).Msg("failed to find user by id")
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	return user, nil
}

// List returns all users (paginated)
func (r *UserRepository) List(ctx context.Context, limit, offset int) ([]*models.User, error) {
	query := `
		SELECT id, email, full_name, password, role, password_reset_token, password_reset_expires_at, created_at, updated_at
		FROM users
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		log.Error().Err(err).Msg("failed to list users")
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	users := []*models.User{}
	for rows.Next() {
		user := &models.User{}
		if err := rows.Scan(
			&user.ID,
			&user.Email,
			&user.FullName,
			&user.Password,
			&user.Role,
			&user.PasswordResetToken,
			&user.PasswordResetExpiresAt,
			&user.CreatedAt,
			&user.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating users: %w", err)
	}

	return users, nil
}

// Update updates a user's information
func (r *UserRepository) Update(ctx context.Context, user *models.User) error {
	query := `
		UPDATE users
		SET email = $2, full_name = $3, updated_at = $4
		WHERE id = $1
		RETURNING updated_at
	`

	now := time.Now()
	err := r.db.QueryRow(
		ctx,
		query,
		user.ID,
		user.Email,
		user.FullName,
		now,
	).Scan(&user.UpdatedAt)

	if err != nil {
		if err == pgx.ErrNoRows {
			return ErrUserNotFound
		}
		log.Error().Err(err).Str("id", user.ID.String()).Msg("failed to update user")
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

// UpdatePassword updates a user's password
func (r *UserRepository) UpdatePassword(ctx context.Context, userID uuid.UUID, hashedPassword string) error {
	query := `
		UPDATE users
		SET password = $2, password_reset_token = NULL, password_reset_expires_at = NULL, updated_at = $3
		WHERE id = $1
		RETURNING updated_at
	`

	now := time.Now()
	var updatedAt time.Time
	err := r.db.QueryRow(ctx, query, userID, hashedPassword, now).Scan(&updatedAt)

	if err != nil {
		if err == pgx.ErrNoRows {
			return ErrUserNotFound
		}
		log.Error().Err(err).Str("id", userID.String()).Msg("failed to update password")
		return fmt.Errorf("failed to update password: %w", err)
	}

	return nil
}

// SetPasswordResetToken sets a password reset token for a user
func (r *UserRepository) SetPasswordResetToken(
	ctx context.Context,
	email string,
	token string,
	expiresAt time.Time,
) error {
	query := `
		UPDATE users
		SET password_reset_token = $2, password_reset_expires_at = $3, updated_at = $4
		WHERE email = $1
	`

	now := time.Now()
	result, err := r.db.Exec(ctx, query, email, token, expiresAt, now)
	if err != nil {
		log.Error().Err(err).Str("email", email).Msg("failed to set password reset token")
		return fmt.Errorf("failed to set password reset token: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrUserNotFound
	}

	return nil
}

// FindByPasswordResetToken finds a user by password reset token
func (r *UserRepository) FindByPasswordResetToken(ctx context.Context, token string) (*models.User, error) {
	query := `
		SELECT id, email, full_name, password, role, password_reset_token, password_reset_expires_at, created_at, updated_at
		FROM users
		WHERE password_reset_token = $1 AND password_reset_expires_at > NOW()
	`

	user := &models.User{}
	err := r.db.QueryRow(ctx, query, token).Scan(
		&user.ID,
		&user.Email,
		&user.FullName,
		&user.Password,
		&user.Role,
		&user.PasswordResetToken,
		&user.PasswordResetExpiresAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.New("invalid or expired reset token")
		}
		log.Error().Err(err).Msg("failed to find user by reset token")
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	return user, nil
}

// UpdateRole updates a user's role (admin only)
func (r *UserRepository) UpdateRole(ctx context.Context, userID uuid.UUID, role models.UserRole) error {
	query := `
		UPDATE users
		SET role = $2, updated_at = $3
		WHERE id = $1
		RETURNING updated_at
	`

	now := time.Now()
	err := r.db.QueryRow(ctx, query, userID, string(role), now).Scan(&now)

	if err != nil {
		if err == pgx.ErrNoRows {
			return ErrUserNotFound
		}
		log.Error().Err(err).Str("id", userID.String()).Msg("failed to update role")
		return fmt.Errorf("failed to update role: %w", err)
	}

	return nil
}

// Delete deletes a user by ID
func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM users WHERE id = $1`

	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		log.Error().Err(err).Str("id", id.String()).Msg("failed to delete user")
		return fmt.Errorf("failed to delete user: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrUserNotFound
	}

	return nil
}
