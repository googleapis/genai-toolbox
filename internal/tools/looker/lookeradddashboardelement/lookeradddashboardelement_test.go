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

package lookeradddashboardelement_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/mcp-toolbox/internal/server"
	"github.com/googleapis/mcp-toolbox/internal/sources"
	"github.com/googleapis/mcp-toolbox/internal/testutils"
	"github.com/googleapis/mcp-toolbox/internal/tools"
	lkr "github.com/googleapis/mcp-toolbox/internal/tools/looker/lookeradddashboardelement"
	"github.com/googleapis/mcp-toolbox/internal/util/parameters"
	"github.com/looker-open-source/sdk-codegen/go/rtl"
	v4 "github.com/looker-open-source/sdk-codegen/go/sdk/v4"
)

type mockAddElementSource struct {
	settings *rtl.ApiSettings
	sdk      *v4.LookerSDK
}

func (m mockAddElementSource) SourceType() string                  { return "looker" }
func (m mockAddElementSource) ToConfig() sources.SourceConfig      { return nil }
func (m mockAddElementSource) IsReadOnly() bool                    { return false }
func (m mockAddElementSource) UseClientAuthorization() bool        { return false }
func (m mockAddElementSource) GetAuthTokenHeaderName() string      { return "Authorization" }
func (m mockAddElementSource) LookerApiSettings() *rtl.ApiSettings { return m.settings }
func (m mockAddElementSource) GetLookerSDK(context.Context, string) (*v4.LookerSDK, error) {
	return m.sdk, nil
}

func TestParseFromYamlLookerAddDashboardElement(t *testing.T) {
	ctx, err := testutils.ContextWithNewLogger()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	tcs := []struct {
		desc string
		in   string
		want server.ToolConfigs
	}{
		{
			desc: "basic example",
			in: `
            kind: tool
            name: example_tool
            type: looker-add-dashboard-element
            source: my-instance
            description: some description
				`,
			want: server.ToolConfigs{
				"example_tool": lkr.Config{
					ConfigBase: tools.ConfigBase{
						Name:         "example_tool",
						Description:  "some description",
						AuthRequired: []string{},
					},
					Type:   "looker-add-dashboard-element",
					Source: "my-instance",
				},
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			// Parse contents
			_, _, _, got, _, _, _, _, err := server.UnmarshalPrimitiveConfig(ctx, testutils.FormatYaml(tc.in))
			if err != nil {
				t.Fatalf("unable to unmarshal: %s", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("incorrect parse: diff %v", diff)
			}
		})
	}

}

func TestFailParseFromYamlLookerAddDashboardElement(t *testing.T) {
	ctx, err := testutils.ContextWithNewLogger()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	tcs := []struct {
		desc string
		in   string
		err  string
	}{
		{
			desc: "Invalid method",
			in: `
            kind: tool
            name: example_tool
            type: looker-add-dashboard-element
            source: my-instance
            method: GOT
            description: some description
			`,
			err: "error unmarshaling tool: unable to parse tool \"example_tool\" as type \"looker-add-dashboard-element\": [3:1] unknown field \"method\"\n   1 | authRequired: []\n   2 | description: some description\n>  3 | method: GOT\n       ^\n   4 | name: example_tool\n   5 | source: my-instance\n   6 | type: looker-add-dashboard-element",
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			// Parse contents
			_, _, _, _, _, _, _, _, err := server.UnmarshalPrimitiveConfig(ctx, testutils.FormatYaml(tc.in))
			if err == nil {
				t.Fatalf("expect parsing to fail")
			}
			errStr := err.Error()
			if !strings.Contains(errStr, tc.err) {
				t.Fatalf("unexpected error string: got %q, want substring %q", errStr, tc.err)
			}
		})
	}

}

