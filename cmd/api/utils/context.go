package utils

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

// Note: This file provides alternative context helpers.
// The recommended approach is to use middleware.GetUserID() from cmd/api/middleware/auth.go
// which is already integrated with the JWT authentication middleware.
// These functions are provided for cases where you need a different context key or pattern.

type contextKey string

const (
	userIDKey contextKey = "userID"
)

// SetUserID sets the user ID in the context
func SetUserID(ctx context.Context, userID uuid.UUID) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// GetUserID retrieves the user ID from the context
// Returns an error if user ID is not found or invalid
func GetUserID(ctx context.Context) (uuid.UUID, error) {
	userID, ok := ctx.Value(userIDKey).(uuid.UUID)
	if !ok {
		return uuid.Nil, errors.New("user ID not found in context")
	}
	return userID, nil
}

// MustGetUserID retrieves the user ID from context or panics
// Use only when you're certain the user ID exists
func MustGetUserID(ctx context.Context) uuid.UUID {
	userID, err := GetUserID(ctx)
	if err != nil {
		panic(err)
	}
	return userID
}
