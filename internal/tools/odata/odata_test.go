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

package odata

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/googleapis/mcp-toolbox/internal/sources"
	"github.com/googleapis/mcp-toolbox/internal/sources/odata"
	"github.com/googleapis/mcp-toolbox/internal/tools"
	"github.com/googleapis/mcp-toolbox/internal/util/parameters"
)

// mockODataSource simulates an OData source for testing tool initialization
type mockODataSource struct {
	baseURL  string
	metadata *odata.ODataMetadata
}

func (m *mockODataSource) SourceType() string {
	return "odata"
}

func (m *mockODataSource) ToConfig() sources.SourceConfig {
	return nil
}

func (m *mockODataSource) HttpBaseURL() string {
	return m.baseURL
}

func (m *mockODataSource) RunODataRequest(req *http.Request, accessToken tools.AccessToken) (any, error) {
	return map[string]interface{}{"d": map[string]interface{}{"results": []interface{}{}}}, nil
}

func (m *mockODataSource) UseClientAuthorization() bool {
	return false
}

func (m *mockODataSource) Metadata() *odata.ODataMetadata {
	return m.metadata
}

func (m *mockODataSource) Compatibility() odata.CompatibilityConfig {
	return odata.CompatibilityConfig{}
}

func (m *mockODataSource) GetAuthTokenHeaderName() string {
	return "Authorization"
}

func TestToolInitializationREAD(t *testing.T) {
	// 1. Setup Mock Metadata
	metadata := &odata.ODataMetadata{
		Version: "2.0",
		EntityTypes: map[string]odata.EntityType{
			"A_SalesOrderType": {
				Name: "A_SalesOrderType",
				Properties: []odata.Property{
					{Name: "SalesOrder", Type: "Edm.String"},
					{Name: "TotalNetAmount", Type: "Edm.Decimal"},
				},
			},
		},
		EntitySets: map[string]odata.EntitySet{
			"A_SalesOrder": {Name: "A_SalesOrder", EntityType: "API_SALES_ORDER_SRV.A_SalesOrderType"},
		},
	}

	mockSrc := &mockODataSource{
		baseURL:  "https://mock.OData/odata",
		metadata: metadata,
	}

	srcs := map[string]sources.Source{
		"my_mock_odata": mockSrc,
	}

	yamlDef := []byte(`
name: read_sales
type: odata
source: my_mock_odata
entitySet: A_SalesOrder
operation: READ
description: Reads sales orders
`)

	var cfg map[string]interface{}
	if err := yaml.Unmarshal(yamlDef, &cfg); err != nil {
		t.Fatalf("Failed to decode YAML: %v", err)
	}
	var config Config
	if err := yaml.Unmarshal(yamlDef, &config); err != nil {
		t.Fatalf("Failed to decode struct: %v", err)
	}

	tool, err := config.Initialize(context.Background())
	if err != nil {
		t.Fatalf("Failed to initialize tool: %v", err)
	}

	ODataTool := tool.(Tool)
	if ODataTool.Method != "GET" {
		t.Errorf("Expected GET for READ operation, got %s", ODataTool.Method)
	}

	// Verify dynamic parameters
	params, err := ODataTool.GetParameters(srcs)
	if err != nil {
		t.Fatalf("Failed to get parameters: %v", err)
	}
	var filterParam parameters.Parameter
	var skipParam parameters.Parameter
	for _, p := range params {
		if p.GetName() == "filter" {
			filterParam = p
		}
		if p.GetName() == "skip" {
			skipParam = p
		}
	}

	if filterParam == nil {
		t.Fatalf("Expected dynamic 'filter' parameter missing")
	}

	// Verify that the filter description dynamically injected the metadata properties
	filterDesc := filterParam.Manifest().Description
	if filterDesc == "" {
		t.Errorf("Filter description should not be empty")
	}
	t.Logf("Generated Filter Desc: %s", filterDesc)

	if skipParam == nil {
		t.Fatalf("Expected dynamic 'skip' parameter missing")
	}
}

func TestApplyODataFormatting(t *testing.T) {
	compat := odata.CompatibilityConfig{UrlQuoting: true}

	// Test the heuristic auto-uppercasing
	if applyODataFormatting("apple", "Currency", "string", "2.0", true, compat) != "'APPLE'" {
		t.Errorf("Expected uppercase 'APPLE' for Currency param")
	}
	if applyODataFormatting("apple", "Description", "string", "2.0", true, compat) != "'apple'" {
		t.Errorf("Expected lowercase 'apple' for Description param")
	}
	if applyODataFormatting("apple", "SalesOrderID", "string", "2.0", true, compat) != "'APPLE'" {
		t.Errorf("Expected uppercase 'APPLE' for SalesOrderID param")
	}
	// Test quote escaping for OData v2
	if applyODataFormatting("O'Brien", "LastName", "string", "2.0", true, compat) != "'O''Brien'" {
		t.Errorf("Expected escaped quotes ''O''''Brien'' for LastName param, got %s", applyODataFormatting("O'Brien", "LastName", "string", "2.0", true, compat))
	}
}

