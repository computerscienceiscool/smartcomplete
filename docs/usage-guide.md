# SmartComplete - Usage Guide

A simple Go library for AI-powered code completion in your application.

## Installation

```bash
go get github.com/computerscienceiscool/smartcomplete
```

## Quick Start (3 Steps)

### Step 1: Import the library

```go
import "github.com/computerscienceiscool/smartcomplete"
```

### Step 2: Create a completion service

```go
// Use default configuration
config := smartcomplete.DefaultConfig()

// Create the service
service, err := smartcomplete.NewCompletionService(config)
if err != nil {
    log.Fatal(err)
}
```

### Step 3: Set your LLM client

SmartComplete needs an LLM client to generate completions. Implement the `GrokkerClient` interface:

```go
type GrokkerClient interface {
    Query(ctx context.Context, llm string, systemMsg string, userMsg string, maxTokens int) (string, int, error)
}
```

Then set it:

```go
service.SetGrokkerClient(yourLLMClient)
```

## Complete Example

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/computerscienceiscool/smartcomplete"
)

func main() {
    // 1. Create configuration
    config := smartcomplete.DefaultConfig()
    
    // 2. Create service
    service, err := smartcomplete.NewCompletionService(config)
    if err != nil {
        log.Fatal(err)
    }
    
    // 3. Set your LLM client
    service.SetGrokkerClient(myLLMClient)
    
    // 4. Implement ProjectGetter (connects to your project system)
    projectGetter := &MyProjectGetter{
        // your implementation
    }
    
    // 5. Create a completion request
    req := smartcomplete.CompletionRequest{
        ProjectID:    "my-project",
        FilePath:     "src/main.go",
        CursorLine:   42,
        CursorColumn: 15,
    }
    
    // 6. Get completion
    ctx := context.Background()
    response, err := service.Complete(ctx, req, projectGetter)
    if err != nil {
        log.Fatal(err)
    }
    
    // 7. Use the completion
    fmt.Println("Completion:", response.Completion)
    fmt.Println("Tokens used:", response.TokensUsed)
    fmt.Println("Cached:", response.CachedResult)
}
```

## Implementing ProjectGetter

Your application needs to implement this interface:

```go
type ProjectGetter interface {
    GetProjectBaseDir(projectID string) (string, error)
    GetProjectAuthorizedFiles(projectID string) ([]string, error)
    GetProjectDiscussionFile(projectID string) (string, error)
    ReadFile(absolutePath string) ([]byte, error)
}
```

### Example Implementation

```go
type MyProjectGetter struct {
    projects map[string]*Project
}

func (g *MyProjectGetter) GetProjectBaseDir(projectID string) (string, error) {
    project, ok := g.projects[projectID]
    if !ok {
        return "", fmt.Errorf("project not found: %s", projectID)
    }
    return project.BaseDir, nil
}

func (g *MyProjectGetter) GetProjectAuthorizedFiles(projectID string) ([]string, error) {
    project, ok := g.projects[projectID]
    if !ok {
        return nil, fmt.Errorf("project not found: %s", projectID)
    }
    return project.AuthorizedFiles, nil
}

func (g *MyProjectGetter) GetProjectDiscussionFile(projectID string) (string, error) {
    project, ok := g.projects[projectID]
    if !ok {
        return "", fmt.Errorf("project not found: %s", projectID)
    }
    return project.DiscussionFile, nil
}

func (g *MyProjectGetter) ReadFile(absolutePath string) ([]byte, error) {
    return os.ReadFile(absolutePath)
}
```

## Implementing GrokkerClient

Connect your LLM (OpenAI, Anthropic, local model, etc.):

```go
type MyLLMClient struct {
    apiKey string
}

