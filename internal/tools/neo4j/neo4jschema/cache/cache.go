// Copyright 2025 Google LLC
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

/*
Package cache provides a simple, thread-safe, in-memory key-value store.
It features item expiration and an optional background process (janitor) that
periodically removes expired items.
*/
package cache

import (
	"sync"
	"time"
)

const (
	// DefaultJanitorInterval is the default interval at which the janitor
	// runs to clean up expired cache items.
	DefaultJanitorInterval = 1 * time.Minute
	// DefaultExpiration is the default time-to-live for a cache item.
	// Note: This constant is defined but not used in the current implementation,
	// as expiration is set on a per-item basis.
	DefaultExpiration = 60
)

// CacheItem represents a value stored in the cache, along with its expiration time.
type CacheItem struct {
	Value      any   // The actual value being stored.
	Expiration int64 // The time when the item expires, as a Unix nano timestamp. 0 means no expiration.
}

// isExpired checks if the cache item has passed its expiration time.
// It returns true if the item is expired, and false otherwise.
func (item CacheItem) isExpired() bool {
	// If Expiration is 0, the item is considered to never expire.
	if item.Expiration == 0 {
		return false
	}
	return time.Now().UnixNano() > item.Expiration
}

// Cache is a thread-safe, in-memory key-value store with self-cleaning capabilities.
type Cache struct {
	items map[string]CacheItem // The underlying map that stores the cache items.
	mu    sync.RWMutex         // A read/write mutex to ensure thread safety for concurrent access.
	stop  chan struct{}        // A channel used to signal the janitor goroutine to stop.
}

// NewCache creates and returns a new Cache instance.
// The janitor for cleaning up expired items is not started by default.
// Use the WithJanitor method to start the cleanup process.
//
// Example:
//
//	c := cache.NewCache()
//	c.Set("myKey", "myValue", 5*time.Minute)
func NewCache() *Cache {
	return &Cache{
		items: make(map[string]CacheItem),
	}
}

// WithJanitor starts a background goroutine (janitor) that periodically cleans up
// expired items from the cache. If a janitor is already running, it will be
// stopped and a new one will be started with the specified interval.
//
// The interval parameter defines how often the janitor should run. If a non-positive
// interval is provided, it defaults to DefaultJanitorInterval (1 minute).
//
// It returns a pointer to the Cache to allow for method chaining.
//
// Example:
//
//	// Create a cache that cleans itself every 10 minutes.
//	c := cache.NewCache().WithJanitor(10 * time.Minute)
//	defer c.Stop() // It's important to stop the janitor when the cache is no longer needed.
func (c *Cache) WithJanitor(interval time.Duration) *Cache {
	if c.stop != nil {
		// If a janitor is already running, stop it before starting a new one.
		close(c.stop)
	}
	c.stop = make(chan struct{})

	// Use the default interval if an invalid one is provided.
	if interval <= 0 {
		interval = DefaultJanitorInterval
	}

	// Start the janitor in a new goroutine.
	go c.janitor(interval)
	return c
}

// Get retrieves an item from the cache by its key.
// It returns the item's value and a boolean. The boolean is true if the key
// was found and the item has not expired. Otherwise, it is false.
//
// Example:
//
//	v, found := c.Get("myKey")
//	if found {
//		fmt.Printf("Found value: %v\n", v)
//	} else {
//		fmt.Println("Key not found or expired.")
//	}
func (c *Cache) Get(key string) (any, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, found := c.items[key]
	// Return false if the item is not found or if it is found but has expired.
	if !found || item.isExpired() {
		return nil, false
	}

	return item.Value, true
}

// Set adds an item to the cache, replacing any existing item with the same key.
//
// The `ttl` (time-to-live) parameter specifies how long the item should remain
// in the cache. If `ttl` is positive, the item will expire after that duration.
// If `ttl` is zero or negative, the item will never expire.
//
// Example:
//
//	// Add a key that expires in 5 minutes.
//	c.Set("sessionToken", "xyz123", 5*time.Minute)
//
//	// Add a key that never expires.
//	c.Set("appConfig", "configValue", 0)
func (c *Cache) Set(key string, value any, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var expiration int64
	// Calculate the expiration time only if ttl is positive.
	if ttl > 0 {
		expiration = time.Now().Add(ttl).UnixNano()
	}

	c.items[key] = CacheItem{
		Value:      value,
		Expiration: expiration,
	}
}

// Stop terminates the background janitor goroutine.
// It is safe to call Stop even if the janitor was never started or has already
// been stopped. This is useful for cleaning up resources.
func (c *Cache) Stop() {
	if c.stop != nil {
		close(c.stop)
		c.stop = nil
	}
}

// janitor is the background cleanup worker. It runs in a separate goroutine.
// It uses a time.Ticker to periodically trigger the deletion of expired items.
func (c *Cache) janitor(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Time to clean up expired items.
			c.deleteExpired()
		case <-c.stop:
			// Stop signal received, exit the goroutine.
			return
		}
	}
}

// deleteExpired scans the cache and removes all items that have expired.
// This function acquires a write lock on the cache to ensure safe mutation.
func (c *Cache) deleteExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for k, v := range c.items {
		if v.isExpired() {
			delete(c.items, k)
		}
	}
}
