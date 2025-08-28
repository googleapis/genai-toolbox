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

package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"

	mcp "github.com/googleapis/genai-toolbox/internal/server/mcp/jsonrpc"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/tests"
)

var (
	ElasticsearchSourceKind = "elasticsearch"
	EsAddress               = os.Getenv("ELASTICSEARCH_URL")
	EsApiKey                = os.Getenv("ELASTICSEARCH_API_KEY")
)

func getElasticsearchVars(t *testing.T) map[string]any {
	if EsAddress == "" {
		t.Fatal("'ELASTICSEARCH_URL' not set")
	}
	return map[string]any{
		"kind":      ElasticsearchSourceKind,
		"addresses": []string{EsAddress},
		"apikey":    EsApiKey,
	}
}

func TestElasticsearchToolEndpoints(t *testing.T) {
	sourceConfig := getElasticsearchVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var args []string

	tools := map[string]any{
		"sources": map[string]any{
			"my-elasticsearch-instance": sourceConfig,
		},
		"tools": map[string]any{
			"esql-tool": map[string]any{
				"kind":        "elasticsearch-esql",
				"source":      "my-elasticsearch-instance",
				"description": "Elasticsearch ES|QL tool",
				"query": `FROM test-index
                         | KEEP first_name, last_name`,
			},
			"esql-with-params-tool": map[string]any{
				"kind":        "elasticsearch-esql",
				"source":      "my-elasticsearch-instance",
				"description": "Elasticsearch ES|QL tool with parameters",
				"query":       `FROM test-index | LIMIT ?limit`,
				"parameters": []any{
					map[string]any{
						"name":        "limit",
						"type":        "integer",
						"description": "limit the number of results",
						"required":    true,
					},
				},
			},
		},
	}

	cmd, cleanup, err := tests.StartCmd(ctx, tools, args...)
	if err != nil {
		t.Fatalf("failed to start cmd: %v", err)
	}
	defer cleanup()

	waitCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	out, err := testutils.WaitForString(waitCtx, regexp.MustCompile(`Server ready to serve`), cmd.Out)
	if err != nil {
		t.Logf("toolbox command logs: \n%s", out)
		t.Fatalf("toolbox didn't start successfully: %s", err)
	}

	esClient, err := elasticsearch.NewBaseClient(elasticsearch.Config{
		Addresses: []string{EsAddress},
		APIKey:    EsApiKey,
	})
	if err != nil {
		t.Fatalf("error creating the Elasticsearch client: %s", err)
	}

	// Delete index if already exists
	defer func() {
		_, err = esapi.IndicesDeleteRequest{
			Index: []string{"test-index"},
		}.Do(ctx, esClient)
		if err != nil {
			t.Fatalf("error deleting index: %s", err)
		}
	}()

	// Index sample documents
	sampleDocs := []string{
		`{"first_name": "John", "last_name": "Doe"}`,
		`{"first_name": "Jane", "last_name": "Smith"}`,
	}
	for _, doc := range sampleDocs {
		_, err := esapi.IndexRequest{
			Index:   "test-index",
			Body:    strings.NewReader(doc),
			Refresh: "true",
		}.Do(ctx, esClient)
		if err != nil {
			t.Fatalf("error indexing document: %s", err)
		}
	}

	// Test tool get endpoint
	tcs := []struct {
		name string
		api  string
		want map[string]any
	}{
		{
			name: "Get Elasticsearch ES|QL tool",
			api:  "http://127.0.0.1:5000/api/tool/esql-tool/",
			want: map[string]any{
				"esql-tool": map[string]any{
					"description":  "Elasticsearch ES|QL tool",
					"parameters":   []any{},
					"authRequired": []any{},
				},
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := http.Get(tc.api)
			if err != nil {
				t.Fatalf("error when sending a request: %s", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != 200 {
				t.Fatalf("response status code is not 200")
			}

			var body map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&body)
			if err != nil {
				t.Fatalf("error parsing response body")
			}

			got, ok := body["tools"]
			if !ok {
				t.Fatalf("unable to find tools in response body")
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}

	// Test tool invoke endpoint
	invokeTcs := []struct {
		name        string
		api         string
		requestBody io.Reader
		want        string
	}{
		{
			name:        "Invoke Elasticsearch ES|QL tool",
			api:         "http://127.0.0.1:5000/api/tool/esql-tool/invoke",
			requestBody: strings.NewReader("{}"),
			want:        `[["John","Doe"],["Jane","Smith"]]`,
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := http.Post(tc.api, "application/json", tc.requestBody)
			if err != nil {
				t.Fatalf("error when sending a request: %s", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				bodyBytes, _ := io.ReadAll(resp.Body)
				t.Fatalf("response status code is not 200, got %d: %s", resp.StatusCode, string(bodyBytes))
			}

			var body map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&body)
			if err != nil {
				t.Fatalf("error parsing response body")
			}
			got, ok := body["result"].(string)

			if !ok {
				t.Fatalf("unable to find result in response body")
			}

			if !strings.Contains(got, tc.want) {
				t.Fatalf("Expected substring not found:\ngot:  %q\nwant: %q (to be contained within got)", got, tc.want)
			}
		})
	}

	// Test tool invoke endpoint
	invokeMcpTcs := []struct {
		name          string
		api           string
		requestBody   mcp.JSONRPCRequest
		requestHeader map[string]string
		want          string
	}{
		{
			name:          "MCP Invoke ES|QL tool",
			api:           "http://127.0.0.1:5000/mcp",
			requestHeader: map[string]string{},
			requestBody: mcp.JSONRPCRequest{
				Jsonrpc: "2.0",
				Id:      "esql-tool",
				Request: mcp.Request{
					Method: "tools/call",
				},
				Params: map[string]any{
					"name":      "esql-tool",
					"arguments": map[string]any{},
				},
			},
			want: "[[\\\"John\\\",\\\"Doe\\\"],[\\\"Jane\\\",\\\"Smith\\\"]]",
		},
	}
	for _, tc := range invokeMcpTcs {
		t.Run(tc.name, func(t *testing.T) {
			reqMarshal, err := json.Marshal(tc.requestBody)
			if err != nil {
				t.Fatalf("unexpected error during marshaling of request body")
			}
			// Send Tool invocation request
			req, err := http.NewRequest(http.MethodPost, tc.api, bytes.NewBuffer(reqMarshal))
			if err != nil {
				t.Fatalf("unable to create request: %s", err)
			}
			req.Header.Add("Content-type", "application/json")
			for k, v := range tc.requestHeader {
				req.Header.Add(k, v)
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("unable to send request: %s", err)
			}
			respBody, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("unable to read request body: %s", err)
			}
			defer resp.Body.Close()
			got := string(bytes.TrimSpace(respBody))

			if !strings.Contains(got, tc.want) {
				t.Fatalf("Expected substring not found:\ngot:  %q\nwant: %q (to be contained within got)", got, tc.want)
			}
		})
	}

	// Test tool invoke endpoint
	invokeWithTemplateTcs := []struct {
		name          string
		insert        bool
		api           string
		requestHeader map[string]string
		requestBody   io.Reader
		want          string
		isErr         bool
	}{
		{
			name:          "Invoke ES|QL tool with parameters (sort ascending)",
			api:           "http://127.0.0.1:5000/api/tool/esql-with-params-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte("{\"limit\": 1}")),
			want:          "[[\"John\",\"John\",\"Doe\",\"Doe\"]]",
			isErr:         false,
		},
	}
	for _, tc := range invokeWithTemplateTcs {
		t.Run(tc.name, func(t *testing.T) {
			// Send Tool invocation request
			req, err := http.NewRequest(http.MethodPost, tc.api, tc.requestBody)
			if err != nil {
				t.Fatalf("unable to create request: %s", err)
			}
			req.Header.Add("Content-type", "application/json")
			for k, v := range tc.requestHeader {
				req.Header.Add(k, v)
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("unable to send request: %s", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				if tc.isErr {
					return
				}
				bodyBytes, _ := io.ReadAll(resp.Body)
				t.Fatalf("response status code is not 200, got %d: %s", resp.StatusCode, string(bodyBytes))
			}

			// Check response body
			var body map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&body)
			if err != nil {
				t.Fatalf("error parsing response body")
			}

			got, ok := body["result"].(string)
			if !ok {
				t.Fatalf("unable to find result in response body")
			}

			if !strings.Contains(got, tc.want) {
				t.Fatalf("Expected substring not found:\ngot:  %q\nwant: %q (to be contained within got)", got, tc.want)
			}
		})
	}
}
