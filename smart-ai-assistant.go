package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/jdkato/prose/v2"
	"github.com/schollz/closestmatch"
	"github.com/tebeka/snowball"
	"gonum.org/v1/gonum/mat"
)

// SmartAssistant represents the enhanced AI assistant
type SmartAssistant struct {
	Name            string
	Brain           *EnhancedBrain
	LearningManager *LearningManager
	VectorDB        *EnhancedVectorDB
	Cache           *Cache
	Config          Config
	httpClient      *http.Client
}

// Config stores all configuration settings
type Config struct {
	ModelPath      string
	CachePath      string
	VectorDBPath   string
	MaxTokens      int
	Temperature    float32
	EmbeddingDim   int
	UseOnline      bool
	MaxCacheSize   int64
	LearningRate   float32
}

// EnhancedBrain manages advanced language processing
type EnhancedBrain struct {
	Embeddings     *WordEmbeddings
	Patterns       *PatternMatcher
	Tokenizer      *NLPProcessor
	SentenceParser *prose.Document
	mu             sync.RWMutex
}

// WordEmbeddings manages word vectors
type WordEmbeddings struct {
	Vectors    map[string][]float32
	Dimension  int
	Stemmer    *snowball.Stemmer
}

// PatternMatcher handles pattern recognition
type PatternMatcher struct {
	Patterns     []Pattern
	Matcher      *closestmatch.ClosestMatch
	MinConfidence float32
}

// Pattern represents a learned pattern
type Pattern struct {
	Input      string
	InputEmbed []float32
	Response   string
	Context    string
	Confidence float32
	Source     string
	Timestamp  time.Time
}

// LearningManager handles online and offline learning
type LearningManager struct {
	brain      *EnhancedBrain
	vectorDB   *EnhancedVectorDB
	cache      *Cache
	httpClient *http.Client
}

// EnhancedVectorDB manages vector storage and search
type EnhancedVectorDB struct {
	Vectors      []Vector
	Index        *mat.Dense
	Path         string
	mu           sync.RWMutex
}

// Cache manages local storage of learned data
type Cache struct {
	Data       map[string]CacheEntry
	Path       string
	MaxSize    int64
	CurrentSize int64
	mu         sync.RWMutex
}

type CacheEntry struct {
	Content   string
	Embedding []float32
	Source    string
	Timestamp time.Time
	Size      int64
}

// NLPProcessor handles text processing
type NLPProcessor struct {
	Document     *prose.Document
	Stemmer      *snowball.Stemmer
	StopWords    map[string]bool
}

// Vector represents a semantic vector
type Vector struct {
	ID        int
	Content   string
	Embedding []float32
	Type      string
	Source    string
	Timestamp time.Time
}

// Initialize new SmartAssistant
func NewSmartAssistant(name string, config Config) (*SmartAssistant, error) {
	// Create directories
	os.MkdirAll(config.ModelPath, 0755)
	os.MkdirAll(config.CachePath, 0755)
	
	// Initialize components
	brain, err := NewEnhancedBrain(config)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize brain: %v", err)
	}
	
	vectorDB, err := NewEnhancedVectorDB(config)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize vector DB: %v", err)
	}
	
	cache := NewCache(config.CachePath, config.MaxCacheSize)
	
	assistant := &SmartAssistant{
		Name:     name,
		Brain:    brain,
		VectorDB: vectorDB,
		Cache:    cache,
		Config:   config,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
	
	assistant.LearningManager = NewLearningManager(brain, vectorDB, cache, assistant.httpClient)
	
	return assistant, nil
}

// Process input and generate response
func (sa *SmartAssistant) ProcessInput(ctx context.Context, input string) (string, error) {
	// Preprocess input
	processedInput, err := sa.Brain.Tokenizer.ProcessText(input)
	if err != nil {
		return "", fmt.Errorf("failed to process input: %v", err)
	}
	
	// Get input embedding
	inputEmbed := sa.Brain.Embeddings.GetEmbedding(processedInput)
	
	// Search for similar patterns
	patterns := sa.Brain.Patterns.FindSimilarPatterns(inputEmbed)
	
	var response string
	if len(patterns) > 0 && patterns[0].Confidence > sa.Brain.Patterns.MinConfidence {
		// Use existing pattern
		response = patterns[0].Response
	} else if sa.Config.UseOnline {
		// Learn from online sources
		learned, err := sa.LearningManager.LearnFromOnline(ctx, input)
		if err != nil {
			return "", fmt.Errorf("failed to learn online: %v", err)
		}
		response = learned
	} else {
		// Generate response from local knowledge
		response = sa.generateLocalResponse(processedInput)
	}
	
	// Update patterns with new input-response pair
	sa.Brain.Patterns.AddPattern(Pattern{
		Input:      input,
		InputEmbed: inputEmbed,
		Response:   response,
		Timestamp:  time.Now(),
		Confidence: 1.0,
	})
	
	return response, nil
}

