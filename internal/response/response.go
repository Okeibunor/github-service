package response

import (
	"encoding/json"
	"net/http"
)

// Response represents a standard API response
type Response struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// PaginatedResponse represents a paginated API response
type PaginatedResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Meta    Pagination  `json:"meta"`
}

// Pagination contains pagination metadata
type Pagination struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	TotalItems int `json:"total_items"`
	TotalPages int `json:"total_pages"`
}

// Success creates a successful response
func Success(message string, data interface{}) Response {
	return Response{
		Status:  "success",
		Message: message,
		Data:    data,
	}
}

// SuccessPaginated creates a successful paginated response
func SuccessPaginated(message string, data interface{}, page, perPage, totalItems int) PaginatedResponse {
	totalPages := (totalItems + perPage - 1) / perPage // Ceiling division
	return PaginatedResponse{
		Status:  "success",
		Message: message,
		Data:    data,
		Meta: Pagination{
			Page:       page,
			PerPage:    perPage,
			TotalItems: totalItems,
			TotalPages: totalPages,
		},
	}
}

// Error creates an error response
func Error(message string) Response {
	return Response{
		Status:  "error",
		Message: message,
	}
}

// JSON writes a JSON response with the given status code
func JSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}
