// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package lookercreatemergequery

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	v4 "github.com/looker-open-source/sdk-codegen/go/sdk/v4"
)

func TestProcessSourceQuery(t *testing.T) {
	t.Run("minimal source query", func(t *testing.T) {
		got, err := processSourceQuery(0, map[string]any{
			"model":   "thelook",
			"explore": "orders",
			"fields":  []any{"orders.created_date", "orders.count"},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.name != "orders" {
			t.Errorf("name = %q, want %q (defaults to the explore)", got.name, "orders")
		}
		if got.mergeFields != nil {
			t.Errorf("mergeFields = %v, want nil", got.mergeFields)
		}
		want := &v4.WriteQuery{
			Model:  "thelook",
			View:   "orders",
			Fields: &[]string{"orders.created_date", "orders.count"},
		}
		if diff := cmp.Diff(want, got.writeQuery); diff != "" {
			t.Errorf("unexpected write query: diff %v", diff)
		}
	})

	t.Run("all optional keys", func(t *testing.T) {
		got, err := processSourceQuery(1, map[string]any{
			"name":              "Web events",
			"model":             "thelook",
			"explore":           "events",
			"fields":            []any{"events.event_date", "events.count"},
			"filters":           map[string]any{`"events.event_date"`: `"30 days"`, "events.count": 3},
			"pivots":            []any{"events.event_date"},
			"sorts":             []any{"events.count desc 0"},
			"limit":             int64(100),
			"filter_expression": "${events.count} > 1",
			"tz":                "America/New_York",
			"dynamic_fields":    []any{map[string]any{"table_calculation": "double", "expression": "${events.count}*2"}},
			"merge_fields": []any{
				map[string]any{"source_field_name": "events.event_date", "field_name": "orders.created_date"},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.name != "Web events" {
			t.Errorf("name = %q, want %q", got.name, "Web events")
		}
		wantMergeFields := []v4.MergeFields{{
			FieldName:       strPtr("orders.created_date"),
			SourceFieldName: strPtr("events.event_date"),
		}}
		if diff := cmp.Diff(&wantMergeFields, got.mergeFields); diff != "" {
			t.Errorf("unexpected merge fields: diff %v", diff)
		}
		want := &v4.WriteQuery{
			Model:  "thelook",
			View:   "events",
			Fields: &[]string{"events.event_date", "events.count"},
			// Wrapping quotes are stripped from both keys and string values.
			Filters:          &map[string]any{"events.event_date": "30 days", "events.count": 3},
			Pivots:           &[]string{"events.event_date"},
			Sorts:            &[]string{"events.count desc 0"},
			Limit:            strPtr("100"),
			FilterExpression: strPtr("${events.count} > 1"),
			QueryTimezone:    strPtr("America/New_York"),
			DynamicFields:    strPtr(`[{"expression":"${events.count}*2","table_calculation":"double"}]`),
		}
		if diff := cmp.Diff(want, got.writeQuery); diff != "" {
			t.Errorf("unexpected write query: diff %v", diff)
		}
	})

	t.Run("empty defaults are omitted", func(t *testing.T) {
		got, err := processSourceQuery(0, map[string]any{
			"model":          "thelook",
			"explore":        "orders",
			"fields":         []any{"orders.count"},
			"dynamic_fields": []any{},
			"merge_fields":   []any{},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.writeQuery.DynamicFields != nil {
			t.Errorf("DynamicFields = %v, want nil", *got.writeQuery.DynamicFields)
		}
		if got.mergeFields != nil {
			t.Errorf("mergeFields = %v, want nil", got.mergeFields)
		}
	})
}

func TestProcessSourceQueryErrors(t *testing.T) {
	tcs := []struct {
		desc string
		in   any
		err  string
	}{
		{
			desc: "not an object",
			in:   "orders",
			err:  "source query #0 must be an object",
		},
		{
			desc: "unknown key",
			in: map[string]any{
				"model": "thelook", "explore": "orders", "fields": []any{"orders.count"},
				"explores": []any{"orders"}, "limits": 5,
			},
			err: `source query #0 has unknown keys [explores limits]`,
		},
		{
			desc: "missing model",
			in:   map[string]any{"explore": "orders", "fields": []any{"orders.count"}},
			err:  `source query #0 requires a non-empty string "model"`,
		},
		{
			desc: "empty explore",
			in:   map[string]any{"model": "thelook", "explore": "", "fields": []any{"orders.count"}},
			err:  `source query #0 requires a non-empty string "explore"`,
		},
		{
			desc: "missing fields",
			in:   map[string]any{"model": "thelook", "explore": "orders"},
			err:  `source query #0 requires "fields"`,
		},
		{
			desc: "empty fields",
			in:   map[string]any{"model": "thelook", "explore": "orders", "fields": []any{}},
			err:  `source query #0 requires at least one entry in "fields"`,
		},
		{
			desc: "non-string field",
			in:   map[string]any{"model": "thelook", "explore": "orders", "fields": []any{1}},
			err:  `"fields" element #0 must be a string, got int`,
		},
		{
			desc: "fractional limit",
			in: map[string]any{
				"model": "thelook", "explore": "orders", "fields": []any{"orders.count"},
				"limit": 1.5,
			},
			err: `"limit" must be an integer, got 1.5`,
		},
		{
			desc: "merge field with unknown key",
			in: map[string]any{
				"model": "thelook", "explore": "orders", "fields": []any{"orders.count"},
				"merge_fields": []any{map[string]any{"field": "orders.id"}},
			},
			err: `"merge_fields" element #0 has unknown key "field"`,
		},
		{
			desc: "incomplete merge field",
			in: map[string]any{
				"model": "thelook", "explore": "orders", "fields": []any{"orders.count"},
				"merge_fields": []any{map[string]any{"field_name": "orders.id"}},
			},
			err: `"merge_fields" element #0 requires both "field_name" and "source_field_name"`,
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			_, err := processSourceQuery(0, tc.in)
			if err == nil {
				t.Fatalf("expected an error, got none")
			}
			if !strings.Contains(err.Error(), tc.err) {
				t.Fatalf("unexpected error string: got %q, want substring %q", err.Error(), tc.err)
			}
		})
	}
}

func TestProcessSourceQueries(t *testing.T) {
	orders := map[string]any{"model": "thelook", "explore": "orders", "fields": []any{"orders.count"}}
	events := map[string]any{"model": "thelook", "explore": "events", "fields": []any{"events.count"}}

	t.Run("two source queries", func(t *testing.T) {
		got, err := processSourceQueries([]any{orders, events})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("got %d source queries, want 2", len(got))
		}
		if got[0].explore != "orders" || got[1].explore != "events" {
			t.Errorf("source query order not preserved: got %q then %q", got[0].explore, got[1].explore)
		}
	})

	tcs := []struct {
		desc string
		in   any
		err  string
	}{
		{
			desc: "not an array",
			in:   orders,
			err:  "'source_queries' must be an array of objects",
		},
		{
			desc: "only one query",
			in:   []any{orders},
			err:  "'source_queries' must contain at least two queries to merge, got 1",
		},
		{
			desc: "empty",
			in:   []any{},
			err:  "'source_queries' must contain at least two queries to merge, got 0",
		},
		{
			desc: "invalid second query",
			in:   []any{orders, map[string]any{"model": "thelook", "explore": "events"}},
			err:  `source query #1 requires "fields"`,
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			_, err := processSourceQueries(tc.in)
			if err == nil {
				t.Fatalf("expected an error, got none")
			}
			if !strings.Contains(err.Error(), tc.err) {
				t.Fatalf("unexpected error string: got %q, want substring %q", err.Error(), tc.err)
			}
		})
	}
}

func TestAsRowLimit(t *testing.T) {
	tcs := []struct {
		desc string
		in   any
		want string
	}{
		{desc: "int", in: 500, want: "500"},
		{desc: "int64", in: int64(500), want: "500"},
		{desc: "whole float", in: float64(500), want: "500"},
		{desc: "string", in: "500", want: "500"},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			got, err := asRowLimit("limit", tc.in)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("asRowLimit(%v) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}

	for _, in := range []any{"many", 1.5, true, nil} {
		if _, err := asRowLimit("limit", in); err == nil {
			t.Errorf("asRowLimit(%v) = nil error, want an error", in)
		}
	}
}

func TestMergeURL(t *testing.T) {
	tcs := []struct {
		desc    string
		hostURL string
		slugs   []string
		want    string
	}{
		{
			desc:    "two source queries",
			hostURL: "https://looker.example.com",
			slugs:   []string{"abc123", "def456"},
			want:    "https://looker.example.com/merge?qids%5B%5D=abc123&qids%5B%5D=def456",
		},
		{
			desc:    "trailing slash is trimmed",
			hostURL: "https://looker.example.com/",
			slugs:   []string{"abc123"},
			want:    "https://looker.example.com/merge?qids%5B%5D=abc123",
		},
		{
			desc:    "no host URL",
			hostURL: "",
			slugs:   []string{"abc123"},
			want:    "",
		},
		{
			desc:    "no slugs",
			hostURL: "https://looker.example.com",
			slugs:   nil,
			want:    "",
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			if got := mergeURL(tc.hostURL, tc.slugs); got != tc.want {
				t.Fatalf("mergeURL(%q, %v) = %q, want %q", tc.hostURL, tc.slugs, got, tc.want)
			}
		})
	}
}

func strPtr(s string) *string {
	return &s
}
