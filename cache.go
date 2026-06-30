package smartcomplete

import (
	"crypto/sha256"
	"fmt"
	"sync"
	"time"
)

// Cache stores recent completions to reduce latency and cost
type Cache struct {
	entries  map[string]*CacheEntry
	mu       sync.RWMutex
	ttl      time.Duration
	maxSize  int
	enabled  bool
}

// CacheEntry represents a cached completion
type CacheEntry struct {
	Response  *CompletionResponse
	CreatedAt time.Time
	FileHash  string
}

// NewCache creates a new cache
func NewCache(ttl time.Duration, maxSize int, enabled bool) *Cache {
	return &Cache{
		entries: make(map[string]*CacheEntry),
		ttl:     ttl,
		maxSize: maxSize,
		enabled: enabled,
	}
}

// Get retrieves a cached completion if valid
func (c *Cache) Get(req CompletionRequest, fileContent string) (*CompletionResponse, bool) {
	if !c.enabled {
		return nil, false
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	key := c.cacheKey(req)
	entry, exists := c.entries[key]

	if !exists {
		return nil, false
	}

	// Check if expired
	if time.Since(entry.CreatedAt) > c.ttl {
		return nil, false
	}

	// Check if file changed (invalidate cache)
	currentHash := hashContent(fileContent)
	if entry.FileHash != currentHash {
		return nil, false
	}

	return entry.Response, true
}

// Put stores a completion in cache
func (c *Cache) Put(req CompletionRequest, fileContent string, resp *CompletionResponse) {
	if !c.enabled {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Simple eviction: if too many entries, remove oldest
	if len(c.entries) > 1000 {
		var oldestKey string
		var oldestTime time.Time
		for key, entry := range c.entries {
			if oldestKey == "" || entry.CreatedAt.Before(oldestTime) {
				oldestKey = key
				oldestTime = entry.CreatedAt
			}
		}
		if oldestKey != "" {
			delete(c.entries, oldestKey)
		}
	}

	key := c.cacheKey(req)
	c.entries[key] = &CacheEntry{
		Response:  resp,
		CreatedAt: time.Now(),
		FileHash:  hashContent(fileContent),
	}
}

func (c *Cache) cacheKey(req CompletionRequest) string {
	return fmt.Sprintf("%s:%s:%d:%d:%s",
		req.ProjectID,
		req.FilePath,
		req.CursorLine,
		req.CursorColumn,
		req.LLM,
	)
}

func hashContent(content string) string {
	hash := sha256.Sum256([]byte(content))
	return fmt.Sprintf("%x", hash)
}
