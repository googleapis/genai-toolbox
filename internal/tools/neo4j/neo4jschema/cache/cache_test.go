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

package cache

import (
	"sync"
	"testing"
	"time"
)

// TestCache_SetAndGet verifies the basic functionality of setting a value
// and immediately retrieving it.
func TestCache_SetAndGet(t *testing.T) {
	cache := NewCache()
	defer cache.Stop()

	key := "testKey"
	value := "testValue"

	cache.Set(key, value, 1*time.Minute)

	retrievedValue, found := cache.Get(key)
	if !found {
		t.Errorf("Expected to find key %q, but it was not found", key)
	}

	if retrievedValue != value {
		t.Errorf("Expected value %q, but got %q", value, retrievedValue)
	}
}

// TestCache_GetExpired tests that an item is not retrievable after it has expired.
func TestCache_GetExpired(t *testing.T) {
	cache := NewCache()
	defer cache.Stop()

	key := "expiredKey"
	value := "expiredValue"

	// Set an item with a very short TTL.
	cache.Set(key, value, 1*time.Millisecond)
	time.Sleep(2 * time.Millisecond) // Wait for the item to expire.

	// Attempt to get the expired item.
	_, found := cache.Get(key)
	if found {
		t.Errorf("Expected key %q to be expired, but it was found", key)
	}
}

// TestCache_SetNoExpiration ensures that an item with a TTL of 0 or less
// does not expire.
func TestCache_SetNoExpiration(t *testing.T) {
	cache := NewCache()
	defer cache.Stop()

	key := "noExpireKey"
	value := "noExpireValue"

	cache.Set(key, value, 0) // Setting with 0 should mean no expiration.
	time.Sleep(5 * time.Millisecond)

	retrievedValue, found := cache.Get(key)
	if !found {
		t.Errorf("Expected to find key %q, but it was not found", key)
	}
	if retrievedValue != value {
		t.Errorf("Expected value %q, but got %q", value, retrievedValue)
	}
}

// TestCache_Janitor verifies that the janitor goroutine automatically removes
// expired items from the cache.
func TestCache_Janitor(t *testing.T) {
	// Initialize cache with a very short janitor interval for quick testing.
	cache := NewCache().WithJanitor(10 * time.Millisecond)
	defer cache.Stop()

	expiredKey := "expired"
	activeKey := "active"

	// Set one item that will expire and one that will not.
	cache.Set(expiredKey, "value", 1*time.Millisecond)
	cache.Set(activeKey, "value", 1*time.Hour)

	// Wait longer than the janitor interval to ensure it has a chance to run.
	time.Sleep(20 * time.Millisecond)

	// Check that the expired key has been removed.
	_, found := cache.Get(expiredKey)
	if found {
		t.Errorf("Expected janitor to clean up expired key %q, but it was found", expiredKey)
	}

	// Check that the active key is still present.
	_, found = cache.Get(activeKey)
	if !found {
		t.Errorf("Expected active key %q to be present, but it was not found", activeKey)
	}
}

// TestCache_Stop ensures that calling the Stop method does not cause a panic,
// regardless of whether the janitor is running or not. It also tests idempotency.
func TestCache_Stop(t *testing.T) {
	t.Run("Stop without janitor", func(t *testing.T) {
		cache := NewCache()
		// Test that calling Stop multiple times on a cache without a janitor is safe.
		cache.Stop()
		cache.Stop()
	})

	t.Run("Stop with janitor", func(t *testing.T) {
		cache := NewCache().WithJanitor(1 * time.Minute)
		// Test that calling Stop multiple times on a cache with a janitor is safe.
		cache.Stop()
		cache.Stop()
	})
}

// TestCache_Concurrent performs a stress test on the cache with concurrent
// reads and writes to check for race conditions.
func TestCache_Concurrent(t *testing.T) {
	cache := NewCache().WithJanitor(100 * time.Millisecond)
	defer cache.Stop()

	var wg sync.WaitGroup
	numGoroutines := 100
	numOperations := 1000

	// Start concurrent writer goroutines.
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				key := string(rune(g*numOperations + j))
				value := g*numOperations + j
				cache.Set(key, value, 10*time.Second)
			}
		}(i)
	}

	// Start concurrent reader goroutines.
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				key := string(rune(g*numOperations + j))
				cache.Get(key) // We don't check the result, just that access is safe.
			}
		}(i)
	}

	// Wait for all goroutines to complete. If a race condition exists, the Go
	// race detector (`go test -race`) will likely catch it.
	wg.Wait()
}
