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
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/googleapis/mcp-toolbox/internal/testutils"
	"github.com/googleapis/mcp-toolbox/tests"
)

var (
	SAPTestBaseURL    = os.Getenv("SAP_TEST_BASE_URL")
	SAPTestClientCert = os.Getenv("SAP_TEST_CLIENT_CERT")
	SAPTestClientKey  = os.Getenv("SAP_TEST_CLIENT_KEY")
	SAPTestUsername   = os.Getenv("SAP_TEST_USERNAME")
	SAPTestPassword   = os.Getenv("SAP_TEST_PASSWORD")
)

func TestMockSAPOData(t *testing.T) {
	// Setup mock server simulating SAP Gateway
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Mock response for metadata
		if r.URL.Path == "/sap/opu/odata/sap/API_SALES_ORDER_SRV/$metadata" {
			w.Header().Set("Content-Type", "application/xml")
			w.Write([]byte(`<edmx:Edmx Version="1.0" xmlns:edmx="http://schemas.microsoft.com/ado/2007/06/edmx"><edmx:DataServices m:DataServiceVersion="2.0" xmlns:m="http://schemas.microsoft.com/ado/2007/08/dataservices/metadata"><Schema Namespace="API_SALES_ORDER_SRV" xmlns="http://schemas.microsoft.com/ado/2008/09/edm"><EntityContainer Name="API_SALES_ORDER_SRV_Entities" m:IsDefaultEntityContainer="true"><EntitySet Name="A_SalesOrder" EntityType="API_SALES_ORDER_SRV.A_SalesOrderType"/></EntityContainer><EntityType Name="A_SalesOrderType"><Key><PropertyRef Name="SalesOrder"/></Key><Property Name="SalesOrder" Type="Edm.String" Nullable="false" MaxLength="10"/><Property Name="SalesOrderType" Type="Edm.String" Nullable="false" MaxLength="4"/><Property Name="SoldToParty" Type="Edm.String" Nullable="false" MaxLength="10"/></EntityType></Schema></edmx:DataServices></edmx:Edmx>`))
			return
		}
		// Mock response for CSRF pre-flight HEAD requests on the BaseURL
		if r.URL.Path == "/sap/opu/odata/sap/API_SALES_ORDER_SRV" {
			// Ensure bearer token was passed
			if r.Header.Get("X-SAP-Auth") == "" {
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":"unauthorized: missing X-SAP-Auth"}`))
				return
			}
			if r.Method == http.MethodHead && r.Header.Get("X-CSRF-Token") == "Fetch" {
				w.Header().Set("X-CSRF-Token", "mock-csrf-token-12345")
				w.Header().Set("Set-Cookie", "sap-session-cookie=mock-cookie-value")
				w.WriteHeader(http.StatusOK)
				return
			}
		}
		// Mock response for entity set read or create
		if r.URL.Path == "/sap/opu/odata/sap/API_SALES_ORDER_SRV/A_SalesOrder" {
			// Ensure bearer token was passed
			if r.Header.Get("X-SAP-Auth") == "" {
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":"unauthorized: missing X-SAP-Auth"}`))
				return
			}
			w.Header().Set("Content-Type", "application/json")
			if r.Method == http.MethodPost {
				// Ensure mock verified the CSRF token
				if r.Header.Get("X-CSRF-Token") != "mock-csrf-token-12345" {
					w.WriteHeader(http.StatusForbidden)
					w.Write([]byte(`{"error":"invalid csrf token"}`))
					return
				}
				w.WriteHeader(http.StatusCreated)
				w.Write([]byte(`{"d":{"SalesOrder":"24","SalesOrderType":"OR1","SoldToParty":"1"}}`))
				return
			}
			w.Write([]byte(`{"d":{"results":[{"SalesOrder":"1"}]}}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Create config for the toolbox
	toolsFile := map[string]any{
		"sources": map[string]any{
			"mock-sap-source": map[string]any{
				"type":           "odata",
				"baseUrl":        ts.URL + "/sap/opu/odata/sap/API_SALES_ORDER_SRV",
				"authStrategy":   "sap-gateway",
				"useClientOauth": "X-SAP-Auth",
			},
		},
		"tools": map[string]any{
			"read-sales-order": map[string]any{
				"type":        "odata",
				"source":      "mock-sap-source",
				"entitySet":   "A_SalesOrder",
				"operation":   "READ",
				"description": "Read Sales Orders",
			},
			"create-sales-order": map[string]any{
				"type":        "odata",
				"source":      "mock-sap-source",
				"entitySet":   "A_SalesOrder",
				"operation":   "CREATE",
				"description": "Create Sales Order",
				"bodyParams": []map[string]any{
					{"name": "SalesOrderType", "type": "string", "description": "Sales Order Type"},
					{"name": "SoldToParty", "type": "string", "description": "Sold To Party"},
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

	// 1. Invoke without authorization header -> Expect 401
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

	// 2. Invoke with authorization header -> Expect 200
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

	// 3. Invoke the create tool via HTTP API (F002)
	apiCreate := "http://127.0.0.1:5000/api/tool/create-sales-order/invoke"
	payloadCreate := `{"SalesOrderType": "OR1", "SoldToParty": "1"}`
	reqCreate, err := http.NewRequest(http.MethodPost, apiCreate, bytes.NewBufferString(payloadCreate))
	if err != nil {
		t.Fatalf("unable to create request: %s", err)
	}
	reqCreate.Header.Add("Content-type", "application/json")
	reqCreate.Header.Add("Authorization", "Bearer mock-oauth-token")

	respCreate, err := http.DefaultClient.Do(reqCreate)
	if err != nil {
		t.Fatalf("unable to send request: %s", err)
	}
	defer respCreate.Body.Close()

	if respCreate.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(respCreate.Body)
		t.Fatalf("response status code is not 200, got %d: %s", respCreate.StatusCode, string(bodyBytes))
	}

	var bodyCreate map[string]any
	if err := json.NewDecoder(respCreate.Body).Decode(&bodyCreate); err != nil {
		t.Fatalf("error parsing response body: %v", err)
	}

	resultCreate, ok := bodyCreate["result"].(string)
	if !ok {
		t.Fatalf("unable to find 'result' string in response body")
	}

	if !regexp.MustCompile(`"SalesOrder":"24"`).MatchString(resultCreate) {
		t.Errorf("unexpected create result: %s", resultCreate)
	}
}

func TestLiveSAPOData(t *testing.T) {
	if SAPTestBaseURL == "" {
		t.Skip("Skipping live test; SAP_TEST_BASE_URL not set")
	}

	hasX509 := SAPTestClientCert != "" && SAPTestClientKey != ""
	hasBasic := SAPTestUsername != "" && SAPTestPassword != ""

	if !hasX509 && !hasBasic {
		t.Skip("Skipping live test; neither X509 nor Basic auth environment variables are fully set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	sourceConfig := map[string]any{
		"type":         "odata",
		"baseUrl":      SAPTestBaseURL,
		"authStrategy": "sap-gateway",
		"disableSslVerification": true,
	}

	if hasX509 {
		sourceConfig["auth"] = map[string]any{
			"type":       "x509",
			"clientCert": SAPTestClientCert,
			"clientKey":  SAPTestClientKey,
		}
		t.Log("Running live test with X509 authentication")
	} else if hasBasic {
		sourceConfig["auth"] = map[string]any{
			"type":     "basic",
			"username": SAPTestUsername,
			"password": SAPTestPassword,
		}
		t.Log("Running live test with Basic authentication")
	}

	toolsFile := map[string]any{
		"sources": map[string]any{
			"live-sap-source": sourceConfig,
		},
		"tools": map[string]any{
			"read-sales-order-live": map[string]any{
				"type":        "odata",
				"source":      "live-sap-source",
				"entitySet":   "A_SalesOrder",
				"operation":   "READ",
				"description": "Read Sales Orders Live",
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

	// Invoke the tool via HTTP API
	api := "http://127.0.0.1:5000/api/tool/read-sales-order-live/invoke"
	req, err := http.NewRequest(http.MethodPost, api, bytes.NewBufferString("{}"))
	if err != nil {
		t.Fatalf("unable to create request: %s", err)
	}
	req.Header.Add("Content-type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("unable to send request: %s", err)
	}
	defer resp.Body.Close()

	// We expect 200 OK if the credentials are correct and the system is accessible.
	// If it's a sandbox, it might return 200 with data or an empty list.
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Fatalf("response status code is not 200, got %d: %s", resp.StatusCode, string(bodyBytes))
	}

	t.Log("Live test completed successfully")
}
