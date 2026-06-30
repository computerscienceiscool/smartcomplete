# SmartComplete Integration Guide for Storm

This guide walks you through integrating SmartComplete into Storm with minimal changes to the existing codebase.

## Prerequisites

- Go 1.21 or later
- Storm project with existing WebSocket infrastructure
- Access to grokker for LLM calls

## Step-by-Step Integration

### 1. Add SmartComplete Dependency

Add to Storm's `go.mod`:

```bash
go get github.com/yourusername/smartcomplete
```

### 2. Create ProjectGetter Adapter

In `storm/completion_adapter.go`:

```go
package main

import (
    "io/ioutil"
    "github.com/yourusername/smartcomplete"
)

type StormProjectGetter struct {
    projects *Projects
}

func NewStormProjectGetter(projects *Projects) *StormProjectGetter {
    return &StormProjectGetter{projects: projects}
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

### 3. Initialize Completion Service

In `storm/main.go` or where you initialize services:

```go
import (
    "github.com/yourusername/smartcomplete"
)

var completionService *smartcomplete.CompletionService

func initializeServices() error {
    // Load configuration
    config, err := smartcomplete.LoadConfig("completion.yaml")
    if err != nil {
        log.Printf("Using default completion config: %v", err)
        config = smartcomplete.DefaultConfig()
    }
    
    // Create completion service
    completionService, err = smartcomplete.NewCompletionService(config)
    if err != nil {
        return fmt.Errorf("failed to initialize completion service: %w", err)
    }
    
    log.Println("SmartComplete service initialized")
    return nil
}
```

### 4. Add WebSocket Handler

In `storm/websocket.go` where you handle messages:

```go
func handleWebSocketMessage(conn *websocket.Conn, msg map[string]interface{}, projects *Projects) {
    msgType, ok := msg["type"].(string)
    if !ok {
        sendError(conn, "Missing message type", nil)
        return
    }
    
    switch msgType {
    case "codeCompletion":
        handleCodeCompletion(conn, msg, projects)
    // ... existing cases
    }
}

func handleCodeCompletion(conn *websocket.Conn, msg map[string]interface{}, projects *Projects) {
    // Extract request parameters
    projectID, _ := msg["projectId"].(string)
    filePath, _ := msg["filePath"].(string)
    cursorLine, _ := msg["cursorLine"].(float64)
    cursorColumn, _ := msg["cursorColumn"].(float64)
    queryID, _ := msg["queryID"].(string)
    
    // Optional parameters
    llm, _ := msg["llm"].(string)
    maxTokens := 500
    if mt, ok := msg["maxTokens"].(float64); ok {
        maxTokens = int(mt)
    }
    
    // Build request
    req := smartcomplete.CompletionRequest{
        ProjectID:    projectID,
        FilePath:     filePath,
        CursorLine:   int(cursorLine),
        CursorColumn: int(cursorColumn),
        LLM:          llm,
        MaxTokens:    maxTokens,
    }
    
    // Create project getter
    getter := NewStormProjectGetter(projects)
    
    // Call completion service with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    resp, err := completionService.Complete(ctx, req, getter)
    if err != nil {
        sendCompletionError(conn, queryID, err)
        return
    }
    
    // Send response
    response := map[string]interface{}{
        "type":         "completionResponse",
        "queryID":      queryID,
        "completion":   resp.Completion,
        "latencyMs":    resp.LatencyMs,
        "model":        resp.Model,
        "tokensUsed":   resp.TokensUsed,
        "cachedResult": resp.CachedResult,
    }
    
    if err := conn.WriteJSON(response); err != nil {
        log.Printf("Failed to send completion response: %v", err)
    }
}

func sendCompletionError(conn *websocket.Conn, queryID string, err error) {
    errResponse := map[string]interface{}{
        "type":    "completionError",
        "queryID": queryID,
        "error":   err.Error(),
    }
    
    var completionErr *smartcomplete.CompletionError
    if errors.As(err, &completionErr) {
        errResponse["code"] = completionErr.Code
        errResponse["message"] = completionErr.Message
    }
    
    conn.WriteJSON(errResponse)
}
```

### 5. Add Frontend UI

In `storm/web/project.html`:

```html
<!-- Add completion button to file editor toolbar -->
<div class="editor-toolbar">
    <button id="completion-btn" onclick="requestCompletion()">
        <i class="fas fa-magic"></i> Complete Code
    </button>
    <!-- ... existing toolbar buttons -->
</div>

<!-- Add completion modal -->
<div id="completion-modal" class="modal" style="display: none;">
    <div class="modal-content">
        <h3>Code Completion</h3>
        <pre id="completion-text"></pre>
        <div class="modal-actions">
            <button id="accept-completion" class="btn-primary">Accept</button>
            <button id="reject-completion" class="btn-secondary">Reject</button>
        </div>
    </div>
</div>
```

In `storm/web/js/project.js`:

```javascript
let pendingCompletion = null;

