package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/yourusername/go-chi-postgres-starter/cmd/api/queue"
)

// QueueHandler handles queue monitoring and management endpoints
type QueueHandler struct {
	queue queue.Queue
}

// NewQueueHandler creates a new queue handler
func NewQueueHandler(q queue.Queue) *QueueHandler {
	return &QueueHandler{queue: q}
}

// ListQueues returns all queue names and their stats
// @Summary List all queues
// @Description Get statistics for all queues (admin only)
// @Tags queue
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/admin/queues [get]
func (h *QueueHandler) ListQueues(w http.ResponseWriter, r *http.Request) {
	statsProvider, ok := h.queue.(queue.StatsProvider)
	if !ok {
		respondError(w, http.StatusNotImplemented, "queue does not support statistics")
		return
	}

	queueNames, err := statsProvider.ListQueues(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list queues: "+err.Error())
		return
	}

	// Get stats for each queue
	queues := make(map[string]*queue.QueueStats)
	for _, name := range queueNames {
		stats, err := statsProvider.GetStats(r.Context(), name)
		if err != nil {
			continue // Skip queues with errors
		}
		queues[name] = stats
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"queues": queues,
		"count":  len(queues),
	})
}

// GetQueueStats returns statistics for a specific queue
// @Summary Get queue statistics
// @Description Get statistics for a specific queue (admin only)
// @Tags queue
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param queueName path string true "Queue name"
// @Param peek query bool false "Peek at jobs (default: false)"
// @Success 200 {object} queue.QueueInfo
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/admin/queues/{queueName} [get]
func (h *QueueHandler) GetQueueStats(w http.ResponseWriter, r *http.Request) {
	queueName := chi.URLParam(r, "queueName")
	if queueName == "" {
		respondError(w, http.StatusBadRequest, "queue name required")
		return
	}

	statsProvider, ok := h.queue.(queue.StatsProvider)
	if !ok {
		respondError(w, http.StatusNotImplemented, "queue does not support statistics")
		return
	}

	// Check if peek parameter is set
	peek := r.URL.Query().Get("peek") == "true"

	// Try to get detailed info if queue supports QueueInfoProvider
	if infoProvider, ok := h.queue.(queue.QueueInfoProvider); ok {
		info, err := infoProvider.GetQueueInfo(r.Context(), queueName, peek)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "failed to get queue info: "+err.Error())
			return
		}
		respondJSON(w, http.StatusOK, info)
		return
	}

	// Fallback to basic stats
	stats, err := statsProvider.GetStats(r.Context(), queueName)
	if err != nil {
		respondError(w, http.StatusNotFound, "queue not found: "+queueName)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"stats":  stats,
		"peeked": false,
	})
}
