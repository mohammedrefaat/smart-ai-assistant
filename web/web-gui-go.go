// File: web/gui.go

package web

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/mohammedrefaat/smart-ai-assistant/assistant"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

//go:embed templates
var content embed.FS

// WebGUI handles the web interface
type WebGUI struct {
	assistant   *assistant.SmartAssistant
	router      *mux.Router
	upgrader    websocket.Upgrader
	connections map[*websocket.Conn]bool
	connMutex   sync.RWMutex
	templates   *template.Template
}

// FileUpload represents an uploaded file
type FileUpload struct {
	Name     string    `json:"name"`
	Size     int64     `json:"size"`
	Type     string    `json:"type"`
	Uploaded time.Time `json:"uploaded"`
}

// NewWebGUI creates a new web interface instance
func NewWebGUI(assistant *assistant.SmartAssistant) (*WebGUI, error) {
	tmpl, err := template.ParseFS(content, "templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("failed to parse templates: %v", err)
	}

	gui := &WebGUI{
		assistant:   assistant,
		router:      mux.NewRouter(),
		connections: make(map[*websocket.Conn]bool),
		templates:   tmpl,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true // For development
			},
		},
	}

	gui.setupRoutes()
	return gui, nil
}

// Start the web server
func (gui *WebGUI) Start(port int) error {
	addr := fmt.Sprintf(":%d", port)
	return http.ListenAndServe(addr, gui.router)
}

// Setup routes for the web interface
func (gui *WebGUI) setupRoutes() {
	gui.router.HandleFunc("/", gui.handleIndex)
	gui.router.HandleFunc("/ws", gui.handleWebSocket)
	gui.router.HandleFunc("/api/chat", gui.handleChat).Methods("POST")
	gui.router.HandleFunc("/api/files", gui.handleFiles).Methods("GET", "POST")

	// Serve static files
	gui.router.PathPrefix("/static/").Handler(
		http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
}

// Handle index page
func (gui *WebGUI) handleIndex(w http.ResponseWriter, r *http.Request) {
	gui.templates.ExecuteTemplate(w, "index.html", nil)
}

// Handle WebSocket connections
func (gui *WebGUI) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := gui.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	gui.connMutex.Lock()
	gui.connections[conn] = true
	gui.connMutex.Unlock()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			break
		}

		response, err := gui.assistant.ProcessInput(r.Context(), string(message))
		if err != nil {
			log.Printf("Processing error: %v", err)
			continue
		}

		if err := conn.WriteMessage(websocket.TextMessage, []byte(response)); err != nil {
			break
		}
	}

	gui.connMutex.Lock()
	delete(gui.connections, conn)
	gui.connMutex.Unlock()
}

// Handle chat API requests
func (gui *WebGUI) handleChat(w http.ResponseWriter, r *http.Request) {
	var message struct {
		Content string `json:"content"`
	}

	if err := json.NewDecoder(r.Body).Decode(&message); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response, err := gui.assistant.ProcessInput(r.Context(), message.Content)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"response": response})
}

// Handle file uploads
func (gui *WebGUI) handleFiles(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		files, err := gui.listUploadedFiles()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(files)

	case http.MethodPost:
		file, header, err := r.FormFile("file")
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		defer file.Close()
	}
}

// listUploadedFiles returns a list of uploaded files
func (w *WebGUI) listUploadedFiles() ([]string, error) {
	// Implement logic to retrieve the list of uploaded files
	var files []string
	// Example logic (replace with actual implementation)
	files = append(files, "file1.pdf", "file2.docx")
	return files, nil
}
