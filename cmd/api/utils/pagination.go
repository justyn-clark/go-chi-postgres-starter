package utils

import (
	"strconv"
)

// PaginationParams holds pagination parameters from request
type PaginationParams struct {
	Page   int
	Limit  int
	Offset int
}

// ParsePagination parses pagination parameters from query string
// Returns default values if not provided or invalid
func ParsePagination(pageStr, limitStr string) PaginationParams {
	page := 1
	limit := 10
	offset := 0

	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
			// Enforce max limit to prevent abuse
			if limit > 100 {
				limit = 100
			}
		}
	}

	offset = (page - 1) * limit

	return PaginationParams{
		Page:   page,
		Limit:  limit,
		Offset: offset,
	}
}

// PaginationResponse represents paginated response metadata
type PaginationResponse struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// NewPaginationResponse creates pagination metadata
func NewPaginationResponse(page, limit, total int) PaginationResponse {
	totalPages := (total + limit - 1) / limit // Ceiling division
	if totalPages < 1 {
		totalPages = 1
	}

	return PaginationResponse{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	}
}
