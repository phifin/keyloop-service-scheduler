package handlers

import (
	"encoding/json"
	"net/http"
)

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, statusCode int, code, message string) {
	writeJSON(w, statusCode, ErrorResponse{
		Error:   code,
		Message: message,
	})
}

func writeErrorWithCode(w http.ResponseWriter, statusCode int, errorCode, message, code string) {
	writeJSON(w, statusCode, ErrorResponse{
		Error:   errorCode,
		Message: message,
		Code:    code,
	})
}
