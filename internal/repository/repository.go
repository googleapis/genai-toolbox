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

package repository

import (
	"time"
)

type Repository interface {
	Create(r Resource) error
	Update(r Resource) error  // if entity is not present, it will run Create
	Delete(name string) error // name is unique
	GetAll() ([]Resource, error)
	GetByName(name string) (Resource, error) // name is unique
}

type ResourceMetadata struct {
	// Optional: Time the resource was marked for deletion
	DeletionTimestamp time.Time
	// Optional: Indicate if the deletion is blocked by a tool
	DeletionBlocked bool
}

// Resource can represent either source, authService, tool or toolset
type Resource struct {
	// The name of the resource
	Name string
	// The type of the resource (e.g. alloydb-postgres)
	Type string
	// The json configuration of the resource
	Configuration string // json configuration
	// Indication on whether the resource is active, defaulted to true
	IsActive bool // default: true
	Metadata ResourceMetadata
}
