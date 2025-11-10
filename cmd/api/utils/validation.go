package utils

import (
	"fmt"
	"strings"
	"sync"

	"github.com/go-playground/validator/v10"
)

var (
	// Validate is a singleton validator instance
	Validate *validator.Validate
	once     sync.Once
)

// getValidator returns the singleton validator instance, initializing it if needed
func getValidator() *validator.Validate {
	once.Do(func() {
		Validate = validator.New()
	})
	return Validate
}

// ValidateStruct validates a struct using validator tags
func ValidateStruct(s interface{}) error {
	if err := getValidator().Struct(s); err != nil {
		var validationErrors []string
		for _, err := range err.(validator.ValidationErrors) {
			validationErrors = append(validationErrors, formatValidationError(err))
		}
		return fmt.Errorf("validation failed: %s", strings.Join(validationErrors, ", "))
	}
	return nil
}

// formatValidationError formats a validation error into a user-friendly message
func formatValidationError(err validator.FieldError) string {
	field := err.Field()
	tag := err.Tag()

	switch tag {
	case "required":
		return fmt.Sprintf("%s is required", field)
	case "email":
		return fmt.Sprintf("%s must be a valid email address", field)
	case "min":
		return fmt.Sprintf("%s must be at least %s characters", field, err.Param())
	case "max":
		return fmt.Sprintf("%s must be at most %s characters", field, err.Param())
	default:
		return fmt.Sprintf("%s is invalid", field)
	}
}

// SanitizeString removes leading/trailing whitespace
func SanitizeString(s string) string {
	return strings.TrimSpace(s)
}

// IsEmpty checks if a string is empty or only whitespace
func IsEmpty(s string) bool {
	return strings.TrimSpace(s) == ""
}
