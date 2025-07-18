// Copyright 2024 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cache

import (
	"sync"
	"time"
)

const (
	DefaultJanitorInterval = 1 * time.Minute  // Default interval for the janitor to clean up expired items.
	DefaultExpiration      = 30 * time.Minute // Default expiration time for cache items (0 means no expiration).
)

// CacheItem holds a value and its expiration time.
type CacheItem struct {
	Value      any
	Expiration int64 // Unix nano timestamp. 0 means no expiration.
}

// isExpired checks if the item has expired.
func (item CacheItem) isExpired() bool {
	if item.Expiration == 0 {
		return false
	}
	return time.Now().UnixNano() > item.Expiration
}

// Cache is a simple thread-safe in-memory cache with self-cleaning.
type Cache struct {
	items map[string]CacheItem
	mu    sync.RWMutex
	stop  chan struct{}
}

// NewCache creates a new cache and starts a self-cleaning goroutine.
// The cleanup interval is set to 1 minute.
func NewCache() *Cache {
	return &Cache{
		items: make(map[string]CacheItem),
	}
}

// WithJanitor allows setting a custom janitor interval for the cache.
func (c *Cache) WithJanitor(interval time.Duration) *Cache {
	if c.stop != nil {
		// If a janitor is already running, we stop it before starting a new one.
		close(c.stop)
	}
	c.stop = make(chan struct{})

	if interval <= 0 {
		interval = DefaultJanitorInterval // Default to 1 minute if an invalid interval is provided.
	}

	go c.janitor(interval)
	return c
}

// Get retrieves an item from the cache. It returns the item and a boolean indicating if it was found and not expired.
func (c *Cache) Get(key string) (any, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	item, found := c.items[key]
	if !found || item.isExpired() {
		return nil, false
	}
	return item.Value, true
}

// Set adds an item to the cache with a specified time-to-live (TTL).
// If ttl is 0 or negative, the item never expires.
func (c *Cache) Set(key string, value any, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var expiration int64
	if ttl > 0 {
		expiration = time.Now().Add(ttl).UnixNano()
	}

	c.items[key] = CacheItem{
		Value:      value,
		Expiration: expiration,
	}
}

// Stop stops the cleaning goroutine. It's safe to call Stop multiple times.
func (c *Cache) Stop() {
	if c.stop != nil {
		close(c.stop)
		c.stop = nil
	}
}

// janitor runs periodically to clean up expired items.
func (c *Cache) janitor(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.deleteExpired()
		case <-c.stop:
			return
		}
	}
}

// deleteExpired removes all expired items from the cache.
func (c *Cache) deleteExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()
	for k, v := range c.items {
		if v.isExpired() {
			delete(c.items, k)
		}
	}
}
