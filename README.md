# SmartComplete - AI Code Completion Library for Storm

SmartComplete is a standalone Go library that provides AI-powered code completion functionality for the Storm multi-project LLM chat application. It integrates seamlessly with Storm's existing architecture with minimal code changes.

## Features

- **Fill-in-Middle (FIM) Completions**: Context-aware code generation using prefix/suffix
- **Project-Aware**: Leverages Storm's project structure (authorized files, AGENTS.md, discussion context)
- **Manual Trigger**: Button/hotkey-based completion (not real-time)
- **Caching**: Intelligent caching with TTL and file hash validation
- **Rate Limiting**: Per-project request throttling (10/min, 50/hour defaults)
- **Multi-LLM Support**: Works with any grokker-compatible LLM
- **Minimal Integration**: Storm only needs to implement one interface

## Installation

```bash
go get github.com/yourusername/smartcomplete
```

## Quick Start

### 1. Initialize the Service

```go
import "github.com/yourusername/smartcomplete"

// Load or create config
config := smartcomplete.DefaultConfig()
// Or load from file:
// config, err := smartcomplete.LoadConfig("completion.yaml")

// Create completion service
completionSvc, err := smartcomplete.NewCompletionService(config)
if err != nil {
    log.Fatalf("Failed to initialize completion service: %v", err)
}
```

### 2. Implement ProjectGetter Interface

Storm needs to implement this simple interface:

```go
type StormProjectGetter struct {
    projects *Projects  // Storm's existing projects registry
}

func (g *StormProjectGetter) GetProjectBaseDir(projectID string) (string, error) {
    project, err := g.projects.Get(projectID)
    if err != nil {
        return "", err
    }
    return project.BaseDir, nil
}

func (g *StormProjectGetter) GetProjectAuthorizedFiles(projectID string) ([]string, error) {
    project, err := g.projects.Get(projectID)
    if err != nil {
        return nil, err
    }
    return project.AuthorizedFiles, nil
}

func (g *StormProjectGetter) GetProjectDiscussionFile(projectID string) (string, error) {
    project, err := g.projects.Get(projectID)
    if err != nil {
        return "", err
    }
    return project.MarkdownFile, nil
}

func (g *StormProjectGetter) ReadFile(absolutePath string) ([]byte, error) {
    return ioutil.ReadFile(absolutePath)
}
```

### 3. Add WebSocket Handler

```go
// Handle new "codeCompletion" message type
case "codeCompletion":
    var req smartcomplete.CompletionRequest
    if err := mapstructure.Decode(msg, &req); err != nil {
        sendError(conn, "Invalid completion request", err)
        return
    }
    
    // Create project getter
    getter := &StormProjectGetter{projects: projects}
    
    // Call completion service
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    resp, err := completionSvc.Complete(ctx, req, getter)
    if err != nil {
        sendError(conn, "Completion failed", err)
        return
    }
    
    // Send response back
    response := map[string]interface{}{
        "type": "completionResponse",
        "queryID": msg["queryID"],
        "completion": resp.Completion,
        "latencyMs": resp.LatencyMs,
        "model": resp.Model,
        "tokensUsed": resp.TokensUsed,
        "cachedResult": resp.CachedResult,
    }
    conn.WriteJSON(response)
```

### 4. Add Frontend UI

```javascript
// In project.html - add completion button to file editor
function requestCompletion() {
    const fileContent = editor.getValue();
    const cursor = editor.getCursor();
    
    const message = {
        type: 'codeCompletion',
        queryID: generateQueryID(),
        projectId: currentProjectID,
        filePath: currentFile,
        cursorLine: cursor.line,
        cursorColumn: cursor.ch,
    };
    
    ws.send(JSON.stringify(message));
    showLoadingIndicator('Generating code...');
}

// Handle completion response
socket.onmessage = (event) => {
    const msg = JSON.parse(event.data);
    
    if (msg.type === 'completionResponse') {
        hideLoadingIndicator();
        showCompletionPreview(msg.completion, msg.queryID);
    }
};

// Show completion with accept/reject buttons
function showCompletionPreview(completion, queryID) {
    const modal = document.getElementById('completion-modal');
    document.getElementById('completion-text').textContent = completion;
    modal.style.display = 'block';
    
    document.getElementById('accept-btn').onclick = () => {
        editor.replaceSelection(completion);
        modal.style.display = 'none';
    };
    
    document.getElementById('reject-btn').onclick = () => {
        modal.style.display = 'none';
    };
}
```