type mockODataSourceWithResponse struct {
	baseURL  string
	metadata *odata.ODataMetadata
	response any
}

func (m *mockODataSourceWithResponse) SourceType() string             { return "odata" }
func (m *mockODataSourceWithResponse) ToConfig() sources.SourceConfig { return nil }
func (m *mockODataSourceWithResponse) HttpBaseURL() string            { return m.baseURL }
func (m *mockODataSourceWithResponse) RunODataRequest(req *http.Request, accessToken tools.AccessToken) (any, error) {
	return m.response, nil
}
func (m *mockODataSourceWithResponse) UseClientAuthorization() bool   { return false }
func (m *mockODataSourceWithResponse) Metadata() *odata.ODataMetadata { return m.metadata }
func (m *mockODataSourceWithResponse) Compatibility() odata.CompatibilityConfig {
	return odata.CompatibilityConfig{}
}

func (m *mockODataSourceWithResponse) GetAuthTokenHeaderName() string {
	return "Authorization"
}

type mockSourceProvider struct {
	sources map[string]sources.Source
}

func (m *mockSourceProvider) GetSource(name string) (sources.Source, bool) {
	s, ok := m.sources[name]
	return s, ok
}

func TestToolInvokePaginationV2(t *testing.T) {
	mockSrc := &mockODataSourceWithResponse{
		baseURL: "https://mock.OData/odata",
		response: map[string]interface{}{
			"d": map[string]interface{}{
				"results": []interface{}{},
				"__next":  "https://mock.OData/odata/A_SalesOrder?$skiptoken=123",
			},
		},
	}

	srcs := map[string]sources.Source{"my_mock_odata": mockSrc}
	sp := &mockSourceProvider{sources: srcs}

	cfg := Config{
		Source:    "my_mock_odata",
		EntitySet: "A_SalesOrder",
		Operation: "READ",
	}
	tool := Tool{
		BaseTool: tools.NewBaseTool(cfg, tools.NewReadOnlyAnnotations(), tools.Manifest{Description: cfg.Description, AuthRequired: cfg.AuthRequired}, cfg.QueryParams),
		Method:   "GET",
	}

	resp, err := tool.Invoke(context.Background(), sp, parameters.ParamValues{}, "")
	if err != nil {
		t.Fatalf("Invoke failed: %v", err)
	}

	respMap, ok := resp.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map response")
	}

	notice, ok := respMap["_NOTICE"].(string)
	if !ok || !strings.Contains(notice, "$skiptoken") {
		t.Errorf("Expected pagination notice, got: %v", respMap["_NOTICE"])
	}
}

func TestToolInvokePaginationV4(t *testing.T) {
	mockSrc := &mockODataSourceWithResponse{
		baseURL: "https://mock.OData/odata",
		response: map[string]interface{}{
			"value":           []interface{}{},
			"@odata.nextLink": "https://mock.OData/odata/A_SalesOrder?$skiptoken=123",
		},
	}

	srcs := map[string]sources.Source{"my_mock_odata": mockSrc}
	sp := &mockSourceProvider{sources: srcs}

	cfg := Config{
		Source:    "my_mock_odata",
		EntitySet: "A_SalesOrder",
		Operation: "READ",
	}
	tool := Tool{
		BaseTool: tools.NewBaseTool(cfg, tools.NewReadOnlyAnnotations(), tools.Manifest{Description: cfg.Description, AuthRequired: cfg.AuthRequired}, cfg.QueryParams),
		Method:   "GET",
	}

	resp, err := tool.Invoke(context.Background(), sp, parameters.ParamValues{}, "")
	if err != nil {
		t.Fatalf("Invoke failed: %v", err)
	}

	respMap, ok := resp.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map response")
	}

	notice, ok := respMap["_NOTICE"].(string)
	if !ok || !strings.Contains(notice, "$skiptoken") {
		t.Errorf("Expected pagination notice, got: %v", respMap["_NOTICE"])
	}
}

// Dummy tests to satisfy the interface completely
var _ sources.Source = &mockODataSource{}
