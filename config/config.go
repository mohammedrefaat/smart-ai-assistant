package config

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type Config struct {
	Server   ServerConfig   `json:"server"`
	Database DatabaseConfig `json:"database"`
	Cache    CacheConfig    `json:"cache"`
	AI       AIConfig       `json:"ai"`
	YouTube  YouTubeConfig  `json:"youtube"`
	Sources  SourcesConfig  `json:"sources"`
	Logger   LoggerConfig   `json:"logger"`
}

type ServerConfig struct {
	Host           string   `json:"host"`
	Port           int      `json:"port"`
	ReadTimeout    Duration `json:"readTimeout"`
	WriteTimeout   Duration `json:"writeTimeout"`
	MaxHeaderBytes int      `json:"maxHeaderBytes"`
	AllowedOrigins []string `json:"allowedOrigins"`
	TrustedProxies []string `json:"trustedProxies"`
	RateLimit      int      `json:"rateLimit"`
	RequestTimeout Duration `json:"requestTimeout"`
	MaxRequestSize int64    `json:"maxRequestSize"`
	EnableHTTPS    bool     `json:"enableHTTPS"`
	CertFile       string   `json:"certFile"`
	KeyFile        string   `json:"keyFile"`
}

type DatabaseConfig struct {
	Host         string   `json:"host"`
	Port         int      `json:"port"`
	User         string   `json:"user"`
	Password     string   `json:"password"`
	Database     string   `json:"database"`
	MaxOpenConns int      `json:"maxOpenConns"`
	MaxIdleConns int      `json:"maxIdleConns"`
	SSLMode      string   `json:"sslMode"`
	Schema       string   `json:"schema"`
	Timeout      Duration `json:"timeout"`
}

type CacheConfig struct {
	Type        string   `json:"type"`
	Host        string   `json:"host"`
	Port        int      `json:"port"`
	Password    string   `json:"password"`
	DB          int      `json:"db"`
	TTL         Duration `json:"ttl"`
	MaxSize     int      `json:"maxSize"`
	EnableCache bool     `json:"enableCache"`
}

type AIConfig struct {
	Model          string   `json:"model"`
	APIKey         string   `json:"apiKey"`
	MaxTokens      int      `json:"maxTokens"`
	Temperature    float64  `json:"temperature"`
	EmbeddingModel string   `json:"embeddingModel"`
	EmbeddingDim   int      `json:"embeddingDim"`
	BatchSize      int      `json:"batchSize"`
	RequestTimeout Duration `json:"requestTimeout"`
	EnableRetries  bool     `json:"enableRetries"`
	MaxRetries     int      `json:"maxRetries"`
	RetryDelay     Duration `json:"retryDelay"`
}

type YouTubeConfig struct {
	APIKey         string   `json:"apiKey"`
	MaxResults     int      `json:"maxResults"`
	QuotaPerDay    int      `json:"quotaPerDay"`
	EnableCache    bool     `json:"enableCache"`
	CacheDuration  Duration `json:"cacheDuration"`
	RequestTimeout Duration `json:"requestTimeout"`
}

type SourcesConfig struct {
	DefaultSchedule   string   `json:"defaultSchedule"`
	MaxSourcesPerUser int      `json:"maxSourcesPerUser"`
	UpdateInterval    Duration `json:"updateInterval"`
	MaxRetries        int      `json:"maxRetries"`
	RetryDelay        Duration `json:"retryDelay"`
	TimeoutDuration   Duration `json:"timeoutDuration"`
	MaxConcurrent     int      `json:"maxConcurrent"`
	CleanupInterval   Duration `json:"cleanupInterval"`
	RetentionPeriod   Duration `json:"retentionPeriod"`
}

type LoggerConfig struct {
	Level         string `json:"level"`
	File          string `json:"file"`
	MaxSize       int    `json:"maxSize"`
	MaxBackups    int    `json:"maxBackups"`
	MaxAge        int    `json:"maxAge"`
	Compress      bool   `json:"compress"`
	EnableJSON    bool   `json:"enableJSON"`
	EnableConsole bool   `json:"enableConsole"`
}

// Duration is a wrapper type for time.Duration for JSON marshaling
type Duration time.Duration

// MarshalJSON implements the json.Marshaler interface
func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

// UnmarshalJSON implements the json.Unmarshaler interface
func (d *Duration) UnmarshalJSON(b []byte) error {
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}

	switch value := v.(type) {
	case float64:
		*d = Duration(time.Duration(value))
		return nil
	case string:
		tmp, err := time.ParseDuration(value)
		if err != nil {
			return err
		}
		*d = Duration(tmp)
		return nil
	default:
		return fmt.Errorf("invalid duration type %T", v)
	}
}

// String returns the string representation of the duration
func (d Duration) String() string {
	return time.Duration(d).String()
}

