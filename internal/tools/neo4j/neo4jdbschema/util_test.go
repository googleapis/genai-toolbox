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

package neo4jdbschema

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
)

func TestHelperFunctions(t *testing.T) {
	t.Run("convertToStringSlice", func(t *testing.T) {
		tests := []struct {
			name  string
			input []any
			want  []string
		}{
			{
				name:  "empty slice",
				input: []any{},
				want:  []string{},
			},
			{
				name:  "string values",
				input: []any{"a", "b", "c"},
				want:  []string{"a", "b", "c"},
			},
			{
				name:  "mixed types",
				input: []any{"string", 123, true, 45.67},
				want:  []string{"string", "123", "true", "45.67"},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got := convertToStringSlice(tt.input)
				assert.Equal(t, tt.want, got)
			})
		}
	})

	t.Run("getStringValue", func(t *testing.T) {
		tests := []struct {
			name  string
			input any
			want  string
		}{
			{
				name:  "nil value",
				input: nil,
				want:  "",
			},
			{
				name:  "string value",
				input: "test",
				want:  "test",
			},
			{
				name:  "int value",
				input: 42,
				want:  "42",
			},
			{
				name:  "bool value",
				input: true,
				want:  "true",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got := getStringValue(tt.input)
				assert.Equal(t, tt.want, got)
			})
		}
	})
}

