// Copyright 2025 Google LLC
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

package cloudgda_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	geminidataanalytics "cloud.google.com/go/geminidataanalytics/apiv1beta"
	"cloud.google.com/go/geminidataanalytics/apiv1beta/geminidataanalyticspb"
	"github.com/googleapis/genai-toolbox/internal/server/mcp/jsonrpc"
	source "github.com/googleapis/genai-toolbox/internal/sources/cloudgda"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/internal/tools/cloudgda"
	"github.com/googleapis/genai-toolbox/tests"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	cloudGdaToolType = "cloud-gemini-data-analytics-query"
)

type mockDataChatServer struct {
	geminidataanalyticspb.UnimplementedDataChatServiceServer
	t *testing.T
}

func (s *mockDataChatServer) QueryData(ctx context.Context, req *geminidataanalyticspb.QueryDataRequest) (*geminidataanalyticspb.QueryDataResponse, error) {
	if req.Prompt == "" {
		s.t.Errorf("missing prompt")
		return nil, fmt.Errorf("missing prompt")
	}

	return &geminidataanalyticspb.QueryDataResponse{
		GeneratedQuery:        "SELECT * FROM table;",
		NaturalLanguageAnswer: "Here is the answer.",
	}, nil
}

func getCloudGdaToolsConfig() map[string]any {
	return map[string]any{
		"sources": map[string]any{
			"my-gda-source": map[string]any{
				"type":      "cloud-gemini-data-analytics",
				"projectId": "test-project",
			},
		},
		"tools": map[string]any{
			"cloud-gda-query": map[string]any{
				"type":        cloudGdaToolType,
				"source":      "my-gda-source",
				"description": "Test GDA Tool",
				"location":    "us-central1",
				"context": map[string]any{
					"datasourceReferences": map[string]any{
						"spannerReference": map[string]any{
							"databaseReference": map[string]any{
								"projectId":  "test-project",
								"instanceId": "test-instance",
								"databaseId": "test-db",
								"engine":     "GOOGLE_SQL",
							},
						},
					},
				},
			},
		},
	}
}

func TestCloudGdaToolEndpoints(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	// Start a gRPC server
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	geminidataanalyticspb.RegisterDataChatServiceServer(s, &mockDataChatServer{t: t})
	go func() {
		if err := s.Serve(lis); err != nil {
			// This might happen on strict shutdown, log if unexpected
			t.Logf("server executed: %v", err)
		}
	}()
	defer s.Stop()

	// Configure toolbox to use the gRPC server
	endpoint := lis.Addr().String()

	// Override client creation
	origFunc := source.NewDataChatClient
	defer func() {
		source.NewDataChatClient = origFunc
	}()

	source.NewDataChatClient = func(ctx context.Context, opts ...option.ClientOption) (*geminidataanalytics.DataChatClient, error) {
		opts = append(opts,
			option.WithEndpoint(endpoint),
			option.WithoutAuthentication(),
			option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())))
		return origFunc(ctx, opts...)
	}

	var args []string
	toolsFile := getCloudGdaToolsConfig()
	cmd, cleanup, err := tests.StartCmd(ctx, toolsFile, args...)
	if err != nil {
		t.Fatalf("command initialization returned an error: %s", err)
	}
	defer cleanup()

	waitCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	out, err := testutils.WaitForString(waitCtx, regexp.MustCompile(`Server ready to serve`), cmd.Out)
	if err != nil {
		t.Logf("toolbox command logs: \n%s", out)
		t.Fatalf("toolbox didn't start successfully: %s", err)
	}

	toolName := "cloud-gda-query"

	// 1. RunToolGetTestByName
	expectedManifest := map[string]any{
		toolName: map[string]any{
			"description": "Test GDA Tool\n\n" + cloudgda.Guidance,
			"parameters": []any{
				map[string]any{
					"name":        "query",
					"type":        "string",
					"description": "A natural language formulation of a database query.",
					"required":    true,
					"authSources": []any{},
				},
			},
			"authRequired": []any{},
		},
	}
	tests.RunToolGetTestByName(t, toolName, expectedManifest)

	// 2. RunToolInvokeParametersTest
	params := []byte(`{"query": "test question"}`)
	tests.RunToolInvokeParametersTest(t, toolName, params, "\"generated_query\":\"SELECT * FROM table;\"")

	// 3. Manual MCP Tool Call Test
	// Initialize MCP session
	sessionId := tests.RunInitialize(t, "2024-11-05")

	// Construct MCP Request
	mcpReq := jsonrpc.JSONRPCRequest{
		Jsonrpc: "2.0",
		Id:      "test-mcp-call",
		Request: jsonrpc.Request{
			Method: "tools/call",
		},
		Params: map[string]any{
			"name": toolName,
			"arguments": map[string]any{
				"query": "test question",
			},
		},
	}
	reqBytes, _ := json.Marshal(mcpReq)

	headers := map[string]string{}
	if sessionId != "" {
		headers["Mcp-Session-Id"] = sessionId
	}

	// Send Request
	resp, respBody := tests.RunRequest(t, http.MethodPost, "http://127.0.0.1:5000/mcp", bytes.NewBuffer(reqBytes), headers)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("MCP request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	// Check Response
	respStr := string(respBody)
	if !strings.Contains(respStr, "SELECT * FROM table;") {
		t.Errorf("MCP response does not contain expected query result: %s", respStr)
	}
}
