package errors

import (
	"fmt"
	"net/http"
)

// APIError represents an API error with HTTP status code
type APIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Err     error  `json:"-"`
}

// Error implements the error interface
func (e *APIError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// Unwrap returns the underlying error
func (e *APIError) Unwrap() error {
	return e.Err
}

// NewAPIError creates a new API error
func NewAPIError(code int, message string) *APIError {
	return &APIError{
		Code:    code,
		Message: message,
	}
}

// WrapAPIError wraps an existing error with API error context
func WrapAPIError(code int, message string, err error) *APIError {
	return &APIError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// Common API errors
var (
	ErrNotFound         = NewAPIError(http.StatusNotFound, "resource not found")
	ErrUnauthorized     = NewAPIError(http.StatusUnauthorized, "unauthorized")
	ErrForbidden        = NewAPIError(http.StatusForbidden, "forbidden")
	ErrBadRequest       = NewAPIError(http.StatusBadRequest, "bad request")
	ErrInternalServer   = NewAPIError(http.StatusInternalServerError, "internal server error")
	ErrConflict         = NewAPIError(http.StatusConflict, "resource conflict")
	ErrValidationFailed = NewAPIError(http.StatusBadRequest, "validation failed")
)

// IsAPIError checks if an error is an APIError
func IsAPIError(err error) bool {
	_, ok := err.(*APIError)
	return ok
}

// GetAPIError extracts APIError from error chain
func GetAPIError(err error) *APIError {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr
	}
	return nil
}
