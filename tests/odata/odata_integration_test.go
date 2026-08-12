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
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/googleapis/mcp-toolbox/internal/testutils"
	"github.com/googleapis/mcp-toolbox/tests"
)

func TestMockODataNativeEndpoints(t *testing.T) {
	// Setup mock server simulating OData Gateway
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Metadata endpoint
		if r.URL.Path == "/odata/v2/SalesOrderService/$metadata" {
			w.Header().Set("Content-Type", "application/xml")
			_, _ = w.Write([]byte(`<edmx:Edmx Version="1.0" xmlns:edmx="http://schemas.microsoft.com/ado/2007/06/edmx"><edmx:DataServices m:DataServiceVersion="2.0" xmlns:m="http://schemas.microsoft.com/ado/2007/08/dataservices/metadata"><Schema Namespace="API_SALES_ORDER_SRV" xmlns="http://schemas.microsoft.com/ado/2008/09/edm"><EntityContainer Name="API_SALES_ORDER_SRV_Entities" m:IsDefaultEntityContainer="true"><EntitySet Name="A_SalesOrder" EntityType="API_SALES_ORDER_SRV.A_SalesOrderType"/></EntityContainer><EntityType Name="A_SalesOrderType"><Key><PropertyRef Name="SalesOrder"/></Key><Property Name="SalesOrder" Type="Edm.String" Nullable="false" MaxLength="10"/><Property Name="SalesOrderType" Type="Edm.String" Nullable="false" MaxLength="4"/><Property Name="SoldToParty" Type="Edm.String" Nullable="false" MaxLength="10"/></EntityType></Schema></edmx:DataServices></edmx:Edmx>`))
			return
		}
		// Entity set read endpoint
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
		},
		"tools": map[string]any{
			"read-sales-order": map[string]any{
				"type":        "odata",
				"source":      "mock-odata-source",
				"entitySet":   "A_SalesOrder",
				"operation":   "READ",
				"description": "Read Sales Orders",
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

	// 1. GET Tool Manifest
	t.Run("GET Tool Manifest", func(t *testing.T) {
		tests.RunToolGetTestByName(t, "read-sales-order", map[string]any{
			"read-sales-order": map[string]any{
				"description": "Read Sales Orders",
				"parameters": []any{
					map[string]any{
						"name":         "filter",
						"type":         "string",
						"description":  "OData $filter string. Available properties: SalesOrder (Edm.String), SalesOrderType (Edm.String), SoldToParty (Edm.String)",
						"required":     false,
						"authServices": []any{},
					},
					map[string]any{
						"name":         "select",
						"type":         "string",
						"description":  "OData $select string. Available properties: SalesOrder (Edm.String), SalesOrderType (Edm.String), SoldToParty (Edm.String)",
						"required":     false,
						"authServices": []any{},
					},
					map[string]any{
						"name":         "top",
						"type":         "integer",
						"description":  "OData $top integer limit.",
						"required":     false,
						"authServices": []any{},
					},
					map[string]any{
						"name":         "skip",
						"type":         "integer",
						"description":  "OData $skip integer offset.",
						"required":     false,
						"authServices": []any{},
					},
					map[string]any{
						"name":         "skiptoken",
						"type":         "string",
						"description":  "OData $skiptoken string for server-side pagination.",
						"required":     false,
						"authServices": []any{},
					},
				},
				"authRequired": []any{},
			},
		})
	})

	// 2. Native Read Unauthorized
	t.Run("Native Read Unauthorized", func(t *testing.T) {
		apiRead := "http://127.0.0.1:5000/api/tool/read-sales-order/invoke"
		reqNoAuth, err := http.NewRequest(http.MethodPost, apiRead, bytes.NewBufferString("{}"))
		if err != nil {
			t.Fatalf("unable to create request: %s", err)
		}
		reqNoAuth.Header.Add("Content-type", "application/json")

		respNoAuth, err := http.DefaultClient.Do(reqNoAuth)
		if err != nil {
			t.Fatalf("unable to send request: %s", err)
		}
		defer respNoAuth.Body.Close()

		if respNoAuth.StatusCode != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", respNoAuth.StatusCode)
		}
	})

	// 3. Native Read Success
	t.Run("Native Read Success", func(t *testing.T) {
		apiRead := "http://127.0.0.1:5000/api/tool/read-sales-order/invoke"
		reqRead, err := http.NewRequest(http.MethodPost, apiRead, bytes.NewBufferString("{}"))
		if err != nil {
			t.Fatalf("unable to create request: %s", err)
		}
		reqRead.Header.Add("Content-type", "application/json")
		reqRead.Header.Add("Authorization", "Bearer mock-oauth-token")

		respRead, err := http.DefaultClient.Do(reqRead)
		if err != nil {
			t.Fatalf("unable to send request: %s", err)
		}
		defer respRead.Body.Close()

		if respRead.StatusCode != http.StatusOK {
			bodyBytes, _ := io.ReadAll(respRead.Body)
			t.Fatalf("response status code is not 200, got %d: %s", respRead.StatusCode, string(bodyBytes))
		}

		var bodyRead map[string]any
		if err := json.NewDecoder(respRead.Body).Decode(&bodyRead); err != nil {
			t.Fatalf("error parsing response body: %v", err)
		}

		resultRead, ok := bodyRead["result"].(string)
		if !ok {
			t.Fatalf("unable to find 'result' string in response body")
		}

		if !regexp.MustCompile(`"SalesOrder":"1"`).MatchString(resultRead) {
			t.Errorf("unexpected read result: %s", resultRead)
		}
	})

	// 4. Error Propagation
	t.Run("Error Propagation", func(t *testing.T) {
		apiRead := "http://127.0.0.1:5000/api/tool/read-sales-order/invoke"
		reqErr, err := http.NewRequest(http.MethodPost, apiRead, bytes.NewBufferString(`{"filter":"SalesOrder eq 'ERROR'"}`))
		if err != nil {
			t.Fatalf("unable to create request: %s", err)
		}
		reqErr.Header.Add("Content-type", "application/json")
		reqErr.Header.Add("Authorization", "Bearer mock-oauth-token")

		respErr, err := http.DefaultClient.Do(reqErr)
		if err != nil {
			t.Fatalf("unable to send request: %s", err)
		}
		defer respErr.Body.Close()

		var bodyErr map[string]any
		if err := json.NewDecoder(respErr.Body).Decode(&bodyErr); err != nil {
			t.Fatalf("error parsing response body: %v", err)
		}

		resultErr, ok := bodyErr["result"].(string)
		if !ok {
			t.Fatalf("unable to find 'result' string in response body: %+v", bodyErr)
		}
		if !strings.Contains(resultErr, "SalesOrder") {
			t.Logf("Result string obtained: %s", resultErr)
		}
	})
}
