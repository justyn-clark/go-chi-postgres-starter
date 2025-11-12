package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/yourusername/go-chi-postgres-starter/cmd/api/models"
	"github.com/yourusername/go-chi-postgres-starter/cmd/api/services"
)

type contextKey string

const userIDKey contextKey = "user_id"
const userEmailKey contextKey = "email"
const userRoleKey contextKey = "role"

// JWTAuth middleware validates JWT tokens and sets user context
func JWTAuth(userService *services.UserService, apiAccessToken string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if route is public (no auth required)
			if isPublicRoute(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			// Try API Access Token first (for service-to-service auth)
			if apiAccessToken != "" {
				apiToken := r.Header.Get("X-API-Token")
				if apiToken == apiAccessToken {
					// Valid API token, proceed without user context
					next.ServeHTTP(w, r)
					return
				}
			}

			// Extract token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "authorization header required", http.StatusUnauthorized)
				return
			}

			// Check for Bearer token
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				http.Error(w, "invalid authorization header format", http.StatusUnauthorized)
				return
			}

			token := parts[1]

			// Validate token
			userID, email, role, err := userService.ValidateToken(token)
			if err != nil {
				http.Error(w, "invalid or expired token", http.StatusUnauthorized)
				return
			}

			// Add user info to context
			ctx := context.WithValue(r.Context(), userIDKey, userID)
			ctx = context.WithValue(ctx, userEmailKey, email)
			ctx = context.WithValue(ctx, userRoleKey, role)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserID extracts the user ID from the request context
func GetUserID(ctx context.Context) (uuid.UUID, bool) {
	userID, ok := ctx.Value(userIDKey).(uuid.UUID)
	return userID, ok
}

// GetUserEmail extracts the user email from the request context
func GetUserEmail(ctx context.Context) (string, bool) {
	email, ok := ctx.Value(userEmailKey).(string)
	return email, ok
}

// GetUserRole extracts the user role from the request context
func GetUserRole(ctx context.Context) (models.UserRole, bool) {
	role, ok := ctx.Value(userRoleKey).(models.UserRole)
	return role, ok
}

// RequireAdmin middleware ensures the user is an admin
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		role, ok := GetUserRole(r.Context())
		if !ok || role != models.RoleAdmin {
			http.Error(w, "admin access required", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// RequireOwnerOrAdmin middleware ensures the user is accessing their own resource or is an admin
// Extracts the resource ID from the URL parameter (e.g., {id} from /api/users/{id})
func RequireOwnerOrAdmin(paramName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get current user ID from context
			currentUserID, ok := GetUserID(r.Context())
			if !ok {
				http.Error(w, "authentication required", http.StatusUnauthorized)
				return
			}

			// Get user role
			role, _ := GetUserRole(r.Context())

			// If admin, allow access to any resource
			if role == models.RoleAdmin {
				next.ServeHTTP(w, r)
				return
			}

			// Extract resource ID from URL parameter
			resourceIDStr := chi.URLParam(r, paramName)
			if resourceIDStr == "" {
				http.Error(w, "resource ID required", http.StatusBadRequest)
				return
			}

			resourceID, err := uuid.Parse(resourceIDStr)
			if err != nil {
				http.Error(w, "invalid resource ID", http.StatusBadRequest)
				return
			}

			// Check if user is accessing their own resource
			if currentUserID != resourceID {
				http.Error(w, "access denied: you can only access your own resources", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// isPublicRoute checks if a route is public (no authentication required)
func isPublicRoute(path string) bool {
	publicRoutes := []string{
		"/api/health",
		"/api/auth/register",
		"/api/auth/login",
		"/api/auth/request-password-reset",
		"/api/auth/reset-password",
		"/swagger",
	}

	for _, route := range publicRoutes {
		if strings.HasPrefix(path, route) {
			return true
		}
	}

	return false
}
