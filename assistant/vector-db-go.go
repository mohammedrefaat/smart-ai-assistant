// File: assistant/vector_db.go

package assistant

import (
	"encoding/gob"
	"os"
	"sync"
	"time"
)

// VectorDB manages vector storage and search
type VectorDB struct {
	Vectors []Vector
	Path    string
	mu      sync.RWMutex
}

// Vector represents a stored vector with metadata
type Vector struct {
	ID        int
	Content   string
	Embedding []float32
	Source    string
	Timestamp time.Time
}

// NewVectorDB creates a new vector database instance
func NewVectorDB(path string) *VectorDB {
	db := &VectorDB{
		Path: path,
	}
	db.load() // Load existing data if available
	return db
}

// Store adds a new vector to the database
func (db *VectorDB) Store(content, source string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	vector := Vector{
		ID:        len(db.Vectors),
		Content:   content,
		Source:    source,
		Timestamp: time.Now(),
	}

	db.Vectors = append(db.Vectors, vector)
	return db.save()
}

// Search finds similar vectors (placeholder implementation)
func (db *VectorDB) Search(query string, limit int) []Vector {
	db.mu.RLock()
	defer db.mu.RUnlock()

	// Simple implementation - return latest vectors
	if len(db.Vectors) <= limit {
		return db.Vectors
	}
	return db.Vectors[len(db.Vectors)-limit:]
}

// save persists the database to disk
func (db *VectorDB) save() error {
	file, err := os.Create(db.Path)
	if err != nil {
		return err
	}
	defer file.Close()

	return gob.NewEncoder(file).Encode(db.Vectors)
}

// load reads the database from disk
func (db *VectorDB) load() error {
	file, err := os.Open(db.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No existing database
		}
		return err
	}
	defer file.Close()

	return gob.NewDecoder(file).Decode(&db.Vectors)
}
