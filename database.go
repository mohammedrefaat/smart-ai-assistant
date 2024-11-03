package main

import (
	"fmt"
	"log"
	"os"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type Document struct {
	ID        int       `db:"id"`
	DocID     string    `db:"doc_id"`
	Content   string    `db:"content"`
	Embedding []float64 `db:"embedding"`
}

// Initialize PostgreSQL connection
func initPostgres() (*sqlx.DB, error) {
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		return nil, fmt.Errorf("DATABASE_URL environment variable not set")
	}

	db, err := sqlx.Connect("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Println("Connected to PostgreSQL!")
	return db, nil
}

// Add a document and its embedding to PostgreSQL
func addDocument(db *sqlx.DB, docID string, content string, embedding []float64) error {
	query := `
        INSERT INTO knowledge_base (doc_id, content, embedding) 
        VALUES ($1, $2, $3)
        ON CONFLICT (doc_id) 
        DO UPDATE SET content = EXCLUDED.content, embedding = EXCLUDED.embedding`

	_, err := db.Exec(query, docID, content, pq.Array(embedding))
	if err != nil {
		return fmt.Errorf("failed to insert document: %w", err)
	}
	return nil
}

// Query similar documents using vector similarity
func querySimilarDocuments(db *sqlx.DB, embedding []float64, topK int) ([]Document, error) {
	query := `
        SELECT id, doc_id, content, embedding::float[] 
        FROM knowledge_base
        ORDER BY embedding <-> $1
        LIMIT $2`

	var documents []Document
	err := db.Select(&documents, query, pq.Array(embedding), topK)
	if err != nil {
		return nil, fmt.Errorf("failed to query similar documents: %w", err)
	}

	return documents, nil
}

// Delete old documents
func deleteOldDocuments(db *sqlx.DB, daysOld int) error {
	query := `
        DELETE FROM knowledge_base 
        WHERE created_at < NOW() - INTERVAL '1 day' * $1`

	_, err := db.Exec(query, daysOld)
	if err != nil {
		return fmt.Errorf("failed to delete old documents: %w", err)
	}
	return nil
}
