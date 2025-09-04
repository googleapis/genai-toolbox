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

package memoryrepo

import (
	"fmt"
	"maps"
	"slices"
	"sync"

	"github.com/googleapis/genai-toolbox/internal/repository"
)

// MemoryRepository is the default repository that is used that uses local memory
type MemoryRepository struct {
	data map[string]repository.Resource
	mu   sync.RWMutex
}

// New initialize and creates new MemoryRepository for all the resources
func New() (
	*MemoryRepository,
	*MemoryRepository,
	*MemoryRepository,
	*MemoryRepository,
) {
	sourceR := &MemoryRepository{data: make(map[string]repository.Resource)}
	authServiceR := &MemoryRepository{data: make(map[string]repository.Resource)}
	toolR := &MemoryRepository{data: make(map[string]repository.Resource)}
	toolsetR := &MemoryRepository{data: make(map[string]repository.Resource)}
	return sourceR, authServiceR, toolR, toolsetR
}

// Create creates a new resource in MemoryRepository
func (r *MemoryRepository) Create(resource repository.Resource) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := resource.Name
	if _, exists := r.data[name]; exists {
		return fmt.Errorf("name %s already exists", name)
	}

	r.data[name] = resource
	return nil
}

func (r *MemoryRepository) Update(resource repository.Resource) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := resource.Name
	r.data[name] = resource
	return nil
}

func (r *MemoryRepository) Delete(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// In the future, we can implement soft delete and garbage collector
	// to clean up deleted datas
	delete(r.data, name)
	return nil
}

func (r *MemoryRepository) GetAll() ([]repository.Resource, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return slices.Collect(maps.Values(r.data)), nil
}

func (r *MemoryRepository) Get(name string) (repository.Resource, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var d repository.Resource
	d, exists := r.data[name]
	if !exists {
		return d, fmt.Errorf("unable to retrieve data: %s", name)
	}
	return d, nil
}
