package utils

import (
	"encoding/json"
	"net/http"
)

func WriteError(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload)
}

type ErrorResponse struct {
	Error  string
	Reason string
}