func (c *MyLLMClient) Query(
    ctx context.Context,
    llm string,
    systemMsg string,
    userMsg string,
    maxTokens int,
) (string, int, error) {
    // Call your LLM API here
    // Example with OpenAI:
    
    response, err := openai.CreateCompletion(ctx, openai.CompletionRequest{
        Model:       llm,
        Messages: []openai.Message{
            {Role: "system", Content: systemMsg},
            {Role: "user", Content: userMsg},
        },
        MaxTokens:   maxTokens,
    })
    
    if err != nil {
        return "", 0, err
    }
    
    completion := response.Choices[0].Message.Content
    tokensUsed := response.Usage.TotalTokens
    
    return completion, tokensUsed, nil
}
```

## Configuration

### Default Configuration

```go
config := smartcomplete.DefaultConfig()
```

Provides:
- LLM: `sonar-deep-research`
- Max Tokens: `500`
- Temperature: `0.2`
- Context Tokens: `10000`
- Cache: Enabled (5 min TTL)
- Rate Limits: 10/min, 50/hour

### Custom Configuration

```go
config := &smartcomplete.Config{
    DefaultLLM:           "gpt-4",
    MaxTokens:            1000,
    Temperature:          0.3,
    RequestTimeout:       60 * time.Second,
    MaxContextTokens:     15000,
    IncludeAgentsFile:    true,
    IncludeDiscussion:    true,
    MaxDiscussionRounds:  5,
    EnableCache:          true,
    CacheTTL:            10 * time.Minute,
    MaxCacheSize:        200 * 1024 * 1024, // 200MB
    MaxRequestsPerMinute: 20,
    MaxRequestsPerHour:   100,
}
```

### Load from YAML File

Create `completion.yaml`:

```yaml
default_llm: "gpt-4"
max_tokens: 1000
temperature: 0.3
request_timeout: 60s
max_context_tokens: 15000
include_agents_file: true
include_discussion: true
max_discussion_rounds: 5
enable_cache: true
cache_ttl: 10m
max_cache_size: 209715200
max_requests_per_minute: 20
max_requests_per_hour: 100
```

Load it:

```go
config, err := smartcomplete.LoadConfig("completion.yaml")
if err != nil {
    log.Fatal(err)
}
```

## Making Completion Requests

### Basic Request

```go
req := smartcomplete.CompletionRequest{
    ProjectID:    "my-project",
    FilePath:     "src/main.go",
    CursorLine:   10,    // 0-indexed
    CursorColumn: 5,     // 0-indexed
}

response, err := service.Complete(ctx, req, projectGetter)
```

### Request with Options

```go
req := smartcomplete.CompletionRequest{
    ProjectID:    "my-project",
    FilePath:     "src/main.go",
    CursorLine:   10,
    CursorColumn: 5,
    LLM:          "gpt-4",           // Override default LLM
    MaxTokens:    1000,               // Override default max tokens
    Temperature:  0.5,                // Override default temperature
    ContextFiles: []string{           // Include additional context
        "src/utils.go",
        "src/types.go",
    },
}

response, err := service.Complete(ctx, req, projectGetter)
```

## Understanding the Response

```go
type CompletionResponse struct {
    Completion   string    // The generated code
    LatencyMs    int64     // How long it took
    Model        string    // Which LLM was used
    TokensUsed   int       // Tokens consumed
    CachedResult bool      // Was this from cache?
    Timestamp    time.Time // When it was generated
}
```

### Example

```go
response, err := service.Complete(ctx, req, projectGetter)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Completion: %s\n", response.Completion)
fmt.Printf("Latency: %dms\n", response.LatencyMs)
fmt.Printf("Model: %s\n", response.Model)
fmt.Printf("Tokens: %d\n", response.TokensUsed)

if response.CachedResult {
    fmt.Println("This was served from cache!")
}
```

## How It Works

SmartComplete gathers context intelligently:

1. **File Content**: Extracts code before and after cursor
2. **AGENTS.md**: Finds project instructions by walking up directories
3. **Discussion**: Includes recent project discussion context
4. **Related Files**: Optionally includes imported or related files
5. **Token Budget**: Prioritizes context to fit within limits

### Context Priority

1. Target file (prefix/suffix) - highest priority
2. AGENTS.md instructions
3. Discussion context
4. Additional files

## Features

### ✅ Caching

Identical requests are cached for 5 minutes (configurable):

```go
// First request - calls LLM
resp1, _ := service.Complete(ctx, req, projectGetter)
fmt.Println(resp1.CachedResult) // false
fmt.Println(resp1.LatencyMs)    // 2000ms

