package main

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/mohammedrefaat/smart-ai-assistant/config"
)

// InitPostgres creates a new database connection
func InitPostgres(cfg *config.Config) (*DB, error) {
	db, err := sqlx.Connect("postgres", cfg.Database.GetDatabaseURL())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	db.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	db.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	db.SetConnMaxLifetime(time.Duration(cfg.Database.Timeout))

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.Database.Timeout))
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}
	if err := initializeExtensions(db); err != nil {
		return nil, fmt.Errorf("failed to initialize extensions: %w", err)
	}

	if err := initializeSchema(db); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return &DB{Sdb: db, cfg: cfg}, nil
}

// initializeSchema creates the necessary tables if they don't exist
func initializeSchema(db *sqlx.DB) error {
	schema := `
		CREATE TABLE IF NOT EXISTS knowledge_base (
			id SERIAL PRIMARY KEY,
			doc_id VARCHAR(255) UNIQUE NOT NULL,
			content TEXT NOT NULL,
			embedding vector(1536),
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);

		CREATE INDEX IF NOT EXISTS idx_knowledge_base_doc_id ON knowledge_base(doc_id);
		CREATE INDEX IF NOT EXISTS idx_knowledge_base_created_at ON knowledge_base(created_at);
		CREATE INDEX IF NOT EXISTS idx_knowledge_base_embedding ON knowledge_base USING ivfflat (embedding vector_cosine_ops)
			WITH (lists = 100);
	`

	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	return nil
}

// initializeExtensions ensures required PostgreSQL extensions are installed
func initializeExtensions(db *sqlx.DB) error {
	// Create the vector extension if it doesn't exist
	_, err := db.Exec(`CREATE EXTENSION IF NOT EXISTS vector`)
	if err != nil {
		return fmt.Errorf("failed to create vector extension: %w", err)
	}

	return nil
}

// AddDocument adds or updates a document in the knowledge base
func (db *DB) AddDocument(ctx context.Context, docID string, content string, embedding []float64) error {
	query := `
		INSERT INTO knowledge_base (doc_id, content, embedding, created_at, updated_at)
		VALUES ($1, $2, $3, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT (doc_id) 
		DO UPDATE SET 
			content = EXCLUDED.content, 
			embedding = EXCLUDED.embedding,
			updated_at = CURRENT_TIMESTAMP
		RETURNING id, created_at, updated_at`

	var doc Document
	err := db.Sdb.QueryRowxContext(ctx, query,
		docID,
		content,
		pq.Array(embedding),
	).Scan(&doc.ID, &doc.CreatedAt, &doc.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to insert document: %w", err)
	}

	return nil
}

// QuerySimilarDocuments finds similar documents using vector similarity
func (db *DB) querySimilarDocuments(ctx context.Context, embedding []float64, topK int, similarityThreshold float64) ([]Document, error) {
	query := `
		SELECT id, doc_id, content, embedding::float[], created_at, updated_at
		FROM knowledge_base
		WHERE 1 - (embedding <=> $1) >= $3
		ORDER BY embedding <=> $1
		LIMIT $2`

	var documents []Document
	err := db.Sdb.SelectContext(ctx, &documents, query,
		pq.Array(embedding),
		topK,
		similarityThreshold,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query similar documents: %w", err)
	}

	return documents, nil
}
func QuerySimilarDocuments(ctx context.Context, embedding []float64, topK int, similarityThreshold float64, db *DB) ([]Document, error) {
	return db.querySimilarDocuments(ctx, embedding, topK, similarityThreshold)
}

// DeleteOldDocuments removes documents older than the specified retention period
func (db *DB) DeleteOldDocuments(ctx context.Context, retentionDays int) (int64, error) {
	query := `
		DELETE FROM knowledge_base 
		WHERE created_at < NOW() - INTERVAL '1 day' * $1
		RETURNING id`

	result, err := db.Sdb.ExecContext(ctx, query, retentionDays)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old documents: %w", err)
	}

	deletedCount, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get deleted count: %w", err)
	}

	return deletedCount, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.Sdb.Close()
}

// GetDocumentByID retrieves a document by its doc_id
func (db *DB) GetDocumentByID(ctx context.Context, docID string) (*Document, error) {
	var doc Document
	query := `
		SELECT id, doc_id, content, embedding::float[], created_at, updated_at
		FROM knowledge_base
		WHERE doc_id = $1`

	err := db.Sdb.GetContext(ctx, &doc, query, docID)
	if err != nil {
		return nil, fmt.Errorf("failed to get document: %w", err)
	}

	return &doc, nil
}

// CountDocuments returns the total number of documents in the knowledge base
func (db *DB) CountDocuments(ctx context.Context) (int64, error) {
	var count int64
	err := db.Sdb.GetContext(ctx, &count, "SELECT COUNT(*) FROM knowledge_base")
	if err != nil {
		return 0, fmt.Errorf("failed to count documents: %w", err)
	}

	return count, nil
}
