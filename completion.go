package smartcomplete

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// CompletionRequest contains all information needed for a completion
type CompletionRequest struct {
	ProjectID    string   `json:"projectId"`
	FilePath     string   `json:"filePath"`
	CursorLine   int      `json:"cursorLine"`
	CursorColumn int      `json:"cursorColumn"`
	LLM          string   `json:"llm,omitempty"`
	MaxTokens    int      `json:"maxTokens,omitempty"`
	ContextFiles []string `json:"contextFiles,omitempty"`
	Temperature  float64  `json:"temperature,omitempty"`
}

// CompletionResponse contains the generated completion
type CompletionResponse struct {
	Completion   string    `json:"completion"`
	LatencyMs    int64     `json:"latencyMs"`
	Model        string    `json:"model"`
	TokensUsed   int       `json:"tokensUsed"`
	CachedResult bool      `json:"cachedResult"`
	Timestamp    time.Time `json:"timestamp"`
}

// ProjectGetter provides access to project data
type ProjectGetter interface {
	GetProjectBaseDir(projectID string) (string, error)
	GetProjectAuthorizedFiles(projectID string) ([]string, error)
	GetProjectDiscussionFile(projectID string) (string, error)
	ReadFile(absolutePath string) ([]byte, error)
}

// GrokkerClient interface for LLM calls
type GrokkerClient interface {
	Query(ctx context.Context, llm string, systemMsg string, userMsg string, maxTokens int) (string, int, error)
}

// CompletionService is the main service
type CompletionService struct {
	config      *Config
	cache       *Cache
	rateLimiter *RateLimiter
	grokker     GrokkerClient
}

// NewCompletionService creates a new service
func NewCompletionService(config *Config) (*CompletionService, error) {
	if config == nil {
		config = DefaultConfig()
	}
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}
	return &CompletionService{
		config:      config,
		cache:       NewCache(config.CacheTTL, config.MaxCacheSize, config.EnableCache),
		rateLimiter: NewRateLimiter(),
	}, nil
}

// SetGrokkerClient sets the LLM client
func (s *CompletionService) SetGrokkerClient(client GrokkerClient) {
	s.grokker = client
}

// Complete generates a code completion
func (s *CompletionService) Complete(
	ctx context.Context,
	req CompletionRequest,
	projectGetter ProjectGetter,
) (*CompletionResponse, error) {
	startTime := time.Now()

	if err := s.validateRequest(req, projectGetter); err != nil {
		return nil, err
	}

	if err := s.rateLimiter.CheckLimit(req.ProjectID, s.config.MaxRequestsPerMinute, s.config.MaxRequestsPerHour); err != nil {
		return nil, err
	}

	baseDir, _ := projectGetter.GetProjectBaseDir(req.ProjectID)
	targetPath := resolveFilePath(baseDir, req.FilePath)
	fileContent, err := projectGetter.ReadFile(targetPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	if s.config.EnableCache {
		if cached, ok := s.cache.Get(req, string(fileContent)); ok {
			cached.CachedResult = true
			return cached, nil
		}
	}

	gatherer := &ContextGatherer{maxTokens: s.config.MaxContextTokens}
	completionCtx, err := gatherer.GatherContext(req, string(fileContent), projectGetter)
	if err != nil {
		return nil, fmt.Errorf("failed to gather context: %w", err)
	}

	formatter := &FIMFormatter{}
	prompt := formatter.FormatPrompt(completionCtx)

	llm := req.LLM
	if llm == "" {
		llm = s.config.DefaultLLM
	}
	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = s.config.MaxTokens
	}

	if s.grokker == nil {
		return nil, fmt.Errorf("grokker client not set")
	}

	systemMsg := "You are an expert code completion assistant. Complete the code at the cursor position. Output ONLY the completion text."
	completion, tokensUsed, err := s.grokker.Query(ctx, llm, systemMsg, prompt, maxTokens)
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	response := &CompletionResponse{
		Completion:   completion,
		LatencyMs:    time.Since(startTime).Milliseconds(),
		Model:        llm,
		TokensUsed:   tokensUsed,
		CachedResult: false,
		Timestamp:    time.Now(),
	}

	if s.config.EnableCache {
		s.cache.Put(req, string(fileContent), response)
	}

	return response, nil
}

func (s *CompletionService) validateRequest(req CompletionRequest, pg ProjectGetter) error {
	if req.ProjectID == "" || req.FilePath == "" {
		return ErrInvalidRequest
	}
	authorizedFiles, err := pg.GetProjectAuthorizedFiles(req.ProjectID)
	if err != nil {
		return err
	}
	baseDir, _ := pg.GetProjectBaseDir(req.ProjectID)
	targetPath := resolveFilePath(baseDir, req.FilePath)
	for _, authFile := range authorizedFiles {
		if filepath.Clean(resolveFilePath(baseDir, authFile)) == filepath.Clean(targetPath) {
			return nil
		}
	}
	return fmt.Errorf("%w: %s", ErrFileNotAuthorized, req.FilePath)
}

func resolveFilePath(baseDir, filePath string) string {
	if filepath.IsAbs(filePath) {
		return filePath
	}
	return filepath.Join(baseDir, filePath)
}

func cleanPath(p string) string {
	return filepath.Clean(strings.TrimSpace(p))
}