// Identical request - from cache
resp2, _ := service.Complete(ctx, req, projectGetter)
fmt.Println(resp2.CachedResult) // true
fmt.Println(resp2.LatencyMs)    // 1ms
```

Cache is invalidated when:
- File content changes
- TTL expires (default 5 min)
- Cache is full (oldest entries removed)

### ✅ Rate Limiting

Per-project rate limits prevent abuse:

```go
// 10 requests per minute, 50 per hour (default)
for i := 0; i < 15; i++ {
    _, err := service.Complete(ctx, req, projectGetter)
    if err != nil {
        fmt.Println(err) // "rate limit exceeded" after 10 requests
        break
    }
}
```

### ✅ Error Handling

All errors include context:

```go
resp, err := service.Complete(ctx, req, projectGetter)
if err != nil {
    var completionErr *smartcomplete.CompletionError
    if errors.As(err, &completionErr) {
        fmt.Println("Code:", completionErr.Code)
        fmt.Println("Message:", completionErr.Message)
    }
}
```

Error codes:
- `VALIDATION_ERROR` - Invalid request
- `RATE_LIMIT` - Too many requests
- `FILE_ACCESS` - File read error
- `PROJECT_ACCESS` - Project not found
- `LLM_ERROR` - LLM call failed
- `TIMEOUT` - Request timeout

## Best Practices

### 1. Set Appropriate Timeouts

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

response, err := service.Complete(ctx, req, projectGetter)
```

### 2. Handle Errors Gracefully

```go
response, err := service.Complete(ctx, req, projectGetter)
if err != nil {
    log.Printf("Completion failed: %v", err)
    // Fall back gracefully - don't crash
    return
}
```

### 3. Monitor Cache Hit Rate

```go
if response.CachedResult {
    metrics.IncrementCacheHits()
} else {
    metrics.IncrementCacheMisses()
}
```

### 4. Tune Configuration for Your Use Case

```go
// For faster responses, reduce context:
config.MaxContextTokens = 5000

// For better completions, increase context:
config.MaxContextTokens = 20000

// For high-traffic apps, increase rate limits:
config.MaxRequestsPerMinute = 50
config.MaxRequestsPerHour = 500
```

## Examples

### Web Application Integration

```go
func handleCompletionRequest(w http.ResponseWriter, r *http.Request) {
    var req smartcomplete.CompletionRequest
    json.NewDecoder(r.Body).Decode(&req)
    
    ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
    defer cancel()
    
    response, err := completionService.Complete(ctx, req, projectGetter)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    json.NewEncoder(w).Encode(response)
}
```

### CLI Tool Integration

```go
func main() {
    service, _ := smartcomplete.NewCompletionService(smartcomplete.DefaultConfig())
    service.SetGrokkerClient(llmClient)
    
    req := smartcomplete.CompletionRequest{
        ProjectID:    "my-project",
        FilePath:     "main.go",
        CursorLine:   10,
        CursorColumn: 0,
    }
    
    ctx := context.Background()
    response, err := service.Complete(ctx, req, projectGetter)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }
    
    fmt.Println(response.Completion)
}
```

## Troubleshooting

### "grokker client not set"

You forgot to set the LLM client:

```go
service.SetGrokkerClient(yourLLMClient)
```

### "file not authorized"

The file must be in the project's authorized files list:

```go
func (g *MyProjectGetter) GetProjectAuthorizedFiles(projectID string) ([]string, error) {
    return []string{
        "src/main.go",      // Include files here
        "src/utils.go",
        "tests/test.go",
    }, nil
}
```

### "rate limit exceeded"

Too many requests. Either:
- Wait a minute
- Increase rate limits in config
- Reset limits: Contact library maintainer for API

## Support

- **GitHub**: https://github.com/computerscienceiscool/smartcomplete
- **Issues**: https://github.com/computerscienceiscool/smartcomplete/issues
- **Documentation**: See README.md and INTEGRATION.md

