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
		wantRels  []Relationship
		wantStats *Statistics
	}{
		{
			name: "empty schema",
			input: &APOCSchemaResult{
				Value: map[string]APOCEntity{},
			},
			wantNodes: nil,
			wantRels:  nil,
			wantStats: &Statistics{
				NodesByLabel:        map[string]int64{},
				RelationshipsByType: map[string]int64{},
				PropertiesByLabel:   map[string]int64{},
				PropertiesByRelType: map[string]int64{},
				TotalNodes:          0,
				TotalRelationships:  0,
				TotalProperties:     0,
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
							"name": {
								Type:      "STRING",
								Unique:    false,
								Indexed:   true,
								Existence: false,
							},
							"age": {
								Type:      "INTEGER",
								Unique:    false,
								Indexed:   false,
								Existence: false,
							},
						},
					},
				},
			},
			wantNodes: []NodeLabel{
				{
					Name:  "Person",
					Count: 100,
					Properties: []PropertyInfo{
						{
							Name:      "age",
							Types:     []string{"INTEGER"},
							Unique:    false,
							Indexed:   false,
							Mandatory: false,
						},
						{
							Name:      "name",
							Types:     []string{"STRING"},
							Unique:    false,
							Indexed:   true,
							Mandatory: false,
						},
					},
				},
			},
			wantRels: nil,
			wantStats: &Statistics{
				NodesByLabel: map[string]int64{
					"Person": 100,
				},
				RelationshipsByType: map[string]int64{},
				PropertiesByLabel: map[string]int64{
					"Person": 200, // 2 properties * 100 nodes
				},
				PropertiesByRelType: map[string]int64{},
				TotalNodes:          100,
				TotalRelationships:  0,
				TotalProperties:     200,
			},
		},
		{
			name: "simple relationship only",
			input: &APOCSchemaResult{
				Value: map[string]APOCEntity{
					"FOLLOWS": {
						Type:  "relationship",
						Count: 50,
						Properties: map[string]APOCProperty{
							"since": {
								Type:      "DATE",
								Unique:    false,
								Indexed:   false,
								Existence: false,
							},
						},
					},
				},
			},
			wantNodes: nil,
			wantRels: []Relationship{
				{
					Type:  "FOLLOWS",
					Count: 50,
					Properties: []PropertyInfo{
						{
							Name:    "since",
							Types:   []string{"DATE"},
							Unique:  false,
							Indexed: false,
						},
					},
				},
			},
			wantStats: &Statistics{
				NodesByLabel:        map[string]int64{},
				RelationshipsByType: map[string]int64{"FOLLOWS": 50},
				PropertiesByLabel:   map[string]int64{},
				PropertiesByRelType: map[string]int64{"FOLLOWS": 50}, // 1 property * 50 rels
				TotalNodes:          0,
				TotalRelationships:  50,
				TotalProperties:     50,
			},
		},
		{
			name: "nodes and relationships with patterns",
			input: &APOCSchemaResult{
				Value: map[string]APOCEntity{
					"Person": {
						Type:  "node",
						Count: 100,
						Properties: map[string]APOCProperty{
							"name": {
								Type:      "STRING",
								Unique:    true,
								Indexed:   true,
								Existence: true,
							},
						},
						Relationships: map[string]APOCRelationshipInfo{
							"FOLLOWS": {
								Direction: "out",
								Labels:    []string{"Person"},
								Count:     80,
							},
						},
					},
					"Post": {
						Type:  "node",
						Count: 200,
						Properties: map[string]APOCProperty{
							"content": {
								Type:    "STRING",
								Indexed: false,
							},
						},
					},
					"FOLLOWS": {
						Type:  "relationship",
						Count: 80,
						Properties: map[string]APOCProperty{
							"since": {
								Type: "DATE",
							},
						},
					},
				},
			},
			wantNodes: []NodeLabel{
				{
					Name:  "Post",
					Count: 200,
					Properties: []PropertyInfo{
						{
							Name:      "content",
							Types:     []string{"STRING"},
							Indexed:   false,
							Unique:    false,
							Mandatory: false,
						},
					},
				},
				{
					Name:  "Person",
					Count: 100,
					Properties: []PropertyInfo{
						{
							Name:      "name",
							Types:     []string{"STRING"},
							Unique:    true,
							Indexed:   true,
							Mandatory: true,
						},
					},
				},
			},
			wantRels: []Relationship{
				{
					Type:      "FOLLOWS",
					Count:     80,
					StartNode: "Person",
					EndNode:   "Person",
					Properties: []PropertyInfo{
						{
							Name:    "since",
							Types:   []string{"DATE"},
							Unique:  false,
							Indexed: false,
						},
					},
				},
			},
			wantStats: &Statistics{
				NodesByLabel: map[string]int64{
					"Person": 100,
					"Post":   200,
				},
				RelationshipsByType: map[string]int64{
					"FOLLOWS": 80,
				},
				PropertiesByLabel: map[string]int64{
					"Person": 100, // 1 property * 100 nodes
					"Post":   200, // 1 property * 200 nodes
				},
				PropertiesByRelType: map[string]int64{
					"FOLLOWS": 80, // 1 property * 80 rels
				},
				TotalNodes:         300,
				TotalRelationships: 80,
				TotalProperties:    380,
			},
		},
		{
			name: "process schema from test.json",
			input: &APOCSchemaResult{
				Value: map[string]APOCEntity{
					"ASSIGNED_TO": {Type: "relationship", Count: 614391, Properties: map[string]APOCProperty{}},
					"BELONGS_TO":  {Type: "relationship", Count: 584877, Properties: map[string]APOCProperty{}},
					"Database": {
						Type:  "node",
						Count: 29376,
						Properties: map[string]APOCProperty{
							"dbid":    {Type: "STRING", Unique: true, Indexed: true},
							"db_name": {Type: "STRING"},
						},
						Relationships: map[string]APOCRelationshipInfo{
							"ASSIGNED_TO": {Direction: "out", Labels: []string{"Project"}, Count: 0},
						},
					},
					"Project": {
						Type:  "node",
						Count: 585016,
						Properties: map[string]APOCProperty{
							"project_id":   {Type: "STRING", Unique: true, Indexed: true},
							"project_name": {Type: "STRING"},
						},
						Relationships: map[string]APOCRelationshipInfo{
							"ASSIGNED_TO": {Direction: "out", Labels: []string{"Organization", "Database"}, Count: 50},
						},
					},
					"Organization": {
						Type:  "node",
						Count: 584702,
						Properties: map[string]APOCProperty{
							"org_id":           {Type: "STRING", Unique: true, Indexed: true},
							"org_display_name": {Type: "STRING"},
						},
						Relationships: map[string]APOCRelationshipInfo{
							"ASSIGNED_TO": {Direction: "in", Labels: []string{"Project"}, Count: 1046},
							"BELONGS_TO":  {Direction: "in", Labels: []string{"BillingAccount"}, Count: 1099},
						},
					},
				},
			},
			// Nodes should be sorted by count desc
			wantNodes: []NodeLabel{
				{
					Name:  "Project",
					Count: 585016,
					Properties: []PropertyInfo{ // Properties sorted alphabetically
						{Name: "project_id", Types: []string{"STRING"}, Unique: true, Indexed: true},
						{Name: "project_name", Types: []string{"STRING"}},
					},
				},
				{
					Name:  "Organization",
					Count: 584702,
					Properties: []PropertyInfo{
						{Name: "org_display_name", Types: []string{"STRING"}},
						{Name: "org_id", Types: []string{"STRING"}, Unique: true, Indexed: true},
					},
				},
				{
					Name:  "Database",
					Count: 29376,
					Properties: []PropertyInfo{
						{Name: "db_name", Types: []string{"STRING"}},
						{Name: "dbid", Types: []string{"STRING"}, Unique: true, Indexed: true},
					},
				},
			},
			// Relationships should be sorted by count desc
			wantRels: []Relationship{
				{
					Type: "ASSIGNED_TO", Count: 614391,
					// Pattern detection finds Project->Organization has highest count (50 vs 0)
					StartNode: "Project", EndNode: "Organization",
					Properties: []PropertyInfo{},
				},
				{
					Type:       "BELONGS_TO",
					Count:      584877,
					StartNode:  "", // No "out" relationships found for BELONGS_TO, so pattern is empty
					EndNode:    "",
					Properties: []PropertyInfo{},
				},
			},
			wantStats: &Statistics{
				TotalNodes:         1199094, // 585016 + 584702 + 29376
				TotalRelationships: 1199268, // 614391 + 584877
				TotalProperties:    2398188, // (585016*2 + 584702*2 + 29376*2) + (614391*0 + 584877*0)
				NodesByLabel: map[string]int64{
					"Project":      585016,
					"Organization": 584702,
					"Database":     29376,
				},
				RelationshipsByType: map[string]int64{
					"ASSIGNED_TO": 614391,
					"BELONGS_TO":  584877,
				},
				PropertiesByLabel: map[string]int64{
					"Project":      1170032, // 585016 * 2
					"Organization": 1169404, // 584702 * 2 -- CORRECTED
					"Database":     58752,   // 29376 * 2
				},
				PropertiesByRelType: map[string]int64{
					"ASSIGNED_TO": 0,
					"BELONGS_TO":  0,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotNodes, gotRels, gotStats := processAPOCSchema(tt.input)

			if diff := cmp.Diff(tt.wantNodes, gotNodes); diff != "" {
				t.Errorf("processAPOCSchema() node labels mismatch (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(tt.wantRels, gotRels); diff != "" {
				t.Errorf("processAPOCSchema() relationships mismatch (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(tt.wantStats, gotStats); diff != "" {
				t.Errorf("processAPOCSchema() statistics mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