## Configuration

### Default Configuration

```go
config := smartcomplete.DefaultConfig()
```

Provides:
- **LLM**: sonar-deep-research
- **Max Tokens**: 500
- **Temperature**: 0.2
- **Context Tokens**: 10,000
- **Cache TTL**: 5 minutes
- **Rate Limits**: 10/min, 50/hour

### Custom Configuration

```go
config := &smartcomplete.Config{
    DefaultLLM:           "sonar-deep-research",
    MaxTokens:            500,
    Temperature:          0.2,
    RequestTimeout:       30 * time.Second,
    MaxContextTokens:     10000,
    IncludeAgentsFile:    true,
    IncludeDiscussion:    true,
    MaxDiscussionRounds:  3,
    EnableCache:          true,
    CacheTTL:            5 * time.Minute,
    MaxCacheSize:        100 * 1024 * 1024, // 100MB
    MaxRequestsPerMinute: 10,
    MaxRequestsPerHour:   50,
}
```

### YAML Configuration

Create `completion.yaml`:

```yaml
default_llm: "sonar-deep-research"
max_tokens: 500
temperature: 0.2
request_timeout: 30s

# Context gathering
max_context_tokens: 10000
include_agents_file: true
include_discussion: true
max_discussion_rounds: 3

# Caching
enable_cache: true
cache_ttl: 5m
max_cache_size: 104857600  # 100MB

# Rate limiting
max_requests_per_minute: 10
max_requests_per_hour: 50
```

Load it:

```go
config, err := smartcomplete.LoadConfig("completion.yaml")
```

## API Reference

### CompletionRequest

```go
type CompletionRequest struct {
    ProjectID    string   `json:"projectId"`      // Required
    FilePath     string   `json:"filePath"`       // Required, relative to BaseDir
    CursorLine   int      `json:"cursorLine"`     // Required, 0-indexed
    CursorColumn int      `json:"cursorColumn"`   // Required, 0-indexed
    LLM          string   `json:"llm,omitempty"`  // Optional, uses default if empty
    MaxTokens    int      `json:"maxTokens,omitempty"`
    ContextFiles []string `json:"contextFiles,omitempty"`
    Temperature  float64  `json:"temperature,omitempty"`
}
```

### CompletionResponse

```go
type CompletionResponse struct {
    Completion   string    `json:"completion"`   // Generated code
    LatencyMs    int64     `json:"latencyMs"`    // Request duration
    Model        string    `json:"model"`        // LLM used
    TokensUsed   int       `json:"tokensUsed"`   // Tokens consumed
    CachedResult bool      `json:"cachedResult"` // Was cached?
    Timestamp    time.Time `json:"timestamp"`    // When generated
}
```

### ProjectGetter Interface

```go
type ProjectGetter interface {
    GetProjectBaseDir(projectID string) (string, error)
    GetProjectAuthorizedFiles(projectID string) ([]string, error)
    GetProjectDiscussionFile(projectID string) (string, error)
    ReadFile(absolutePath string) ([]byte, error)
}
```

## WebSocket Messages

### Request (codeCompletion)

```json
{
  "type": "codeCompletion",
  "queryID": "uuid-1234",
  "projectId": "my-project",
  "filePath": "src/main.go",
  "cursorLine": 42,
  "cursorColumn": 8,
  "llm": "sonar-deep-research",
  "maxTokens": 500
}
```

### Response (completionResponse)

```json
{
  "type": "completionResponse",
  "queryID": "uuid-1234",
  "completion": "result := a + b\n    fmt.Println(result)",
  "latencyMs": 1847,
  "model": "sonar-deep-research",
  "tokensUsed": 234,
  "cachedResult": false
}
```

### Error (completionError)

```json
{
  "type": "error",
  "queryID": "uuid-1234",
  "error": "File not authorized",
  "message": "File src/secret.go is not in project authorized files list"
}
```

## How It Works

### Context Gathering

SmartComplete intelligently gathers context for completions:

1. **Target File**: Extracts prefix (before cursor) and suffix (after cursor)
2. **AGENTS.md**: Walks up directory tree to find project instructions
3. **Discussion**: Extracts last N rounds from project's markdown file
4. **Additional Files**: Can include imports or related files
5. **Token Budget**: Prioritizes target file → AGENTS → discussion → related files