// LearnFromOnline searches and learns from web content
func (lm *LearningManager) LearnFromOnline(ctx context.Context, query string) (string, error) {
	// Check cache first
	if cached, exists := lm.cache.Get(query); exists {
		return cached.Content, nil
	}
	
	// Search and scrape relevant content
	urls := lm.searchRelevantURLs(query)
	var allContent []string
	
	for _, url := range urls {
		content, err := lm.scrapeContent(url)
		if err != nil {
			continue
		}
		
		// Process and store content
		processed, err := lm.brain.Tokenizer.ProcessText(content)
		if err != nil {
			continue
		}
		
		embedding := lm.brain.Embeddings.GetEmbedding(processed)
		
		// Store in vector DB and cache
		lm.vectorDB.Add(Vector{
			Content:   processed,
			Embedding: embedding,
			Source:    url,
			Timestamp: time.Now(),
		})
		
		lm.cache.Add(query, CacheEntry{
			Content:   processed,
			Embedding: embedding,
			Source:    url,
			Timestamp: time.Now(),
		})
		
		allContent = append(allContent, processed)
	}
	
	// Generate response from learned content
	response := lm.generateResponse(allContent)
	return response, nil
}

// Enhanced vector search
func (vdb *EnhancedVectorDB) Search(embedding []float32, limit int) []Vector {
	vdb.mu.RLock()
	defer vdb.mu.RUnlock()
	
	// Convert search embedding to matrix
	searchVec := mat.NewDense(1, len(embedding), float64Slice(embedding))
	
	// Calculate similarities using matrix multiplication
	similarities := mat.NewDense(1, len(vdb.Vectors), nil)
	similarities.Mul(searchVec, vdb.Index.T())
	
	// Get top results
	type searchResult struct {
		vector Vector
		score  float64
	}
	
	results := make([]searchResult, len(vdb.Vectors))
	for i := range vdb.Vectors {
		results[i] = searchResult{
			vector: vdb.Vectors[i],
			score:  similarities.At(0, i),
		}
	}
	
	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})
	
	vectors := make([]Vector, 0, limit)
	for i := 0; i < limit && i < len(results); i++ {
		vectors = append(vectors, results[i].vector)
	}
	
	return vectors
}

// Cache management
func (c *Cache) Add(key string, entry CacheEntry) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	// Check size limit
	if c.CurrentSize+entry.Size > c.MaxSize {
		c.evictOldEntries()
	}
	
	c.Data[key] = entry
	c.CurrentSize += entry.Size
	
	return c.save()
}

func (c *Cache) evictOldEntries() {
	type cacheItem struct {
		key       string
		timestamp time.Time
	}
	
	items := make([]cacheItem, 0, len(c.Data))
	for k, v := range c.Data {
		items = append(items, cacheItem{k, v.Timestamp})
	}
	
	sort.Slice(items, func(i, j int) bool {
		return items[i].timestamp.Before(items[j].timestamp)
	})
	
	// Remove oldest entries until under size limit
	for _, item := range items {
		if c.CurrentSize <= c.MaxSize*80/100 { // Keep 20% buffer
			break
		}
		entry := c.Data[item.key]
		c.CurrentSize -= entry.Size
		delete(c.Data, item.key)
	}
}

// Main function
func main() {
	homeDir, _ := os.UserHomeDir()
	config := Config{
		ModelPath:    filepath.Join(homeDir, ".ai-assistant/models"),
		CachePath:    filepath.Join(homeDir, ".ai-assistant/cache"),
		VectorDBPath: filepath.Join(homeDir, ".ai-assistant/vectordb.gob"),
		MaxTokens:    2000,
		Temperature:  0.7,
		EmbeddingDim: 300,
		UseOnline:    true,
		MaxCacheSize: 1 << 30, // 1GB
		LearningRate: 0.1,
	}

	assistant, err := NewSmartAssistant("SmartAI", config)
	if err != nil {
		fmt.Printf("Error initializing assistant: %v\n", err)
		return
	}

	scanner := bufio.NewScanner(os.Stdin)
	ctx := context.Background()

	fmt.Printf("%s: Hello! I'm ready to learn and help. What would you like to know?\n", assistant.Name)

	for {
		fmt.Print("You: ")
		if !scanner.Scan() {
			break
		}

		input := scanner.Text()
		if strings.ToLower(input) == "exit" {
			break
		}

		response, err := assistant.ProcessInput(ctx, input)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		fmt.Printf("%s: %s\n", assistant.Name, response)
	}
}

// Utility functions
func float64Slice(f32 []float32) []float64 {
	f64 := make([]float64, len(f32))
	for i, v := range f32 {
		f64[i] = float64(v)
	}
	return f64
}
