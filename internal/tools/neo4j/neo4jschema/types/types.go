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

// Package types contains the shared data structures for Neo4j schema representation.
package types

// SchemaInfo represents the complete database schema.
type SchemaInfo struct {
	NodeLabels    []NodeLabel    `json:"nodeLabels"`
	Relationships []Relationship `json:"relationships"`
	Constraints   []Constraint   `json:"constraints"`
	Indexes       []Index        `json:"indexes"`
	DatabaseInfo  DatabaseInfo   `json:"databaseInfo"`
	Statistics    Statistics     `json:"statistics"`
	Errors        []string       `json:"errors,omitempty"`
}

// NodeLabel represents a node label with its properties.
type NodeLabel struct {
	Name       string         `json:"name"`
	Properties []PropertyInfo `json:"properties"`
	Count      int64          `json:"count"`
}

// RelConnectivityInfo holds information about a relationship's start and end nodes,
// primarily used during schema extraction without APOC procedures.
type RelConnectivityInfo struct {
	StartNode string
	EndNode   string
	Count     int64
}

// Relationship represents a relationship type with its properties.
type Relationship struct {
	Type       string         `json:"type"`
	Properties []PropertyInfo `json:"properties"`
	StartNode  string         `json:"startNode,omitempty"`
	EndNode    string         `json:"endNode,omitempty"`
	Count      int64          `json:"count"`
}

// PropertyInfo represents a property with its data types.
type PropertyInfo struct {
	Name      string   `json:"name"`
	Types     []string `json:"types"`
	Mandatory bool     `json:"-"`
	Unique    bool     `json:"-"`
	Indexed   bool     `json:"-"`
}

// Constraint represents a database constraint.
type Constraint struct {
	Name       string   `json:"name"`
	Type       string   `json:"type"`
	EntityType string   `json:"entityType"`
	Label      string   `json:"label,omitempty"`
	Properties []string `json:"properties"`
}

// Index represents a database index.
type Index struct {
	Name       string   `json:"name"`
	State      string   `json:"state"`
	Type       string   `json:"type"`
	EntityType string   `json:"entityType"`
	Label      string   `json:"label,omitempty"`
	Properties []string `json:"properties"`
}

// DatabaseInfo contains general database information.
type DatabaseInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Edition string `json:"edition,omitempty"`
}

// Statistics contains database statistics.
type Statistics struct {
	TotalNodes          int64            `json:"totalNodes"`
	TotalRelationships  int64            `json:"totalRelationships"`
	TotalProperties     int64            `json:"totalProperties"`
	NodesByLabel        map[string]int64 `json:"nodesByLabel"`
	RelationshipsByType map[string]int64 `json:"relationshipsByType"`
	PropertiesByLabel   map[string]int64 `json:"propertiesByLabel"`
	PropertiesByRelType map[string]int64 `json:"propertiesByRelType"`
}

// APOCSchemaResult represents the result from apoc.meta.schema().
type APOCSchemaResult struct {
	Value map[string]APOCEntity `json:"value"`
}

// APOCEntity represents a node or relationship in APOC schema.
type APOCEntity struct {
	Type          string                          `json:"type"`
	Count         int64                           `json:"count"`
	Labels        []string                        `json:"labels,omitempty"`
	Properties    map[string]APOCProperty         `json:"properties"`
	Relationships map[string]APOCRelationshipInfo `json:"relationships,omitempty"`
}

// APOCProperty represents property info from APOC.
type APOCProperty struct {
	Type      string `json:"type"`
	Indexed   bool   `json:"indexed"`
	Unique    bool   `json:"unique"`
	Existence bool   `json:"existence"`
}

// APOCRelationshipInfo represents relationship info from APOC.
type APOCRelationshipInfo struct {
	Count      int64                   `json:"count"`
	Direction  string                  `json:"direction"`
	Labels     []string                `json:"labels"`
	Properties map[string]APOCProperty `json:"properties"`
}
