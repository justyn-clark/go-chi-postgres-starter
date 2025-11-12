package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/yourusername/go-chi-postgres-starter/cmd/api/database"
	"github.com/yourusername/go-chi-postgres-starter/cmd/api/models"
)

// HealthHandler handles health check requests
type HealthHandler struct {
	db *database.DB
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(db *database.DB) *HealthHandler {
	return &HealthHandler{db: db}
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

var startTime = time.Now()

// Health returns the health status of the API
// @Summary Health check
// @Description Check if the API is running and database is accessible
// @Tags Health
// @Accept json
// @Produce json
// @Success 200 {object} models.HealthResponse
// @Failure 503 {object} models.HealthResponse "Service unavailable if database is down"
// @Router /api/health [get]
func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	response := models.HealthResponse{
		Status:    "ok",
		Service:   "go-chi-postgres-starter",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Uptime:    time.Since(startTime).String(),
	}

	// Check database health if database is available
	if h.db != nil {
		dbHealth := h.checkDatabaseHealth(r.Context())
		response.Database = &dbHealth

		// If database is unhealthy, return 503
		if dbHealth.Status != "ok" {
			response.Status = "degraded"
			respondJSON(w, http.StatusServiceUnavailable, response)
			return
		}
	}

	respondJSON(w, http.StatusOK, response)
}

// checkDatabaseHealth checks the database connection health
func (h *HealthHandler) checkDatabaseHealth(ctx context.Context) models.DatabaseHealth {
	start := time.Now()
	err := h.db.Health(ctx)
	duration := time.Since(start)

	if err != nil {
		return models.DatabaseHealth{
			Status:       "error",
			ResponseTime: duration.String(),
			Error:        err.Error(),
		}
	}

	return models.DatabaseHealth{
		Status:       "ok",
		ResponseTime: duration.String(),
	}
}
