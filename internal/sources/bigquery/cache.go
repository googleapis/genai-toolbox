// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package bigquery

import (
	"sync"
	"time"
)

// Item holds the cached value and its expiration timestamp
type Item struct {
	Value     any
	ExpiresAt int64 // Unix nano timestamp
}

// IsExpired checks if the item is expired
func (item Item) IsExpired() bool {
	return time.Now().UnixNano() > item.ExpiresAt
}

// OnEvictFunc is the signature for the callback
type OnEvictFunc func(key string, value any)

// Cache is a thread-safe, expiring key-value store
type Cache struct {
	mu      sync.RWMutex
	items   map[string]Item
	onEvict OnEvictFunc
}

// NewCache creates a new cache and cleans up every 55 min
func NewCache(onEvict OnEvictFunc) *Cache {
	const cleanupInterval = 55 * time.Minute

	c := &Cache{
		items:   make(map[string]Item),
		onEvict: onEvict,
	}

	go c.startCleanup(cleanupInterval)
	return c
}

// startCleanup runs a ticker to periodically delete expired items
func (c *Cache) startCleanup(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		c.DeleteExpired()
	}
}

// delete is an internal helper that assumes the write lock is held
func (c *Cache) delete(key string, item Item) {
	if c.onEvict != nil {
		c.onEvict(key, item.Value)
	}
	delete(c.items, key)
}

// Set adds an item to the cache
func (c *Cache) Set(key string, value any) {
	const ttl = 55 * time.Minute
	expires := time.Now().Add(ttl).UnixNano()

	c.mu.Lock()
	defer c.mu.Unlock()

	// If item already exists, evict the old one before replacing
	if oldItem, found := c.items[key]; found {
		c.delete(key, oldItem)
	}

	c.items[key] = Item{
		Value:     value,
		ExpiresAt: expires,
	}
}

// Get retrieves an item from the cache
func (c *Cache) Get(key string) (any, bool) {
	c.mu.RLock()
	item, found := c.items[key]
	if !found || item.IsExpired() {
		c.mu.RUnlock()
		return nil, false
	}
	c.mu.RUnlock()

	return item.Value, true
}

// Delete manually evicts an item
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if item, found := c.items[key]; found {
		c.delete(key, item)
	}
}

// DeleteExpired removes all expired items
func (c *Cache) DeleteExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for key, item := range c.items {
		if item.IsExpired() {
			c.delete(key, item)
		}
	}
}