### FIM (Fill-in-Middle) Prompt

The library constructs a structured prompt:

```
You are an expert {language} programmer. Complete the code below.

PROJECT INSTRUCTIONS:
{AGENTS.md content}

RECENT PROJECT DISCUSSION:
{Last 3 discussion rounds}

RELATED FILES:
{Additional context files}

CODE BEFORE CURSOR:
{Prefix}

CODE AFTER CURSOR:
{Suffix}

INSTRUCTIONS:
Complete only the middle section between the cursor position.
Provide syntactically correct, idiomatic {language} code.
Do not repeat the prefix or suffix.
Output only the completion, nothing else.
```

### Caching

Completions are cached with:
- **Key**: `{projectID}:{filePath}:{cursorLine}:{cursorColumn}:{llm}`
- **Validation**: File content hash must match
- **TTL**: 5 minutes (configurable)
- **Eviction**: Simple oldest-first when cache exceeds max size

Cache hits avoid LLM calls entirely, reducing latency to <1ms.

### Rate Limiting

Per-project rate limiting prevents abuse:
- **Minute Window**: 10 requests/minute (default)
- **Hour Window**: 50 requests/hour (default)
- **Reset**: Windows reset on a rolling basis

## Error Handling

All errors are wrapped with context:

```go
resp, err := completionSvc.Complete(ctx, req, getter)
if err != nil {
    var completionErr *smartcomplete.CompletionError
    if errors.As(err, &completionErr) {
        log.Printf("Code: %s, Message: %s", completionErr.Code, completionErr.Message)
        // Handle specific error codes
        switch completionErr.Code {
        case smartcomplete.CodeRateLimit:
            // Handle rate limit
        case smartcomplete.CodeFileAccess:
            // Handle file access error
        default:
            // Handle generic error
        }
    }
}
```

### Error Codes

- `VALIDATION_ERROR`: Invalid request parameters
- `RATE_LIMIT`: Rate limit exceeded
- `FILE_ACCESS`: File read/authorization error
- `PROJECT_ACCESS`: Project not found or inaccessible
- `CONTEXT_ERROR`: Context gathering failed
- `LLM_ERROR`: LLM request failed
- `CACHE_ERROR`: Cache operation failed
- `TIMEOUT`: Request timeout
- `INTERNAL_ERROR`: Internal service error

## Performance

### Expected Latencies

- **Cache Hit**: <1ms
- **Cache Miss + LLM**: 1-15 seconds (depends on LLM)
- **Context Gathering**: ~50ms (for typical project)
- **Validation**: <10ms

### Resource Usage

- **Memory**: ~5-10MB per active project
- **Cache**: Up to 100MB (configurable)
- **Network**: Only during LLM calls

## Testing

Run tests:

```bash
go test ./...
```

Run with coverage:

```bash
go test -cover ./...
```

## Architecture

```
Storm (imports smartcomplete)
  ├─ Implements ProjectGetter interface
  ├─ Calls completionSvc.Complete()
  └─ Handles WebSocket integration

SmartComplete Library
  ├─ completion.go     # Main service
  ├─ config.go         # Configuration
  ├─ context.go        # Context gathering
  ├─ fim.go            # FIM prompt formatting
  ├─ cache.go          # Response caching
  ├─ errors.go         # Error types
  └─ ratelimit.go      # Rate limiting
```

## Roadmap

### Current (v1.0)
- ✅ Basic FIM completions
- ✅ Project-aware context
- ✅ Caching
- ✅ Rate limiting

### Future (v1.1+)
- [ ] Streaming responses
- [ ] Multiple completion options
- [ ] Edit suggestions (not just FIM)
- [ ] Smart import handling
- [ ] Test generation
- [ ] Documentation generation

## Contributing

Contributions welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass
5. Submit a pull request

## License

MIT License - see LICENSE file for details

## Support

- **Issues**: [GitHub Issues](https://github.com/yourusername/smartcomplete/issues)
- **Discussions**: [GitHub Discussions](https://github.com/yourusername/smartcomplete/discussions)
- **Documentation**: [Full docs](https://docs.smartcomplete.dev)

## Changelog

### v1.0.0 (Initial Release)
- Basic FIM completions
- Project-aware context gathering
- AGENTS.md integration
- Discussion context extraction
- Response caching with TTL
- Per-project rate limiting
- Comprehensive error handling