func TestInvokeDashboardFilters(t *testing.T) {
	ctx, err := testutils.ContextWithNewLogger()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tcs := []struct {
		desc          string
		dashFilters   any
		wantListen    []map[string]any
		wantNilFilter bool
	}{
		{
			desc: "groups multiple dashboard filters into a single filterable listen array",
			dashFilters: []any{
				map[string]any{"dashboard_filter_name": "Date", "field": "order_items.created_date"},
				map[string]any{"dashboard_filter_name": "State", "field": "users.state"},
			},
			wantListen: []map[string]any{
				{"dashboard_filter_name": "Date", "field": "order_items.created_date"},
				{"dashboard_filter_name": "State", "field": "users.state"},
			},
		},
		{
			desc:          "omits filterables entries when no dashboard filters are specified",
			dashFilters:   []any{},
			wantNilFilter: true,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			var capturedElementBody map[string]any
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch {
				case r.Method == http.MethodPost && r.URL.Path == "/api/4.0/queries":
					_, _ = w.Write([]byte(`{"id":"101"}`))
				case r.Method == http.MethodPost && r.URL.Path == "/api/4.0/dashboard_elements":
					bodyBytes, readErr := io.ReadAll(r.Body)
					if readErr != nil {
						t.Errorf("failed reading body: %v", readErr)
					}
					if uErr := json.Unmarshal(bodyBytes, &capturedElementBody); uErr != nil {
						t.Errorf("failed unmarshaling body: %v", uErr)
					}
					_, _ = w.Write([]byte(`{"id":"202","title":"Revenue Tile"}`))
				default:
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			defer ts.Close()

			settings := &rtl.ApiSettings{BaseUrl: ts.URL}
			session := rtl.NewAuthSession(*settings)
			session.Client = *ts.Client()
			sdk := v4.NewLookerSDK(session)

			cfg := lkr.Config{
				ConfigBase: tools.ConfigBase{
					Name:        "add_dashboard_element",
					Description: "Add dashboard element",
				},
				Type:   "looker-add-dashboard-element",
				Source: "my-instance",
			}
			toolInstance, err := cfg.Initialize(ctx)
			if err != nil {
				t.Fatalf("Initialize failed: %v", err)
			}

			toolParams, err := toolInstance.GetParameters(nil)
			if err != nil {
				t.Fatalf("GetParameters failed: %v", err)
			}
			params, pErr := parameters.ParseParams(toolParams, map[string]any{
				"dashboard_id":      "10",
				"title":             "Revenue Tile",
				"model":             "thelook",
				"explore":           "order_items",
				"fields":            []any{"order_items.created_date", "users.state", "order_items.total_sale_price"},
				"dashboard_filters": tc.dashFilters,
			}, nil)
			if pErr != nil {
				t.Fatalf("ParseParams failed: %v", pErr)
			}

			_, tbErr := toolInstance.Invoke(ctx, mockAddElementSource{settings: settings, sdk: sdk}, params, "")
			if tbErr != nil {
				t.Fatalf("Invoke failed: %v", tbErr)
			}

			rm, ok := capturedElementBody["result_maker"].(map[string]any)
			if !ok {
				t.Fatalf("expected result_maker in request body, got %v", capturedElementBody)
			}
			if tc.wantNilFilter {
				if rm["filterables"] != nil {
					t.Fatalf("expected nil filterables when no dashboard_filters are provided, got %v", rm["filterables"])
				}
				return
			}

			filterables, ok := rm["filterables"].([]any)
			if !ok || len(filterables) != 1 {
				t.Fatalf("expected exactly 1 ResultMakerFilterables entry, got %v", rm["filterables"])
			}
			fb0, ok := filterables[0].(map[string]any)
			if !ok {
				t.Fatalf("expected filterables[0] to be map, got %T", filterables[0])
			}
			listenRaw, ok := fb0["listen"].([]any)
			if !ok {
				t.Fatalf("expected filterables[0].listen to be slice, got %v", fb0["listen"])
			}
			gotListen := make([]map[string]any, 0, len(listenRaw))
			for _, item := range listenRaw {
				m, _ := item.(map[string]any)
				gotListen = append(gotListen, m)
			}
			if diff := cmp.Diff(tc.wantListen, gotListen); diff != "" {
				t.Errorf("filterables[0].listen mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
