package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"time"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yourusername/go-api-starter/cmd/api/database"
	"github.com/yourusername/go-api-starter/cmd/api/handlers"
	"github.com/yourusername/go-api-starter/cmd/api/middleware"
	"github.com/yourusername/go-api-starter/cmd/api/models"
	"github.com/yourusername/go-api-starter/cmd/api/repository"
	"github.com/yourusername/go-api-starter/cmd/api/services"
)

// Note: Tests require a test database
// Set TEST_DATABASE_URL environment variable or use default:
// postgresql://postgres:postgres@localhost:5434/go_api_starter_test?sslmode=disable
// Make sure to run migrations before running tests: make test-setup

var testRouter *chi.Mux
var testDB *database.DB

// setupTestRouter creates a test router with all routes configured
func setupTestRouter(t *testing.T) *chi.Mux {
	if testRouter != nil {
		return testRouter
	}

	// Get test database URL from environment or use default
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = os.Getenv("DATABASE_URL")
	}
	if dbURL == "" {
		// Default to Docker database port (5434) for test database
		dbURL = "postgresql://postgres:postgres@localhost:5434/go_api_starter_test?sslmode=disable"
	}

	// Connect to test database
	ctx := context.Background()
	var err error
	testDB, err = database.NewDB(ctx, dbURL)
	require.NoError(t, err, "Failed to connect to test database")

	// Create router
	router := chi.NewRouter()

	// Middleware stack
	router.Use(chiMiddleware.RequestID)
	router.Use(chiMiddleware.RealIP)
	router.Use(middleware.SimpleRequestLogger)
	router.Use(middleware.Recoverer)
	router.Use(chiMiddleware.Compress(5))
	router.Use(chiMiddleware.Timeout(60 * time.Second))

	// Initialize repositories
	userRepo := repository.NewUserRepository(testDB.Pool)

	// Initialize services
	jwtSecret := "test-jwt-secret-for-testing-only"
	userService := services.NewUserService(userRepo, jwtSecret)

	// Initialize handlers
	healthHandler := handlers.NewHealthHandler(testDB)
	authHandler := handlers.NewAuthHandler(userService)
	userHandler := handlers.NewUserHandler(userService)

	// API Routes
	router.Route("/api", func(r chi.Router) {
		// Apply JWT auth to all routes
		r.Use(middleware.JWTAuth(userService, ""))

		// Public endpoints
		r.Get("/health", healthHandler.Health)
		r.Post("/auth/register", authHandler.Register)
		r.Post("/auth/login", authHandler.Login)

		// Protected endpoints
		r.Route("/users", func(r chi.Router) {
			r.Get("/", userHandler.ListUsers)
			r.Post("/", userHandler.CreateUser)
			r.Get("/{id}", userHandler.GetUser)
			r.Put("/{id}", userHandler.UpdateUser)
			r.Delete("/{id}", userHandler.DeleteUser)
		})
	})

	testRouter = router
	return router
}

// teardownTest cleans up test resources
func teardownTest(t *testing.T) {
	if testDB != nil {
		// Clean up test data
		ctx := context.Background()
		_, err := testDB.Pool.Exec(ctx, "TRUNCATE TABLE users CASCADE")
		if err != nil {
			t.Logf("Warning: Failed to clean up test data: %v", err)
		}
	}
}

// TestHealthEndpoint tests the health check endpoint
func TestHealthEndpoint(t *testing.T) {
	router := setupTestRouter(t)

	t.Run("health check returns ok", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/health", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response models.HealthResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "ok", response.Status)
		assert.Equal(t, "go-api-starter", response.Service)
		assert.NotEmpty(t, response.Timestamp)
	})
}

