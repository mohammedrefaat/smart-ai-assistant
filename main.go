// File: main.go

package main

import (
	"log"
	"os"

	"github.com/mohammedrefaat/smart-ai-assistant/assistant"
	"github.com/mohammedrefaat/smart-ai-assistant/web"
)

func main() {
	// Initialize configuration
	config := assistant.Config{
		ModelPath:    "./models",
		CachePath:    "./cache",
		VectorDBPath: "./vectordb.gob",
		MaxTokens:    2000,
		Temperature:  0.7,
		EmbeddingDim: 300,
		UseOnline:    true,
		MaxCacheSize: 1 << 30, // 1GB
		LearningRate: 0.1,
	}

	// Create required directories
	requiredDirs := []string{"./models", "./cache", "./uploads", "./static"}
	for _, dir := range requiredDirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Fatalf("Failed to create directory %s: %v", dir, err)
		}
	}

	// Initialize assistant
	smartAssistant, err := assistant.NewSmartAssistant("SmartAI", config)
	if err != nil {
		log.Fatalf("Failed to initialize assistant: %v", err)
	}

	// Initialize web GUI
	gui, err := web.NewWebGUI(smartAssistant)
	if err != nil {
		log.Fatalf("Failed to initialize web GUI: %v", err)
	}

	// Start web server
	log.Printf("Starting server on http://localhost:8080")
	if err := gui.Start(8080); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
