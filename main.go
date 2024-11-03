package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/mohammedrefaat/smart-ai-assistant/config"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig("./config.json")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	db, err = InitPostgres(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Sdb.Close()

	http.HandleFunc("/chat", chatHandler)

	server := &http.Server{
		Addr:           ":8080",
		ReadTimeout:    time.Second * 30,
		WriteTimeout:   time.Second * 30,
		MaxHeaderBytes: 1 << 20,
	}

	fmt.Println("Server started at :8080")
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}
