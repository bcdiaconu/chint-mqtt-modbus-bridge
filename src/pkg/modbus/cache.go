package modbus

import (
	"sync"
	"time"
)

// ValueCache stores cached command results with TTL
type ValueCache struct {
	cache map[string]*CachedResult
	mutex sync.RWMutex
	ttl   time.Duration
}

// NewValueCache creates a new value cache
func NewValueCache(ttl time.Duration) *ValueCache {
	return &ValueCache{
		cache: make(map[string]*CachedResult),
		ttl:   ttl,
	}
}

// Get retrieves a cached value if it exists and is not expired
func (c *ValueCache) Get(key string) (*CommandResult, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	cached, exists := c.cache[key]
	if !exists {
		return nil, false
	}

	// Check if expired
	if time.Since(cached.Timestamp) > c.ttl {
		return nil, false
	}

	return cached.Result, true
}

// Set stores a value in the cache
func (c *ValueCache) Set(key string, result *CommandResult) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.cache[key] = &CachedResult{
		Result:    result,
		Timestamp: time.Now(),
	}
}

// Clear removes all cached values
func (c *ValueCache) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.cache = make(map[string]*CachedResult)
}

// GetAll returns all cached values (for debugging)
func (c *ValueCache) GetAll() map[string]*CachedResult {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	// Return a copy to avoid race conditions
	result := make(map[string]*CachedResult, len(c.cache))
	for k, v := range c.cache {
		result[k] = v
	}
	return result
}
