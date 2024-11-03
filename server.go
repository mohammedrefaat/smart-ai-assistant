package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/jmoiron/sqlx"
)

type ChatRequest struct {
	Query string `json:"query"`
}

type ChatResponse struct {
	Response string   `json:"response"`
	Sources  []string `json:"sources,omitempty"`
}

// Add db as a package-level variable
var db *sqlx.DB

// Initialize the db variable
func initServer(database *sqlx.DB) {
	db = database
}

func chatHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Generate embedding for the query
	queryEmbedding, err := generateEmbedding(req.Query)
	if err != nil {
		http.Error(w, "Failed to generate embedding", http.StatusInternalServerError)
		return
	}

	// Use the global db variable to retrieve similar documents
	docs, err := querySimilarDocuments(db, queryEmbedding, 3)
	if err != nil {
		http.Error(w, "Failed to retrieve context", http.StatusInternalServerError)
		return
	}

	// Build context from similar documents
	var contexts []string
	var sources []string
	for _, doc := range docs {
		contexts = append(contexts, doc.Content)
		sources = append(sources, doc.DocID)
	}

	// Build prompt with context
	prompt := fmt.Sprintf(`Use the following information to answer the question:

Context:
%s

Question: %s

Answer:`, strings.Join(contexts, "\n\n"), req.Query)

	// Generate response using Ollama
	response, err := generateText(prompt)
	if err != nil {
		http.Error(w, "Failed to generate response", http.StatusInternalServerError)
		return
	}

	// Send response
	chatResp := ChatResponse{
		Response: response,
		Sources:  sources,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(chatResp)
}
