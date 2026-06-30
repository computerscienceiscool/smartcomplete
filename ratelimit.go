package smartcomplete

import (
	"sync"
	"time"
)

// RateLimiter tracks request counts per project
type RateLimiter struct {
	requestCounts map[string]*RequestCount
	mu            sync.RWMutex
}

// RequestCount tracks requests within time windows
type RequestCount struct {
	Minute          int
	Hour            int
	LastMinuteReset time.Time
	LastHourReset   time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		requestCounts: make(map[string]*RequestCount),
	}
}

// CheckLimit checks if a request is within rate limits
func (r *RateLimiter) CheckLimit(projectID string, maxPerMin, maxPerHour int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	count, exists := r.requestCounts[projectID]
	if !exists {
		count = &RequestCount{
			LastMinuteReset: now,
			LastHourReset:   now,
		}
		r.requestCounts[projectID] = count
	}

	// Reset counters if time windows passed
	if now.Sub(count.LastMinuteReset) >= time.Minute {
		count.Minute = 0
		count.LastMinuteReset = now
	}
	if now.Sub(count.LastHourReset) >= time.Hour {
		count.Hour = 0
		count.LastHourReset = now
	}

	// Check limits
	if count.Minute >= maxPerMin {
		return WrapRateLimitError(
			"per-minute rate limit exceeded",
			ErrRateLimitExceeded,
		)
	}
	if count.Hour >= maxPerHour {
		return WrapRateLimitError(
			"per-hour rate limit exceeded",
			ErrRateLimitExceeded,
		)
	}

	// Increment counters
	count.Minute++
	count.Hour++

	return nil
}

// Reset resets all rate limit counters for a project
func (r *RateLimiter) Reset(projectID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.requestCounts, projectID)
}

// ResetAll resets all rate limit counters
func (r *RateLimiter) ResetAll() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.requestCounts = make(map[string]*RequestCount)
}

// GetStats returns current rate limit statistics for a project
func (r *RateLimiter) GetStats(projectID string) (minuteCount, hourCount int, ok bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	count, exists := r.requestCounts[projectID]
	if !exists {
		return 0, 0, false
	}

	now := time.Now()

	// Check if counts are still valid
	minuteValid := now.Sub(count.LastMinuteReset) < time.Minute
	hourValid := now.Sub(count.LastHourReset) < time.Hour

	minuteCount = 0
	hourCount = 0

	if minuteValid {
		minuteCount = count.Minute
	}
	if hourValid {
		hourCount = count.Hour
	}

	return minuteCount, hourCount, true
}
