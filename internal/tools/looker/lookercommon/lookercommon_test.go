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

package lookercommon_test

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/internal/tools/looker/lookercommon"
	v4 "github.com/looker-open-source/sdk-codegen/go/sdk/v4"
)

func TestExtractLookerFieldProperties(t *testing.T) {
	ctx, err := testutils.ContextWithNewLogger()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	// Helper function to create string pointers
	stringPtr := func(s string) *string { return &s }
	stringArrayPtr := func(s []string) *[]string { return &s }
	boolPtr := func(b bool) *bool { return &b }

	tcs := []struct {
		desc   string
		fields []v4.LookmlModelExploreField
		want   []any
	}{
		{
			desc: "field with all properties including description",
			fields: []v4.LookmlModelExploreField{
				{
					Name:             stringPtr("dimension_name"),
					Type:             stringPtr("string"),
					Label:            stringPtr("Dimension Label"),
					LabelShort:       stringPtr("Dim Label"),
					Description:      stringPtr("This is a dimension description"),
					Suggestable:      boolPtr(true),
					SuggestExplore:   stringPtr("explore"),
					SuggestDimension: stringPtr("dimension"),
					Suggestions:      stringArrayPtr([]string{"foo", "bar", "baz"}),
				},
			},
			want: []any{
				map[string]any{
					"name":              "dimension_name",
					"type":              "string",
					"label":             "Dimension Label",
					"label_short":       "Dim Label",
					"description":       "This is a dimension description",
					"suggest_explore":   "explore",
					"suggest_dimension": "dimension",
					"suggestions":       []string{"foo", "bar", "baz"},
				},
			},
		},
		{
			desc: "field with missing description",
			fields: []v4.LookmlModelExploreField{
				{
					Name:       stringPtr("dimension_name"),
					Type:       stringPtr("string"),
					Label:      stringPtr("Dimension Label"),
					LabelShort: stringPtr("Dim Label"),
					// Description is nil
				},
			},
			want: []any{
				map[string]any{
					"name":        "dimension_name",
					"type":        "string",
					"label":       "Dimension Label",
					"label_short": "Dim Label",
					// description should not be present in the map
				},
			},
		},
		{
			desc: "field with only required fields",
			fields: []v4.LookmlModelExploreField{
				{
					Name: stringPtr("simple_dimension"),
					Type: stringPtr("number"),
				},
			},
			want: []any{
				map[string]any{
					"name": "simple_dimension",
					"type": "number",
				},
			},
		},
		{
			desc:   "empty fields list",
			fields: []v4.LookmlModelExploreField{},
			want:   []any{},
		},
		{
			desc: "multiple fields with mixed properties",
			fields: []v4.LookmlModelExploreField{
				{
					Name:        stringPtr("dim1"),
					Type:        stringPtr("string"),
					Label:       stringPtr("First Dimension"),
					Description: stringPtr("First dimension description"),
				},
				{
					Name:       stringPtr("dim2"),
					Type:       stringPtr("number"),
					LabelShort: stringPtr("Dim2"),
				},
			},
			want: []any{
				map[string]any{
					"name":        "dim1",
					"type":        "string",
					"label":       "First Dimension",
					"description": "First dimension description",
				},
				map[string]any{
					"name":        "dim2",
					"type":        "number",
					"label_short": "Dim2",
				},
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			got, err := lookercommon.ExtractLookerFieldProperties(ctx, &tc.fields, true)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("incorrect result: diff %v", diff)
			}
		})
	}
}

func TestExtractLookerFieldPropertiesWithNilFields(t *testing.T) {
	ctx, err := testutils.ContextWithNewLogger()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	got, err := lookercommon.ExtractLookerFieldProperties(ctx, nil, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []any{}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("incorrect result: diff %v", diff)
	}
}

//type QueryApiClientContext struct {
//	Name            string            `json:"name"`
//	Attributes      map[string]string `json:"attributes,omitempty"`
//	ExtraAttributes map[string]string `json:"extra_attributes,omitempty"`
//}
//
//type RenderOptions struct {
//	Format string `json:"format"`
//}
//
//type RequestRunInlineQuery2 struct {
//	Query             v4.WriteQuery         `json:"query"`
//	RenderOpts        RenderOptions         `json:"render_options"`
//	QueryApiClientCtx QueryApiClientContext `json:"query_api_client_context"`
//}

func TestRequestRunInlineQuery2(t *testing.T) {
	fields := make([]string, 1)
	fields[0] = "foo.bar"
	wq := v4.WriteQuery{
		Model:  "model",
		View:   "explore",
		Fields: &fields,
	}
	req2 := lookercommon.RequestRunInlineQuery2{
		Query: wq,
		RenderOpts: lookercommon.RenderOptions{
			Format: "json",
		},
		QueryApiClientCtx: lookercommon.QueryApiClientContext{
			Name: "MCP Toolbox",
		},
	}
	json, err := json.Marshal(req2)
	if err != nil {
		t.Fatalf("Could not marshall req2 as json")
	}
	got := string(json)
	want := `{"query":{"model":"model","view":"explore","fields":["foo.bar"]},"render_options":{"format":"json"},"query_api_client_context":{"name":"MCP Toolbox"}}`
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("incorrect result: diff %v", diff)
	}

}
