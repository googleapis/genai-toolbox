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

package helpers

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/tools/neo4j/neo4jschema/types"
)

func TestHelperFunctions(t *testing.T) {
	t.Run("ConvertToStringSlice", func(t *testing.T) {
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
				got := ConvertToStringSlice(tt.input)
				if diff := cmp.Diff(tt.want, got); diff != "" {
					t.Errorf("ConvertToStringSlice() mismatch (-want +got):\n%s", diff)
				}
			})
		}
	})

	t.Run("GetStringValue", func(t *testing.T) {
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
				got := GetStringValue(tt.input)
				if got != tt.want {
					t.Errorf("GetStringValue() got %q, want %q", got, tt.want)
				}
			})
		}
	})
}

func TestMapToAPOCSchema(t *testing.T) {
	tests := []struct {
		name    string
		input   map[string]any
		want    *types.APOCSchemaResult
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
			want: &types.APOCSchemaResult{
				Value: map[string]types.APOCEntity{
					"Person": {
						Type:  "node",
						Count: 150,
						Properties: map[string]types.APOCProperty{
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
			want:    &types.APOCSchemaResult{Value: map[string]types.APOCEntity{}},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MapToAPOCSchema(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("MapToAPOCSchema() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("MapToAPOCSchema() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestProcessAPOCSchema(t *testing.T) {
	tests := []struct {
		name          string
		input         *types.APOCSchemaResult
		wantNodes     []types.NodeLabel
		wantRels      []types.Relationship
		wantStats     *types.Statistics
		statsAreEmpty bool
	}{
		{
			name: "empty schema",
			input: &types.APOCSchemaResult{
				Value: map[string]types.APOCEntity{},
			},
			wantNodes:     nil,
			wantRels:      []types.Relationship{},
			statsAreEmpty: true,
		},
		{
			name: "simple node only",
			input: &types.APOCSchemaResult{
				Value: map[string]types.APOCEntity{
					"Person": {
						Type:  "node",
						Count: 100,
						Properties: map[string]types.APOCProperty{
							"name": {Type: "STRING", Indexed: true},
							"age":  {Type: "INTEGER"},
						},
					},
				},
			},
			wantNodes: []types.NodeLabel{
				{
					Name:  "Person",
					Count: 100,
					Properties: []types.PropertyInfo{
						{Name: "age", Types: []string{"INTEGER"}},
						{Name: "name", Types: []string{"STRING"}, Indexed: true},
					},
				},
			},
			wantRels: []types.Relationship{},
			wantStats: &types.Statistics{
				NodesByLabel:      map[string]int64{"Person": 100},
				PropertiesByLabel: map[string]int64{"Person": 2},
				TotalNodes:        100,
				TotalProperties:   200,
			},
		},
		{
			name: "nodes and relationships",
			input: &types.APOCSchemaResult{
				Value: map[string]types.APOCEntity{
					"Person": {
						Type:  "node",
						Count: 100,
						Properties: map[string]types.APOCProperty{
							"name": {Type: "STRING", Unique: true, Indexed: true, Existence: true},
						},
						Relationships: map[string]types.APOCRelationshipInfo{
							"KNOWS": {
								Direction: "out",
								Count:     50,
								Labels:    []string{"Person"},
								Properties: map[string]types.APOCProperty{
									"since": {Type: "INTEGER"},
								},
							},
						},
					},
					"Post": {
						Type:       "node",
						Count:      200,
						Properties: map[string]types.APOCProperty{"content": {Type: "STRING"}},
					},
					"FOLLOWS": {Type: "relationship", Count: 80},
				},
			},
			wantNodes: []types.NodeLabel{
				{
					Name:  "Post",
					Count: 200,
					Properties: []types.PropertyInfo{
						{Name: "content", Types: []string{"STRING"}},
					},
				},
				{
					Name:  "Person",
					Count: 100,
					Properties: []types.PropertyInfo{
						{Name: "name", Types: []string{"STRING"}, Unique: true, Indexed: true, Mandatory: true},
					},
				},
			},
			wantRels: []types.Relationship{
				{
					Type:      "KNOWS",
					StartNode: "Person",
					EndNode:   "Person",
					Count:     50,
					Properties: []types.PropertyInfo{
						{Name: "since", Types: []string{"INTEGER"}},
					},
				},
			},
			wantStats: &types.Statistics{
				NodesByLabel:        map[string]int64{"Person": 100, "Post": 200},
				RelationshipsByType: map[string]int64{"KNOWS": 50},
				PropertiesByLabel:   map[string]int64{"Person": 1, "Post": 1},
				PropertiesByRelType: map[string]int64{"KNOWS": 1},
				TotalNodes:          300,
				TotalRelationships:  50,
				TotalProperties:     350, // (100*1 + 200*1) for nodes + (50*1) for rels
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotNodes, gotRels, gotStats := ProcessAPOCSchema(tt.input)

			if diff := cmp.Diff(tt.wantNodes, gotNodes); diff != "" {
				t.Errorf("ProcessAPOCSchema() node labels mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.wantRels, gotRels); diff != "" {
				t.Errorf("ProcessAPOCSchema() relationships mismatch (-want +got):\n%s", diff)
			}
			if tt.statsAreEmpty {
				tt.wantStats = &types.Statistics{}
			}

			if diff := cmp.Diff(tt.wantStats, gotStats); diff != "" {
				t.Errorf("ProcessAPOCSchema() statistics mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestProcessNonAPOCSchema(t *testing.T) {
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
		relConnectivity := map[string]types.RelConnectivityInfo{
			"LIVES_IN": {StartNode: "Person", EndNode: "City", Count: 8},
		}

		wantNodes := []types.NodeLabel{
			{
				Name:  "Person",
				Count: 10,
				Properties: []types.PropertyInfo{
					{Name: "age", Types: []string{"INTEGER"}},
					{Name: "name", Types: []string{"STRING"}},
				},
			},
			{
				Name:  "City",
				Count: 5,
				Properties: []types.PropertyInfo{
					{Name: "name", Types: []string{"STRING", "TEXT"}},
				},
			},
		}
		wantRels := []types.Relationship{
			{
				Type:      "LIVES_IN",
				Count:     8,
				StartNode: "Person",
				EndNode:   "City",
				Properties: []types.PropertyInfo{
					{Name: "since", Types: []string{"DATE"}},
				},
			},
		}
		wantStats := &types.Statistics{
			TotalNodes:          15,
			TotalRelationships:  8,
			TotalProperties:     33, // (10*2 + 5*1) for nodes + (8*1) for rels
			NodesByLabel:        map[string]int64{"Person": 10, "City": 5},
			RelationshipsByType: map[string]int64{"LIVES_IN": 8},
			PropertiesByLabel:   map[string]int64{"Person": 2, "City": 1},
			PropertiesByRelType: map[string]int64{"LIVES_IN": 1},
		}

		gotNodes, gotRels, gotStats := ProcessNonAPOCSchema(nodeCounts, nodePropsMap, relCounts, relPropsMap, relConnectivity)

		if diff := cmp.Diff(wantNodes, gotNodes); diff != "" {
			t.Errorf("ProcessNonAPOCSchema() nodes mismatch (-want +got):\n%s", diff)
		}
		if diff := cmp.Diff(wantRels, gotRels); diff != "" {
			t.Errorf("ProcessNonAPOCSchema() relationships mismatch (-want +got):\n%s", diff)
		}
		if diff := cmp.Diff(wantStats, gotStats); diff != "" {
			t.Errorf("ProcessNonAPOCSchema() stats mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("empty schema", func(t *testing.T) {
		gotNodes, gotRels, gotStats := ProcessNonAPOCSchema(
			map[string]int64{},
			map[string]map[string]map[string]bool{},
			map[string]int64{},
			map[string]map[string]map[string]bool{},
			map[string]types.RelConnectivityInfo{},
		)

		if len(gotNodes) != 0 {
			t.Errorf("expected 0 nodes, got %d", len(gotNodes))
		}
		if len(gotRels) != 0 {
			t.Errorf("expected 0 relationships, got %d", len(gotRels))
		}
		if diff := cmp.Diff(&types.Statistics{}, gotStats); diff != "" {
			t.Errorf("ProcessNonAPOCSchema() stats mismatch (-want +got):\n%s", diff)
		}
	})
}