// Default configurations
var defaults = Config{
	Server: ServerConfig{
		Host:           "0.0.0.0",
		Port:           8080,
		ReadTimeout:    Duration(15 * time.Second),
		WriteTimeout:   Duration(15 * time.Second),
		MaxHeaderBytes: 1 << 20, // 1MB
		RateLimit:      100,     // requests per minute
		RequestTimeout: Duration(30 * time.Second),
		MaxRequestSize: 10 << 20, // 10MB
	},
	Database: DatabaseConfig{
		Host:         "localhost",
		Port:         5432,
		MaxOpenConns: 25,
		MaxIdleConns: 25,
		SSLMode:      "disable",
		Schema:       "public",
		Timeout:      Duration(5 * time.Second),
	},
	Cache: CacheConfig{
		Type:        "redis",
		Host:        "localhost",
		Port:        6379,
		DB:          0,
		TTL:         Duration(24 * time.Hour),
		MaxSize:     1000,
		EnableCache: true,
	},
	AI: AIConfig{
		Model:          "gpt-3.5-turbo",
		MaxTokens:      2000,
		Temperature:    0.7,
		EmbeddingModel: "text-embedding-ada-002",
		EmbeddingDim:   1536,
		BatchSize:      32,
		RequestTimeout: Duration(30 * time.Second),
		MaxRetries:     3,
		RetryDelay:     Duration(1 * time.Second),
	},
	YouTube: YouTubeConfig{
		MaxResults:     50,
		QuotaPerDay:    10000,
		EnableCache:    true,
		CacheDuration:  Duration(24 * time.Hour),
		RequestTimeout: Duration(10 * time.Second),
	},
	Sources: SourcesConfig{
		DefaultSchedule:   "0 */6 * * *", // Every 6 hours
		MaxSourcesPerUser: 100,
		UpdateInterval:    Duration(15 * time.Minute),
		MaxRetries:        3,
		RetryDelay:        Duration(5 * time.Second),
		TimeoutDuration:   Duration(1 * time.Minute),
		MaxConcurrent:     5,
		CleanupInterval:   Duration(24 * time.Hour),
		RetentionPeriod:   Duration(30 * 24 * time.Hour), // 30 days
	},
	Logger: LoggerConfig{
		Level:         "info",
		MaxSize:       100, // megabytes
		MaxBackups:    3,
		MaxAge:        28, // days
		Compress:      true,
		EnableJSON:    true,
		EnableConsole: true,
	},
}

// LoadConfig loads the configuration from a JSON file
func LoadConfig(configPath string) (*Config, error) {
	config := defaults

	// If config file exists, load it
	if configPath != "" {
		file, err := os.ReadFile(configPath)
		if err != nil {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}

		if err := json.Unmarshal(file, &config); err != nil {
			return nil, fmt.Errorf("error parsing config file: %w", err)
		}
	}

	// Override with environment variables if they exist
	config.loadFromEnv()

	// Validate configuration
	if err := config.validate(); err != nil {
		return nil, fmt.Errorf("config validation error: %w", err)
	}

	return &config, nil
}

// loadFromEnv loads configuration from environment variables
func (c *Config) loadFromEnv() {
	// Load environment variables if they exist
	if envPort := os.Getenv("SERVER_PORT"); envPort != "" {
		if port, err := parseInt(envPort); err == nil {
			c.Server.Port = port
		}
	}

	if dbHost := os.Getenv("DB_HOST"); dbHost != "" {
		c.Database.Host = dbHost
	}
	if dbUser := os.Getenv("DB_USER"); dbUser != "" {
		c.Database.User = dbUser
	}
	if dbPass := os.Getenv("DB_PASSWORD"); dbPass != "" {
		c.Database.Password = dbPass
	}
	if dbName := os.Getenv("DB_NAME"); dbName != "" {
		c.Database.Database = dbName
	}

	if aiKey := os.Getenv("AI_API_KEY"); aiKey != "" {
		c.AI.APIKey = aiKey
	}

	if ytKey := os.Getenv("YOUTUBE_API_KEY"); ytKey != "" {
		c.YouTube.APIKey = ytKey
	}
}

// validate checks if the configuration is valid
func (c *Config) validate() error {
	if c.Database.User == "" || c.Database.Password == "" {
		return fmt.Errorf("database credentials not provided")
	}
	if c.AI.APIKey == "" {
		return fmt.Errorf("AI API key not provided")
	}
	if c.YouTube.APIKey == "" {
		return fmt.Errorf("YouTube API key not provided")
	}
	return nil
}

// GetDatabaseURL returns the formatted database connection string
func (c *DatabaseConfig) GetDatabaseURL() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.User,
		c.Password,
		c.Host,
		c.Port,
		c.Database,
		c.SSLMode,
	)
}

// Helper function to parse integer from string
func parseInt(s string) (int, error) {
	var v int
	if _, err := fmt.Sscanf(s, "%d", &v); err != nil {
		return 0, err
	}
	return v, nil
}