function requestCompletion() {
    if (!currentFile || !currentProjectID) {
        showNotification('No file open', 'error');
        return;
    }
    
    const cursor = editor.getCursor();
    const queryID = generateQueryID();
    
    const message = {
        type: 'codeCompletion',
        queryID: queryID,
        projectId: currentProjectID,
        filePath: currentFile,
        cursorLine: cursor.line,
        cursorColumn: cursor.ch,
    };
    
    ws.send(JSON.stringify(message));
    showNotification('Generating code completion...', 'info');
    
    // Show loading state
    document.getElementById('completion-btn').disabled = true;
}

// Add to existing WebSocket message handler
function handleWebSocketMessage(msg) {
    switch (msg.type) {
        case 'completionResponse':
            handleCompletionResponse(msg);
            break;
        case 'completionError':
            handleCompletionError(msg);
            break;
        // ... existing cases
    }
}

function handleCompletionResponse(msg) {
    document.getElementById('completion-btn').disabled = false;
    
    pendingCompletion = {
        text: msg.completion,
        queryID: msg.queryID,
        cursor: editor.getCursor(),
    };
    
    // Show completion in modal
    document.getElementById('completion-text').textContent = msg.completion;
    document.getElementById('completion-modal').style.display = 'block';
    
    // Add metadata
    const meta = msg.cachedResult ? '(cached)' : `(${msg.latencyMs}ms, ${msg.tokensUsed} tokens)`;
    showNotification(`Completion ready ${meta}`, 'success');
}

function handleCompletionError(msg) {
    document.getElementById('completion-btn').disabled = false;
    showNotification(`Completion failed: ${msg.error}`, 'error');
}

// Accept completion
document.getElementById('accept-completion').onclick = function() {
    if (pendingCompletion) {
        editor.setCursor(pendingCompletion.cursor);
        editor.replaceSelection(pendingCompletion.text);
        document.getElementById('completion-modal').style.display = 'none';
        pendingCompletion = null;
    }
};

// Reject completion
document.getElementById('reject-completion').onclick = function() {
    document.getElementById('completion-modal').style.display = 'none';
    pendingCompletion = null;
};

// Keyboard shortcut: Ctrl+Space for completion
editor.addKeyMap({
    'Ctrl-Space': function(cm) {
        requestCompletion();
    }
});
```

### 6. Add CSS Styling

In `storm/web/css/project.css`:

```css
#completion-btn {
    background: #667eea;
    color: white;
    border: none;
    padding: 8px 16px;
    border-radius: 4px;
    cursor: pointer;
    margin-right: 8px;
}

#completion-btn:hover {
    background: #5568d3;
}

#completion-btn:disabled {
    background: #ccc;
    cursor: not-allowed;
}

#completion-modal {
    position: fixed;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    background: rgba(0, 0, 0, 0.5);
    z-index: 1000;
    display: flex;
    align-items: center;
    justify-content: center;
}

#completion-modal .modal-content {
    background: white;
    padding: 24px;
    border-radius: 8px;
    max-width: 800px;
    max-height: 80vh;
    overflow: auto;
}

#completion-modal pre {
    background: #f5f5f5;
    padding: 16px;
    border-radius: 4px;
    overflow-x: auto;
    font-family: 'Monaco', 'Courier New', monospace;
    font-size: 14px;
}

#completion-modal .modal-actions {
    margin-top: 16px;
    display: flex;
    gap: 8px;
    justify-content: flex-end;
}
```

## Configuration

Create `completion.yaml` in Storm's root directory:

```yaml
default_llm: "sonar-deep-research"
max_tokens: 500
temperature: 0.2
request_timeout: 30s
max_context_tokens: 10000
include_agents_file: true
include_discussion: true
max_discussion_rounds: 3
enable_cache: true
cache_ttl: 5m
max_cache_size: 104857600
max_requests_per_minute: 10
max_requests_per_hour: 50
```

## Testing

1. **Start Storm**: `go run .`
2. **Open a project** with authorized files
3. **Open a file** in the editor
4. **Click "Complete Code"** or press `Ctrl+Space`
5. **Review completion** in modal
6. **Accept or reject** the suggestion

## Troubleshooting

### Completion fails with "file not authorized"
- Ensure the file is in the project's authorized files list
- Check that file paths are relative to project BaseDir

### Rate limit errors
- Adjust `max_requests_per_minute` and `max_requests_per_hour` in config
- Use `completionService.RateLimiter.Reset(projectID)` to reset limits

### Cache not working
- Ensure `enable_cache: true` in config
- Check that file content hasn't changed
- Verify cursor position is exactly the same

### Timeout errors
- Increase `request_timeout` in config
- Check grokker LLM connection

## Performance Tips

1. **Use caching**: Keep `enable_cache: true`
2. **Tune context size**: Reduce `max_context_tokens` if completions are slow
3. **Limit discussion rounds**: Set `max_discussion_rounds: 2` for faster context gathering
4. **Choose faster LLMs**: Use smaller models for faster responses

## Next Steps

- Add telemetry/metrics for completion usage
- Implement streaming responses for real-time completion
- Add completion history/suggestions panel
- Create admin dashboard for monitoring rate limits
- Add A/B testing for different LLM models

## Support

If you encounter issues:
1. Check logs for detailed error messages
2. Verify all interface methods are implemented correctly
3. Test with minimal config first
4. Open an issue on GitHub
