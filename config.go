package smartcomplete

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds library configuration
type Config struct {
	DefaultLLM           string        `yaml:"default_llm"`
	MaxTokens            int           `yaml:"max_tokens"`
	Temperature          float64       `yaml:"temperature"`
	RequestTimeout       time.Duration `yaml:"request_timeout"`
	MaxContextTokens     int           `yaml:"max_context_tokens"`
	IncludeAgentsFile    bool          `yaml:"include_agents_file"`
	IncludeDiscussion    bool          `yaml:"include_discussion"`
	MaxDiscussionRounds  int           `yaml:"max_discussion_rounds"`
	EnableCache          bool          `yaml:"enable_cache"`
	CacheTTL             time.Duration `yaml:"cache_ttl"`
	MaxCacheSize         int           `yaml:"max_cache_size"`
	MaxRequestsPerMinute int           `yaml:"max_requests_per_minute"`
	MaxRequestsPerHour   int           `yaml:"max_requests_per_hour"`
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		DefaultLLM:           "sonar-deep-research",
		MaxTokens:            500,
		Temperature:          0.2,
		RequestTimeout:       30 * time.Second,
		MaxContextTokens:     10000,
		IncludeAgentsFile:    true,
		IncludeDiscussion:    true,
		MaxDiscussionRounds:  3,
		EnableCache:          true,
		CacheTTL:             5 * time.Minute,
		MaxCacheSize:         100 * 1024 * 1024, // 100MB
		MaxRequestsPerMinute: 10,
		MaxRequestsPerHour:   50,
	}
}

// LoadConfig loads configuration from file or uses defaults
func LoadConfig(path string) (*Config, error) {
	if path == "" {
		return DefaultConfig(), nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	config := DefaultConfig()
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return config, nil
}

// Validate checks if configuration is valid
func (c *Config) Validate() error {
	if c.DefaultLLM == "" {
		return fmt.Errorf("default_llm cannot be empty")
	}
	if c.MaxTokens <= 0 {
		return fmt.Errorf("max_tokens must be positive")
	}
	if c.Temperature < 0 || c.Temperature > 2 {
		return fmt.Errorf("temperature must be between 0 and 2")
	}
	if c.MaxContextTokens <= 0 {
		return fmt.Errorf("max_context_tokens must be positive")
	}
	if c.MaxRequestsPerMinute <= 0 {
		return fmt.Errorf("max_requests_per_minute must be positive")
	}
	if c.MaxRequestsPerHour <= 0 {
		return fmt.Errorf("max_requests_per_hour must be positive")
	}
	return nil
}
