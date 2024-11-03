package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/mohammedrefaat/smart-ai-assistant/config"
)

// Global database instance
var db *DB

// Document represents a document in the knowledge base
type Document struct {
	ID        int       `db:"id"`
	DocID     string    `db:"doc_id"`
	Content   string    `db:"content"`
	Embedding []float64 `db:"embedding"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

// DB wraps sqlx.DB to provide custom functionality
type DB struct {
	Sdb *sqlx.DB
	cfg *config.Config
}

// DatabaseConfig holds database configuration

// ChatRequest represents the incoming chat request
type ChatRequest struct {
	Query string `json:"query"`
}

// ChatResponse represents the outgoing chat response
type ChatResponse struct {
	Response string   `json:"response"`
	Sources  []string `json:"sources,omitempty"`
}

// chatHandler processes incoming chat requests
func chatHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		fmt.Println("Error decoding request:", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	queryEmbedding, err := generateEmbedding(req.Query)
	if err != nil {
		http.Error(w, "Failed to generate embedding", http.StatusInternalServerError)
		return
	}

	docs, err := QuerySimilarDocuments(context.Background(), queryEmbedding, 10, 0.5, db)
	if err != nil {
		http.Error(w, "Failed to retrieve context", http.StatusInternalServerError)
		return
	}

	var contexts []string
	var sources []string
	for _, doc := range docs {
		contexts = append(contexts, doc.Content)
		sources = append(sources, doc.DocID)
	}

	prompt := buildPrompt(contexts, req.Query)
	response, err := generateText(prompt)
	if err != nil {
		http.Error(w, "Failed to generate response", http.StatusInternalServerError)
		return
	}

	chatResp := ChatResponse{
		Response: response,
		Sources:  sources,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(chatResp)
}

// buildPrompt creates the prompt for text generation
func buildPrompt(contexts []string, query string) string {
	return fmt.Sprintf(`Use the following information to answer the question:

Context:
%s

Question: %s

Answer:`, strings.Join(contexts, "\n\n"), query)
}
