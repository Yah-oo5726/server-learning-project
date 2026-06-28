package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type errorResponse struct {
	Body string `json:"body"`
}

type successResponse struct {
	CleanedBody string `json:"cleaned_body"`
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	fmt.Println("Error:", message)
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

func cleanMessage(message string) string {
	censoredWords := []string{"kerfuffle", "sharbert", "fornax"}
	messageLower := strings.ToLower(message)
	wordsLower := strings.Fields(messageLower)
	words := strings.Fields(message)
	for i, word := range wordsLower {
		for _, censored := range censoredWords {
			if word == censored {
				words[i] = "****"
				break
			}
		}
	}
	return strings.Join(words, " ")
}
