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
	"testing"
	"time"
)

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

func TestCache_GetExpired(t *testing.T) {
	cache := NewCache()
	defer cache.Stop()

	key := "expiredKey"
	value := "expiredValue"

	cache.Set(key, value, 1*time.Millisecond)
	time.Sleep(2 * time.Millisecond) // Wait for the item to expire

	_, found := cache.Get(key)
	if found {
		t.Errorf("Expected key %q to be expired, but it was found", key)
	}
}

func TestCache_SetNoExpiration(t *testing.T) {
	cache := NewCache()
	defer cache.Stop()

	key := "noExpireKey"
	value := "noExpireValue"

	cache.Set(key, value, 0)         // No expiration
	time.Sleep(5 * time.Millisecond) // Wait a bit

	retrievedValue, found := cache.Get(key)
	if !found {
		t.Errorf("Expected to find key %q, but it was not found", key)
	}
	if retrievedValue != value {
		t.Errorf("Expected value %q, but got %q", value, retrievedValue)
	}
}

func TestCache_Janitor(t *testing.T) {
	// Initialize cache with a very short janitor interval for testing
	cache := NewCache().WithJanitor(10 * time.Millisecond)
	defer cache.Stop()

	expiredKey := "expired"
	activeKey := "active"

	cache.Set(expiredKey, "value", 1*time.Millisecond)
	cache.Set(activeKey, "value", 1*time.Hour)

	// Wait longer than the janitor interval to ensure it runs
	time.Sleep(20 * time.Millisecond)

	_, found := cache.Get(expiredKey)
	if found {
		t.Errorf("Expected janitor to clean up expired key %q, but it was found", expiredKey)
	}

	_, found = cache.Get(activeKey)
	if !found {
		t.Errorf("Expected active key %q to be present, but it was not found", activeKey)
	}
}

func TestCache_Stop(t *testing.T) {
	t.Run("Stop without janitor", func(t *testing.T) {
		cache := NewCache()
		// Test that calling Stop multiple times doesn't panic
		cache.Stop()
		cache.Stop()
	})

	t.Run("Stop with janitor", func(t *testing.T) {
		cache := NewCache().WithJanitor(1 * time.Minute)
		// Test that calling Stop multiple times doesn't panic
		cache.Stop()
		cache.Stop()
	})
}

func TestCache_Concurrent(t *testing.T) {
	cache := NewCache().WithJanitor(100 * time.Millisecond)
	defer cache.Stop()

	var wg sync.WaitGroup
	numGoroutines := 100
	numOperations := 1000

	// Concurrent writes
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

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				key := string(rune(g*numOperations + j))
				cache.Get(key)
			}
		}(i)
	}

	wg.Wait()
}
