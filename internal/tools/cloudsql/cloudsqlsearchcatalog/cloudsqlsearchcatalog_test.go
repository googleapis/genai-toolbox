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

package cloudsqlsearchcatalog_test

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/mcp-toolbox/internal/server"
	"github.com/googleapis/mcp-toolbox/internal/sources"
	"github.com/googleapis/mcp-toolbox/internal/sources/dataplex/searchcatalog"
	"github.com/googleapis/mcp-toolbox/internal/testutils"
	"github.com/googleapis/mcp-toolbox/internal/tools"
	"github.com/googleapis/mcp-toolbox/internal/tools/cloudsql/cloudsqlsearchcatalog"
	"github.com/googleapis/mcp-toolbox/internal/util/parameters"
)

func TestParseFromYamlCloudSQLSearch(t *testing.T) {
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
			desc: "mysql example",
			in: `
            kind: tool
            name: example_tool
            type: mysql-search-catalog
            source: my-instance
            description: some description
            `,
			want: server.ToolConfigs{
				"example_tool": cloudsqlsearchcatalog.Config{
					Name:         "example_tool",
					Type:         "mysql-search-catalog",
					Source:       "my-instance",
					Description:  "some description",
					AuthRequired: []string{},
				},
			},
		},
		{
			desc: "mssql example",
			in: `
            kind: tool
            name: example_tool_mssql
            type: mssql-search-catalog
            source: my-mssql-instance
            description: some mssql description
            `,
			want: server.ToolConfigs{
				"example_tool_mssql": cloudsqlsearchcatalog.Config{
					Name:         "example_tool_mssql",
					Type:         "mssql-search-catalog",
					Source:       "my-mssql-instance",
					Description:  "some mssql description",
					AuthRequired: []string{},
				},
			},
		},
		{
			desc: "postgres example",
			in: `
            kind: tool
            name: example_tool_pg
            type: postgres-search-catalog
            source: my-pg-instance
            description: some pg description
            `,
			want: server.ToolConfigs{
				"example_tool_pg": cloudsqlsearchcatalog.Config{
					Name:         "example_tool_pg",
					Type:         "postgres-search-catalog",
					Source:       "my-pg-instance",
					Description:  "some pg description",
					AuthRequired: []string{},
				},
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			// Parse contents
			_, _, _, got, _, _, err := server.UnmarshalResourceConfig(ctx, testutils.FormatYaml(tc.in))
			if err != nil {
				t.Fatalf("unable to unmarshal: %s", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("incorrect parse: diff %v", diff)
			}
		})
	}
}

type mockCloudSQLSource struct {
	projectID              string
	useClientAuthorization bool
	searchResponse         []searchcatalog.DataplexSearchResponse
	err                    error
}

func (m mockCloudSQLSource) ProjectID() string            { return m.projectID }
func (m mockCloudSQLSource) UseClientAuthorization() bool { return m.useClientAuthorization }
func (m mockCloudSQLSource) InvokeSearchCatalog(ctx context.Context, params map[string]any, tokenStr string) ([]searchcatalog.DataplexSearchResponse, error) {
	return m.searchResponse, m.err
}
func (m mockCloudSQLSource) SourceType() string             { return "cloud-sql-postgres" }
func (m mockCloudSQLSource) ToConfig() sources.SourceConfig { return nil }

type mockSourceProvider struct {
	source sources.Source
}

func (m mockSourceProvider) GetSource(name string) (sources.Source, bool) {
	if m.source != nil {
		return m.source, true
	}
	return nil, false
}

func TestConfig_Initialize(t *testing.T) {
	cfg := cloudsqlsearchcatalog.Config{
		Name:        "test-tool",
		Type:        "postgres-search-catalog",
		Source:      "test-source",
		Description: "Test description",
	}

	tool, err := cfg.Initialize(nil)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	if tool.GetName() != "test-tool" {
		t.Errorf("GetName() = %v, want %v", tool.GetName(), "test-tool")
	}

	if tool.GetDescription() != "Test description" {
		t.Errorf("GetDescription() = %v, want %v", tool.GetDescription(), "Test description")
	}
}

func TestTool_Invoke(t *testing.T) {
	ctx := context.Background()
	mockSource := mockCloudSQLSource{
		searchResponse: []searchcatalog.DataplexSearchResponse{
			{
				DataplexEntry: "test-entry",
			},
		},
	}
	sourceProvider := mockSourceProvider{source: mockSource}

	cfg := cloudsqlsearchcatalog.Config{
		Name:   "test-tool",
		Type:   "postgres-search-catalog",
		Source: "test-source",
	}
	tool, err := cfg.Initialize(nil)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	params := parameters.ParamValues{
		{
			Name:  "prompt",
			Value: "test prompt",
		},
	}

	resp, err := tool.Invoke(ctx, sourceProvider, params, tools.AccessToken(""))
	if err != nil {
		t.Fatalf("Invoke failed: %v", err)
	}

	results, ok := resp.([]searchcatalog.DataplexSearchResponse)
	if !ok {
		t.Fatalf("expected []searchcatalog.DataplexSearchResponse, got %T", resp)
	}

	if len(results) != 1 || results[0].DataplexEntry != "test-entry" {
		t.Errorf("unexpected results: %v", results)
	}
}
