package main

import (
	"encoding/json"
	"net/http"
)

type errorResponse struct {
	Body string `json:"body"`
}

type successResponse struct {
	Valid bool `json:"valid"`
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	response := errorResponse{Body: message}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	responseJSON, err := json.Marshal(response)
	if err != nil {
		http.Error(w, "Error marshalling error response", http.StatusInternalServerError)
		return
	}
	w.Write(responseJSON)
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	responseJSON, err := json.Marshal(payload)
	if err != nil {
		http.Error(w, "Error marshalling error response", http.StatusInternalServerError)
		return
	}
	w.Write(responseJSON)
}
