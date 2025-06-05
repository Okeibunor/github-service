package app

import (
	"encoding/json"
	"net/http"
)

type ResponseStatus string

const (
	StatusSuccess ResponseStatus = "success"
	StatusError   ResponseStatus = "error"
	StatusFail    ResponseStatus = "fail"
)

type Response struct {
	Status  ResponseStatus `json:"status"`
	Message string         `json:"message"`
	Data    interface{}    `json:"data,omitempty"`
}

// respondWithJSON sends a JSON response with the given status code and payload
func respondWithJSON(w http.ResponseWriter, code int, response Response) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(response)
}

func NewSuccessResponse(message string, data interface{}) Response {
	return Response{
		Status:  StatusSuccess,
		Message: message,
		Data:    data,
	}
}

func NewErrorResponse(message string) Response {
	return Response{
		Status:  StatusError,
		Message: message,
	}
}

func NewFailResponse(message string) Response {
	return Response{
		Status:  StatusFail,
		Message: message,
	}
}
