// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
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
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/googleapis/mcp-toolbox/internal/testutils"
	"github.com/googleapis/mcp-toolbox/tests"
)

func TestMockODataMCPEndpoints(t *testing.T) {
	// Setup mock server simulating OData Gateway
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/odata/v2/SalesOrderService/$metadata" {
			w.Header().Set("Content-Type", "application/xml")
			_, _ = w.Write([]byte(`<edmx:Edmx Version="1.0" xmlns:edmx="http://schemas.microsoft.com/ado/2007/06/edmx"><edmx:DataServices m:DataServiceVersion="2.0" xmlns:m="http://schemas.microsoft.com/ado/2007/08/dataservices/metadata"><Schema Namespace="API_SALES_ORDER_SRV" xmlns="http://schemas.microsoft.com/ado/2008/09/edm"><EntityContainer Name="API_SALES_ORDER_SRV_Entities" m:IsDefaultEntityContainer="true"><EntitySet Name="A_SalesOrder" EntityType="API_SALES_ORDER_SRV.A_SalesOrderType"/></EntityContainer><EntityType Name="A_SalesOrderType"><Key><PropertyRef Name="SalesOrder"/></Key><Property Name="SalesOrder" Type="Edm.String" Nullable="false" MaxLength="10"/><Property Name="SalesOrderType" Type="Edm.String" Nullable="false" MaxLength="4"/><Property Name="SoldToParty" Type="Edm.String" Nullable="false" MaxLength="10"/></EntityType></Schema></edmx:DataServices></edmx:Edmx>`))
			return
		}
		if r.URL.Path == "/odata/v2/SampleFlightService/$metadata" {
			w.Header().Set("Content-Type", "application/xml")
			_, _ = w.Write([]byte(`<edmx:Edmx Version="1.0" xmlns:edmx="http://schemas.microsoft.com/ado/2007/06/edmx"><edmx:DataServices m:DataServiceVersion="2.0" xmlns:m="http://schemas.microsoft.com/ado/2007/08/dataservices/metadata"><Schema Namespace="RMTSAMPLEFLIGHT" xmlns="http://schemas.microsoft.com/ado/2008/09/edm"><EntityContainer Name="RMTSAMPLEFLIGHT_Entities" m:IsDefaultEntityContainer="true"><FunctionImport Name="CheckFlightAvailability" ReturnType="RMTSAMPLEFLIGHT.FlightAvailability" HttpMethod="GET"><Parameter Name="airlineid" Type="Edm.String" Mode="In"/><Parameter Name="connectionid" Type="Edm.String" Mode="In"/><Parameter Name="flightdate" Type="Edm.DateTime" Mode="In"/></FunctionImport></EntityContainer></Schema></edmx:DataServices></edmx:Edmx>`))
			return
		}
		if r.URL.Path == "/odata/v2/SalesOrderService/A_SalesOrder" {
			if r.Header.Get("X-OData-Token") == "" {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error":"unauthorized: missing X-OData-Token"}`))
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"d":{"results":[{"SalesOrder":"1","SalesOrderType":"OR","SoldToParty":"100001"}]}}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	toolsFile := map[string]any{
		"sources": map[string]any{
			"mock-odata-source": map[string]any{
				"type":           "odata",
				"baseUrl":        ts.URL + "/odata/v2/SalesOrderService",
				"authStrategy":   "gateway",
				"useClientOauth": "X-OData-Token",
			},
			"mock-odata-flight-source": map[string]any{
				"type":           "odata",
				"baseUrl":        ts.URL + "/odata/v2/SampleFlightService",
				"authStrategy":   "gateway",
				"useClientOauth": "X-OData-Token",
			},
		},
		"tools": map[string]any{
			"read-sales-order": map[string]any{
				"type":        "odata",
				"source":      "mock-odata-source",
				"entitySet":   "A_SalesOrder",
				"operation":   "READ",
				"description": "Read Sales Orders",
			},
			"create-sales-order": map[string]any{
				"type":        "odata",
				"source":      "mock-odata-source",
				"entitySet":   "A_SalesOrder",
				"operation":   "CREATE",
				"description": "Create Sales Order",
				"bodyParams": []map[string]any{
					{"name": "SalesOrderType", "type": "string", "description": "Sales Order Type"},
					{"name": "SoldToParty", "type": "string", "description": "Sold To Party"},
				},
			},
			"check-flight-availability": map[string]any{
				"type":        "odata",
				"source":      "mock-odata-flight-source",
				"entitySet":   "CheckFlightAvailability",
				"operation":   "FUNCTION_IMPORT",
				"description": "Check Flight Availability",
				"queryParams": []map[string]any{
					{"name": "airlineid", "type": "string", "description": "Airline"},
					{"name": "connectionid", "type": "string", "description": "Connection"},
					{"name": "flightdate", "type": "string", "description": "Date"},
				},
			},
		},
	}

	args := []string{"--enable-api"}
	cmd, cleanup, err := tests.StartCmd(ctx, toolsFile, args...)
	if err != nil {
		t.Fatalf("command initialization returned an error: %v", err)
	}
	defer cleanup()

	waitCtx, cancelWait := context.WithTimeout(ctx, 20*time.Second)
	defer cancelWait()
	out, err := testutils.WaitForString(waitCtx, regexp.MustCompile(`Server ready to serve`), cmd.Out)
	if err != nil {
		t.Logf("toolbox command logs: \n%s", out)
		t.Fatalf("toolbox didn't start successfully: %v", err)
	}

	sessionID := tests.RunInitialize(t, "2024-11-05")

	// 1. MCP Tools List
	t.Run("MCP Tools List", func(t *testing.T) {
		expectedRegistry := []tests.MCPToolManifest{
			{
				Name:        "read-sales-order",
				Description: "Read Sales Orders",
				InputSchema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"filter":    map[string]any{"type": "string", "description": "OData $filter string. Available properties: SalesOrder (Edm.String), SalesOrderType (Edm.String), SoldToParty (Edm.String)"},
						"select":    map[string]any{"type": "string", "description": "OData $select string. Available properties: SalesOrder (Edm.String), SalesOrderType (Edm.String), SoldToParty (Edm.String)"},
						"top":       map[string]any{"type": "integer", "description": "OData $top integer limit."},
						"skip":      map[string]any{"type": "integer", "description": "OData $skip integer offset."},
						"skiptoken": map[string]any{"type": "string", "description": "OData $skiptoken string for server-side pagination."},
					},
					"required": []any{},
				},
			},
			{
				Name:        "create-sales-order",
				Description: "Create Sales Order",
				InputSchema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"SalesOrderType": map[string]any{"type": "string", "description": "Sales Order Type"},
						"SoldToParty":    map[string]any{"type": "string", "description": "Sold To Party"},
					},
					"required": []any{"SalesOrderType", "SoldToParty"},
				},
			},
			{
				Name:        "check-flight-availability",
				Description: "Check Flight Availability",
				InputSchema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"airlineid":    map[string]any{"type": "string", "description": "Airline"},
						"connectionid": map[string]any{"type": "string", "description": "Connection"},
						"flightdate":   map[string]any{"type": "string", "description": "Date"},
					},
					"required": []any{"airlineid", "connectionid", "flightdate"},
				},
			},
		}
		tests.RunMCPToolsListMethod(t, expectedRegistry)
	})

	// 2. MCP Tool Call Success
	t.Run("MCP Tool Call Success", func(t *testing.T) {
		headers := map[string]string{
			"X-OData-Token":     "Bearer mock-oauth-token",
			"Mcp-Session-Id": sessionID,
		}

		status, resp, err := tests.InvokeMCPTool(t, "read-sales-order", map[string]any{}, headers)
		if err != nil {
			t.Fatalf("native error executing read-sales-order: %s", err)
		}
		if status != http.StatusOK {
			t.Fatalf("expected status 200, got %d", status)
		}
		if resp.Error != nil {
			t.Fatalf("mcp call returned error: %v", resp.Error)
		}
		if len(resp.Result.Content) == 0 {
			t.Fatalf("mcp returned empty content field")
		}
		gotResult := resp.Result.Content[0].Text
		if !strings.Contains(gotResult, `"SalesOrder":"1"`) {
			t.Fatalf(`expected %q to contain "SalesOrder":"1"`, gotResult)
		}
	})

	// 3. MCP Tool Call Unauthorized
	t.Run("MCP Tool Call Unauthorized", func(t *testing.T) {
		headers := map[string]string{
			"Mcp-Session-Id": sessionID,
		}
		statusErr, respErr, _ := tests.InvokeMCPTool(t, "read-sales-order", map[string]any{}, headers)
		if statusErr == http.StatusUnauthorized {
			// Successfully blocked by HTTP middleware / authorization check
			return
		}
		if statusErr != http.StatusOK {
			t.Fatalf("expected status 200 or 401, got %d", statusErr)
		}
		tests.AssertMCPError(t, respErr, "missing access token")
	})
}
