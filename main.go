package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	// Initialize database
	var err error
	db, err := initPostgres()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize the server with the database connection
	initServer(db)

	// Initialize knowledge ingester
	youtubeAPIKey := os.Getenv("YOUTUBE_API_KEY")
	ingester, err := NewIngester(db, youtubeAPIKey)
	if err != nil {
		log.Fatalf("Failed to initialize ingester: %v", err)
	}

	// Start the ingester
	ingester.Start()
	defer ingester.Stop()

	// Add source handling endpoints
	http.HandleFunc("/api/source", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var source struct {
			Type     string `json:"type"`
			URL      string `json:"url"`
			Schedule string `json:"schedule"`
		}

		if err := json.NewDecoder(r.Body).Decode(&source); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if err := ingester.AddSource(source.Type, source.URL, source.Schedule); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
	})

	// Existing chat handler
	http.HandleFunc("/chat", chatHandler)

	fmt.Println("Server started at :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}