// TestAuthEndpoints tests authentication endpoints
func TestAuthEndpoints(t *testing.T) {
	router := setupTestRouter(t)
	defer teardownTest(t)

	t.Run("register new user", func(t *testing.T) {
		registerReq := models.RegisterRequest{
			Email:    "test@example.com",
			Password: "password123",
			FullName: "Test User",
		}

		body, _ := json.Marshal(registerReq)
		req := httptest.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response models.UserResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, registerReq.Email, response.Email)
		assert.Equal(t, registerReq.FullName, response.FullName)
		assert.NotEmpty(t, response.ID)
	})

	t.Run("register duplicate email fails", func(t *testing.T) {
		registerReq := models.RegisterRequest{
			Email:    "duplicate@example.com",
			Password: "password123",
			FullName: "First User",
		}

		body, _ := json.Marshal(registerReq)
		req := httptest.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)

		// Try to register again with same email
		req2 := httptest.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(body))
		req2.Header.Set("Content-Type", "application/json")
		w2 := httptest.NewRecorder()

		router.ServeHTTP(w2, req2)
		assert.Equal(t, http.StatusConflict, w2.Code)
	})

	t.Run("register with invalid data fails", func(t *testing.T) {
		testCases := []struct {
			name    string
			request models.RegisterRequest
			want    int
		}{
			{
				name: "missing email",
				request: models.RegisterRequest{
					Password: "password123",
					FullName: "Test User",
				},
				want: http.StatusBadRequest,
			},
			{
				name: "missing password",
				request: models.RegisterRequest{
					Email:    "test@example.com",
					FullName: "Test User",
				},
				want: http.StatusBadRequest,
			},
			{
				name: "short password",
				request: models.RegisterRequest{
					Email:    "test@example.com",
					Password: "short",
					FullName: "Test User",
				},
				want: http.StatusBadRequest,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				body, _ := json.Marshal(tc.request)
				req := httptest.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(body))
				req.Header.Set("Content-Type", "application/json")
				w := httptest.NewRecorder()

				router.ServeHTTP(w, req)
				assert.Equal(t, tc.want, w.Code)
			})
		}
	})

	t.Run("login with valid credentials", func(t *testing.T) {
		// First register a user
		registerReq := models.RegisterRequest{
			Email:    "login@example.com",
			Password: "password123",
			FullName: "Login User",
		}

		body, _ := json.Marshal(registerReq)
		req := httptest.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusCreated, w.Code)

		// Now login
		loginReq := models.LoginRequest{
			Email:    "login@example.com",
			Password: "password123",
		}

		body, _ = json.Marshal(loginReq)
		req = httptest.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response models.LoginResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.NotEmpty(t, response.Token)
		assert.NotNil(t, response.User)
		assert.Equal(t, loginReq.Email, response.User.Email)
	})

	t.Run("login with invalid credentials fails", func(t *testing.T) {
		loginReq := models.LoginRequest{
			Email:    "nonexistent@example.com",
			Password: "wrongpassword",
		}

		body, _ := json.Marshal(loginReq)
		req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

// TestUserEndpoints tests user management endpoints
func TestUserEndpoints(t *testing.T) {
	router := setupTestRouter(t)
	defer teardownTest(t)

	// Register and login to get a token
	registerReq := models.RegisterRequest{
		Email:    "user@example.com",
		Password: "password123",
		FullName: "Test User",
	}

	body, _ := json.Marshal(registerReq)
	req := httptest.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var userResponse models.UserResponse
	err := json.Unmarshal(w.Body.Bytes(), &userResponse)
	require.NoError(t, err)
	userID := userResponse.ID

	// Login to get token
	loginReq := models.LoginRequest{
		Email:    "user@example.com",
		Password: "password123",
	}

	body, _ = json.Marshal(loginReq)
	req = httptest.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var loginResponse models.LoginResponse
	err = json.Unmarshal(w.Body.Bytes(), &loginResponse)
	require.NoError(t, err)
	token := loginResponse.Token

	t.Run("list users requires authentication", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/users", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("list users with authentication", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/users", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var users []models.UserResponse
		err := json.Unmarshal(w.Body.Bytes(), &users)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(users), 1)
	})

	t.Run("list users with pagination", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/users?limit=5&offset=0", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var users []models.UserResponse
		err := json.Unmarshal(w.Body.Bytes(), &users)
		require.NoError(t, err)
		assert.LessOrEqual(t, len(users), 5)
	})

	t.Run("get user by id requires authentication", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/users/"+userID.String(), nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("get user by id with authentication", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/users/"+userID.String(), nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var user models.UserResponse
		err := json.Unmarshal(w.Body.Bytes(), &user)
		require.NoError(t, err)
		assert.Equal(t, userID, user.ID)
		assert.Equal(t, registerReq.Email, user.Email)
	})

	t.Run("get non-existent user returns 404", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/users/00000000-0000-0000-0000-000000000000", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("create user requires authentication", func(t *testing.T) {
		createReq := models.RegisterRequest{
			Email:    "newuser@example.com",
			Password: "password123",
			FullName: "New User",
		}

		body, _ := json.Marshal(createReq)
		req := httptest.NewRequest("POST", "/api/users", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("create user with authentication", func(t *testing.T) {
		createReq := models.RegisterRequest{
			Email:    "newuser@example.com",
			Password: "password123",
			FullName: "New User",
		}

		body, _ := json.Marshal(createReq)
		req := httptest.NewRequest("POST", "/api/users", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var user models.UserResponse
		err := json.Unmarshal(w.Body.Bytes(), &user)
		require.NoError(t, err)
		assert.Equal(t, createReq.Email, user.Email)
		assert.Equal(t, createReq.FullName, user.FullName)
	})

	t.Run("update user requires authentication", func(t *testing.T) {
		updateReq := models.UpdateUserRequest{
			Email:    "updated@example.com",
			FullName: "Updated User",
		}

		body, _ := json.Marshal(updateReq)
		req := httptest.NewRequest("PUT", "/api/users/"+userID.String(), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("update user with authentication", func(t *testing.T) {
		updateReq := models.UpdateUserRequest{
			Email:    "updated@example.com",
			FullName: "Updated User",
		}

		body, _ := json.Marshal(updateReq)
		req := httptest.NewRequest("PUT", "/api/users/"+userID.String(), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var user models.UserResponse
		err := json.Unmarshal(w.Body.Bytes(), &user)
		require.NoError(t, err)
		assert.Equal(t, updateReq.Email, user.Email)
		assert.Equal(t, updateReq.FullName, user.FullName)
	})

	t.Run("delete user requires authentication", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/users/"+userID.String(), nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("delete user with authentication", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/users/"+userID.String(), nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)

		// Verify user is deleted
		req2 := httptest.NewRequest("GET", "/api/users/"+userID.String(), nil)
		req2.Header.Set("Authorization", "Bearer "+token)
		w2 := httptest.NewRecorder()

		router.ServeHTTP(w2, req2)
		assert.Equal(t, http.StatusNotFound, w2.Code)
	})
}
