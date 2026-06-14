// Copyright 2026 Google LLC
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

package searchcatalog_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/mcp-toolbox/internal/sources/dataplex/searchcatalog"
	"github.com/googleapis/mcp-toolbox/internal/util/parameters"
)

func TestConstructSearchQueryHelper(t *testing.T) {
	tests := []struct {
		name      string
		predicate string
		operator  string
		items     []string
		want      string
	}{
		{
			name:      "empty items",
			predicate: "projectid",
			operator:  "=",
			items:     []string{},
			want:      "",
		},
		{
			name:      "single item",
			predicate: "projectid",
			operator:  "=",
			items:     []string{"p1"},
			want:      "projectid=p1",
		},
		{
			name:      "multiple items",
			predicate: "projectid",
			operator:  "=",
			items:     []string{"p1", "p2"},
			want:      "(projectid=p1 OR projectid=p2)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := searchcatalog.ConstructSearchQueryHelper(tt.predicate, tt.operator, tt.items)
			if got != tt.want {
				t.Errorf("ConstructSearchQueryHelper() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConstructSearchQuery(t *testing.T) {
	tests := []struct {
		name       string
		prompt     string
		projectIds []string
		parentIds  []string
		types      []string
		system     string
		want       string
	}{
		{
			name:       "all empty",
			prompt:     "search",
			projectIds: []string{},
			parentIds:  []string{},
			types:      []string{},
			system:     "",
			want:       "search ",
		},
		{
			name:       "with project",
			prompt:     "search",
			projectIds: []string{"p1"},
			parentIds:  []string{},
			types:      []string{},
			system:     "",
			want:       "search projectid=p1",
		},
		{
			name:       "with all",
			prompt:     "search",
			projectIds: []string{"p1"},
			parentIds:  []string{"d1"},
			types:      []string{"t1"},
			system:     "sys1",
			want:       "search projectid=p1 AND parent=d1 AND type=t1 AND system=sys1",
		},
		{
			name:       "with multiple items",
			prompt:     "search",
			projectIds: []string{"p1", "p2"},
			parentIds:  []string{"d1"},
			types:      []string{},
			system:     "",
			want:       "search (projectid=p1 OR projectid=p2) AND parent=d1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := searchcatalog.ConstructSearchQuery(tt.prompt, tt.projectIds, tt.parentIds, tt.types, tt.system)
			if got != tt.want {
				t.Errorf("ConstructSearchQuery() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractType(t *testing.T) {
	typeMap := map[string]string{
		"type1": "MAPPED_TYPE1",
	}

	tests := []struct {
		name           string
		resourceString string
		want           string
	}{
		{
			name:           "found in map",
			resourceString: "projects/p/locations/l/entryTypes/type1",
			want:           "MAPPED_TYPE1",
		},
		{
			name:           "not found in map",
			resourceString: "projects/p/locations/l/entryTypes/type2",
			want:           "",
		},
		{
			name:           "no slash",
			resourceString: "type1",
			want:           "type1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := searchcatalog.ExtractType(tt.resourceString, typeMap)
			if got != tt.want {
				t.Errorf("ExtractType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConvertAnySliceToTyped(t *testing.T) {
	tests := []struct {
		name     string
		s        []any
		itemType string
		want     any
		wantErr  bool
	}{
		{
			name:     "strings",
			s:        []any{"a", "b"},
			itemType: "string",
			want:     []string{"a", "b"},
			wantErr:  false,
		},
		{
			name:     "integers",
			s:        []any{1, 2},
			itemType: "integer",
			want:     []int64{1, 2},
			wantErr:  false,
		},
		{
			name:     "floats",
			s:        []any{1.1, 2.2},
			itemType: "float",
			want:     []float64{1.1, 2.2},
			wantErr:  false,
		},
		{
			name:     "booleans",
			s:        []any{true, false},
			itemType: "boolean",
			want:     []bool{true, false},
			wantErr:  false,
		},
		{
			name:     "string error",
			s:        []any{"a", 1},
			itemType: "string",
			want:     nil,
			wantErr:  true,
		},
		{
			name:     "unknown type returns nil and no error",
			s:        []any{"a"},
			itemType: "unknown",
			want:     nil,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parameters.ConvertAnySliceToTyped(tt.s, tt.itemType)
			if (err != nil) != tt.wantErr {
				t.Errorf("ConvertAnySliceToTyped() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if diff := cmp.Diff(tt.want, got); diff != "" {
					t.Errorf("ConvertAnySliceToTyped() diff (-want +got):\n%s", diff)
				}
			}
		})
	}
}
