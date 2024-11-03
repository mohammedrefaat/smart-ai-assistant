package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/jmoiron/sqlx"
	"github.com/ledongthuc/pdf"
	"github.com/mmcdole/gofeed"
	"github.com/robfig/cron/v3" // Add cron package import
	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
)

// Knowledge source types
const (
	SourceTypeAPI     = "api"
	SourceTypeLink    = "link"
	SourceTypePDF     = "pdf"
	SourceTypeYouTube = "youtube"
	SourceTypeRSS     = "rss"
	SourceTypeText    = "text"
)

// Source represents a knowledge source configuration
type Source struct {
	ID          string    `db:"id"`
	Type        string    `db:"type"`
	URL         string    `db:"url"`
	Schedule    string    `db:"schedule"` // Cron expression
	LastUpdated time.Time `db:"last_updated"`
	Active      bool      `db:"active"`
}

// Content represents processed content from any source
type Content struct {
	Title       string
	Text        string
	Source      string
	URL         string
	PublishedAt time.Time
}

type Ingester struct {
	db           *sqlx.DB
	apiProcessor *APIProcessor
	webProcessor *WebProcessor
	pdfProcessor *PDFProcessor
	ytProcessor  *YouTubeProcessor
	rssProcessor *RSSProcessor
	cron         *cron.Cron
}

// APIProcessor processes REST API endpoints
type APIProcessor struct {
	client *http.Client
}

func (p *APIProcessor) Fetch(url string) ([]Content, error) {
	resp, err := p.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	// Convert the API response to content
	// This is a simple example - adjust based on your API structure
	content := Content{
		Title:       fmt.Sprintf("API Data from %s", url),
		Text:        fmt.Sprintf("%v", data),
		Source:      url,
		URL:         url,
		PublishedAt: time.Now(),
	}

	return []Content{content}, nil
}

// WebProcessor processes web links
type WebProcessor struct {
	client *http.Client
}

func (p *WebProcessor) Fetch(url string) ([]Content, error) {
	resp, err := p.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	// Extract main content
	var textContent strings.Builder
	doc.Find("article, main, .content, #content").Each(func(i int, s *goquery.Selection) {
		textContent.WriteString(s.Text())
	})

	content := Content{
		Title:       doc.Find("title").Text(),
		Text:        textContent.String(),
		Source:      url,
		URL:         url,
		PublishedAt: time.Now(),
	}

	return []Content{content}, nil
}

// PDFProcessor processes PDF files
type PDFProcessor struct{}

func (p *PDFProcessor) Fetch(filepath string) ([]Content, error) {
	f, r, err := pdf.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var content strings.Builder
	totalPage := r.NumPage()

	for pageIndex := 1; pageIndex <= totalPage; pageIndex++ {
		page := r.Page(pageIndex)
		if page.V.IsNull() {
			continue
		}
		text, err := page.GetPlainText(nil)
		if err != nil {
			continue
		}
		content.WriteString(text)
	}

	return []Content{{
		Title:       filepath,
		Text:        content.String(),
		Source:      filepath,
		URL:         filepath,
		PublishedAt: time.Now(),
	}}, nil
}

// YouTubeProcessor processes YouTube videos using captions/transcripts
type YouTubeProcessor struct {
	service *youtube.Service
}

func NewYouTubeProcessor(apiKey string) (*YouTubeProcessor, error) {
	ctx := context.Background()
	service, err := youtube.NewService(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, err
	}
	return &YouTubeProcessor{service: service}, nil
}

func (p *YouTubeProcessor) Fetch(videoID string) ([]Content, error) {
	// Get video details
	call := p.service.Videos.List([]string{"snippet"}).Id(videoID)
	response, err := call.Do()
	if err != nil {
		return nil, err
	}

	if len(response.Items) == 0 {
		return nil, fmt.Errorf("video not found")
	}

	video := response.Items[0]

	// Get captions (Note: This is simplified - you'll need to implement caption fetching)
	// You might want to use youtube-dl or a similar tool for actual caption fetching

	content := Content{
		Title:       video.Snippet.Title,
		Text:        video.Snippet.Description, // In reality, you'd want to add captions here
		Source:      "youtube",
		URL:         fmt.Sprintf("https://youtube.com/watch?v=%s", videoID),
		PublishedAt: time.Now(),
	}

	return []Content{content}, nil
}