func TestMapToAPOCSchema(t *testing.T) {
	tests := []struct {
		name    string
		input   map[string]any
		want    *APOCSchemaResult
		wantErr bool
	}{
		{
			name: "simple node schema",
			input: map[string]any{
				"Person": map[string]any{
					"type":  "node",
					"count": int64(150),
					"properties": map[string]any{
						"name": map[string]any{
							"type":      "STRING",
							"unique":    false,
							"indexed":   true,
							"existence": false,
						},
					},
				},
			},
			want: &APOCSchemaResult{
				Value: map[string]APOCEntity{
					"Person": {
						Type:  "node",
						Count: 150,
						Properties: map[string]APOCProperty{
							"name": {
								Type:      "STRING",
								Unique:    false,
								Indexed:   true,
								Existence: false,
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name:    "empty input",
			input:   map[string]any{},
			want:    &APOCSchemaResult{Value: map[string]APOCEntity{}},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := mapToAPOCSchema(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("mapToAPOCSchema() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("mapToAPOCSchema() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestProcessAPOCSchema(t *testing.T) {
	tests := []struct {
		name      string
		input     *APOCSchemaResult
		wantNodes []NodeLabel
		wantStats *Statistics
	}{
		{
			name: "empty schema",
			input: &APOCSchemaResult{
				Value: map[string]APOCEntity{},
			},
			wantNodes: nil,
			wantStats: &Statistics{
				NodesByLabel:        map[string]int64{},
				RelationshipsByType: map[string]int64{},
				PropertiesByLabel:   map[string]int64{},
				PropertiesByRelType: map[string]int64{},
			},
		},
		{
			name: "simple node only",
			input: &APOCSchemaResult{
				Value: map[string]APOCEntity{
					"Person": {
						Type:  "node",
						Count: 100,
						Properties: map[string]APOCProperty{
							"name": {Type: "STRING", Indexed: true},
							"age":  {Type: "INTEGER"},
						},
					},
				},
			},
			wantNodes: []NodeLabel{
				{
					Name:  "Person",
					Count: 100,
					Properties: []PropertyInfo{
						{Name: "age", Types: []string{"INTEGER"}},
						{Name: "name", Types: []string{"STRING"}, Indexed: true},
					},
				},
			},
			wantStats: &Statistics{
				NodesByLabel:        map[string]int64{"Person": 100},
				RelationshipsByType: map[string]int64{},
				PropertiesByLabel:   map[string]int64{"Person": 200},
				PropertiesByRelType: map[string]int64{},
				TotalNodes:          100,
				TotalProperties:     200,
			},
		},
		{
			name: "relationship is ignored",
			input: &APOCSchemaResult{
				Value: map[string]APOCEntity{
					"FOLLOWS": {Type: "relationship", Count: 50},
				},
			},
			wantNodes: nil,
			wantStats: &Statistics{
				NodesByLabel:        map[string]int64{},
				RelationshipsByType: map[string]int64{},
				PropertiesByLabel:   map[string]int64{},
				PropertiesByRelType: map[string]int64{},
			},
		},
		{
			name: "nodes and relationships, only nodes are processed",
			input: &APOCSchemaResult{
				Value: map[string]APOCEntity{
					"Person": {
						Type:  "node",
						Count: 100,
						Properties: map[string]APOCProperty{
							"name": {Type: "STRING", Unique: true, Indexed: true, Existence: true},
						},
					},
					"Post": {
						Type:       "node",
						Count:      200,
						Properties: map[string]APOCProperty{"content": {Type: "STRING"}},
					},
					"FOLLOWS": {Type: "relationship", Count: 80},
				},
			},
			wantNodes: []NodeLabel{
				{
					Name:  "Post",
					Count: 200,
					Properties: []PropertyInfo{
						{Name: "content", Types: []string{"STRING"}},
					},
				},
				{
					Name:  "Person",
					Count: 100,
					Properties: []PropertyInfo{
						{Name: "name", Types: []string{"STRING"}, Unique: true, Indexed: true, Mandatory: true},
					},
				},
			},
			wantStats: &Statistics{
				NodesByLabel:        map[string]int64{"Person": 100, "Post": 200},
				RelationshipsByType: map[string]int64{},
				PropertiesByLabel:   map[string]int64{"Person": 100, "Post": 200},
				PropertiesByRelType: map[string]int64{},
				TotalNodes:          300,
				TotalProperties:     300,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotNodes, gotStats := processAPOCSchema(tt.input)

			if diff := cmp.Diff(tt.wantNodes, gotNodes); diff != "" {
				t.Errorf("processAPOCSchema() node labels mismatch (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(tt.wantStats, gotStats); diff != "" {
				t.Errorf("processAPOCSchema() statistics mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestProcessNoneAPOCSchema(t *testing.T) {
	t.Run("full schema processing", func(t *testing.T) {
		nodeCounts := map[string]int64{"Person": 10, "City": 5}
		nodePropsMap := map[string]map[string]map[string]bool{
			"Person": {"name": {"STRING": true}, "age": {"INTEGER": true}},
			"City":   {"name": {"STRING": true, "TEXT": true}},
		}
		relCounts := map[string]int64{"LIVES_IN": 8}
		relPropsMap := map[string]map[string]map[string]bool{
			"LIVES_IN": {"since": {"DATE": true}},
		}
		relConnectivity := map[string]struct {
			startNode string
			endNode   string
			count     int64
		}{
			"LIVES_IN": {startNode: "Person", endNode: "City", count: 8},
		}

		wantNodes := []NodeLabel{
			{
				Name:  "Person",
				Count: 10,
				Properties: []PropertyInfo{
					{Name: "age", Types: []string{"INTEGER"}},
					{Name: "name", Types: []string{"STRING"}},
				},
			},
			{
				Name:  "City",
				Count: 5,
				Properties: []PropertyInfo{
					{Name: "name", Types: []string{"STRING", "TEXT"}},
				},
			},
		}
		wantRels := []Relationship{
			{
				Type:      "LIVES_IN",
				Count:     8,
				StartNode: "Person",
				EndNode:   "City",
				Properties: []PropertyInfo{
					{Name: "since", Types: []string{"DATE"}},
				},
			},
		}
		wantStats := &Statistics{
			TotalNodes:          15,
			TotalRelationships:  8,
			TotalProperties:     (10*2 + 5*1) + (8 * 1), // 25 + 8 = 33
			NodesByLabel:        map[string]int64{"Person": 10, "City": 5},
			RelationshipsByType: map[string]int64{"LIVES_IN": 8},
			PropertiesByLabel:   map[string]int64{"Person": 2, "City": 1},
			PropertiesByRelType: map[string]int64{"LIVES_IN": 1},
		}

		gotNodes, gotRels, gotStats := processNoneAPOCSchema(nodeCounts, nodePropsMap, relCounts, relPropsMap, relConnectivity)

		if diff := cmp.Diff(wantNodes, gotNodes); diff != "" {
			t.Errorf("processNoneAPOCSchema() nodes mismatch (-want +got):\n%s", diff)
		}
		if diff := cmp.Diff(wantRels, gotRels); diff != "" {
			t.Errorf("processNoneAPOCSchema() relationships mismatch (-want +got):\n%s", diff)
		}
		if diff := cmp.Diff(wantStats, gotStats); diff != "" {
			t.Errorf("processNoneAPOCSchema() stats mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("empty schema", func(t *testing.T) {
		gotNodes, gotRels, gotStats := processNoneAPOCSchema(
			map[string]int64{},
			map[string]map[string]map[string]bool{},
			map[string]int64{},
			map[string]map[string]map[string]bool{},
			map[string]struct {
				startNode string
				endNode   string
				count     int64
			}{},
		)

		if len(gotNodes) != 0 {
			t.Errorf("expected 0 nodes, got %d", len(gotNodes))
		}
		if len(gotRels) != 0 {
			t.Errorf("expected 0 relationships, got %d", len(gotRels))
		}
		if diff := cmp.Diff(&Statistics{
			NodesByLabel:        map[string]int64{},
			RelationshipsByType: map[string]int64{},
			PropertiesByLabel:   map[string]int64{},
			PropertiesByRelType: map[string]int64{},
		}, gotStats); diff != "" {
			t.Errorf("processNoneAPOCSchema() stats mismatch (-want +got):\n%s", diff)
		}
	})
}
