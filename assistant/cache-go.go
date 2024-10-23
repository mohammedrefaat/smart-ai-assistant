// File: assistant/cache.go

package assistant

import (
	"io"
	"os"
	"sync"
	"time"
)

// Cache manages local storage of learned data
type Cache struct {
	Data        map[string]CacheEntry
	Path        string
	MaxSize     int64
	CurrentSize int64
	mu          sync.RWMutex
}

// CacheEntry represents a single cached item
type CacheEntry struct {
	Content   string
	Embedding []float32
	Source    string
	Timestamp time.Time
	Size      int64
}

// NewCache creates a new cache instance
func NewCache(path string, maxSize int64) *Cache {
	return &Cache{
		Data:    make(map[string]CacheEntry),
		Path:    path,
		MaxSize: maxSize,
	}
}

// Add stores a new entry in the cache
func (c *Cache) Add(key string, entry CacheEntry) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.CurrentSize+entry.Size > c.MaxSize {
		c.evictOldEntries()
	}

	c.Data[key] = entry
	c.CurrentSize += entry.Size
	return nil
}

// Get retrieves an entry from the cache
func (c *Cache) Get(key string) (CacheEntry, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.Data[key]
	return entry, exists
}

// LoadFile loads and caches a file's content
func (c *Cache) LoadFile(filepath string) (string, error) {
	// Check cache first
	if entry, exists := c.Get(filepath); exists {
		return entry.Content, nil
	}

	// Read file
	file, err := os.Open(filepath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}

	// Cache the content
	fileInfo, err := file.Stat()
	if err != nil {
		return "", err
	}

	entry := CacheEntry{
		Content:   string(content),
		Source:    filepath,
		Timestamp: time.Now(),
		Size:      fileInfo.Size(),
	}

	if err := c.Add(filepath, entry); err != nil {
		return "", err
	}

	return string(content), nil
}

// evictOldEntries removes old entries to free up space
func (c *Cache) evictOldEntries() {
	var oldestKey string
	var oldestTime time.Time

	// Find oldest entry
	for key, entry := range c.Data {
		if oldestKey == "" || entry.Timestamp.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.Timestamp
		}
	}

	// Remove oldest entry if found
	if oldestKey != "" {
		entry := c.Data[oldestKey]
		delete(c.Data, oldestKey)
		c.CurrentSize -= entry.Size
	}
}