// RSSProcessor processes RSS feeds
type RSSProcessor struct {
	parser *gofeed.Parser
}

func (p *RSSProcessor) Fetch(feedURL string) ([]Content, error) {
	feed, err := p.parser.ParseURL(feedURL)
	if err != nil {
		return nil, err
	}

	var contents []Content
	for _, item := range feed.Items {
		publishedAt := time.Now()
		if item.PublishedParsed != nil {
			publishedAt = *item.PublishedParsed
		}

		content := Content{
			Title:       item.Title,
			Text:        item.Description,
			Source:      feed.Title,
			URL:         item.Link,
			PublishedAt: publishedAt,
		}
		contents = append(contents, content)
	}

	return contents, nil
}

func NewIngester(db *DB, youtubeAPIKey string) (*Ingester, error) {
	ytProcessor, err := NewYouTubeProcessor(youtubeAPIKey)
	if err != nil {
		return nil, err
	}

	return &Ingester{
		db:           db.Sdb,
		apiProcessor: &APIProcessor{client: http.DefaultClient},
		webProcessor: &WebProcessor{client: http.DefaultClient},
		pdfProcessor: &PDFProcessor{},
		ytProcessor:  ytProcessor,
		rssProcessor: &RSSProcessor{parser: gofeed.NewParser()},
		cron:         cron.New(cron.WithSeconds()),
	}, nil
}

func (i *Ingester) Start() {
	i.cron.Start()

	// Schedule periodic source checks
	i.cron.AddFunc("*/15 * * * *", func() { // Every 15 minutes
		i.processActiveSources()
	})
}

func (i *Ingester) Stop() {
	i.cron.Stop()
}

func (i *Ingester) processActiveSources() {
	sources, err := i.getActiveSources()
	if err != nil {
		log.Printf("Error getting active sources: %v", err)
		return
	}

	var wg sync.WaitGroup
	for _, source := range sources {
		wg.Add(1)
		go func(src Source) {
			defer wg.Done()
			if err := i.processSource(src); err != nil {
				log.Printf("Error processing source %s: %v", src.ID, err)
			}
		}(source)
	}
	wg.Wait()
}

func (i *Ingester) processSource(source Source) error {
	var contents []Content
	var err error

	switch source.Type {
	case SourceTypeAPI:
		contents, err = i.apiProcessor.Fetch(source.URL)
	case SourceTypeLink:
		contents, err = i.webProcessor.Fetch(source.URL)
	case SourceTypePDF:
		contents, err = i.pdfProcessor.Fetch(source.URL)
	case SourceTypeYouTube:
		contents, err = i.ytProcessor.Fetch(source.URL)
	case SourceTypeRSS:
		contents, err = i.rssProcessor.Fetch(source.URL)
	default:
		return fmt.Errorf("unknown source type: %s", source.Type)
	}

	if err != nil {
		return err
	}

	// Process each piece of content
	for _, content := range contents {
		// Generate embedding
		embedding, err := generateEmbedding(content.Text)
		if err != nil {
			log.Printf("Error generating embedding for content from %s: %v", source.URL, err)
			continue
		}

		// Add document to database
		err = db.AddDocument(context.Background(), fmt.Sprintf("%s-%d", source.ID, time.Now().UnixNano()), content.Text, embedding)

		if err != nil {
			log.Printf("Error adding document from %s: %v", source.URL, err)
			continue
		}
	}

	// Update last processed time
	return i.updateSourceLastUpdated(source.ID)
}

func (i *Ingester) getActiveSources() ([]Source, error) {
	var sources []Source
	query := `SELECT * FROM knowledge_sources WHERE active = true`
	err := i.db.Select(&sources, query)
	return sources, err
}

func (i *Ingester) updateSourceLastUpdated(sourceID string) error {
	query := `UPDATE knowledge_sources SET last_updated = NOW() WHERE id = $1`
	_, err := i.db.Exec(query, sourceID)
	return err
}

// AddSource adds a new knowledge source
func (i *Ingester) AddSource(sourceType, url, schedule string) error {
	query := `
        INSERT INTO knowledge_sources (id, type, url, schedule)
        VALUES ($1, $2, $3, $4)`

	sourceID := fmt.Sprintf("%s-%d", sourceType, time.Now().UnixNano())
	_, err := i.db.Exec(query, sourceID, sourceType, url, schedule)
	return err
}
