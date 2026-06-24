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

package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/googleapis/mcp-toolbox/internal/resources"
	"github.com/googleapis/mcp-toolbox/internal/server/mcp/jsonrpc"
	"github.com/googleapis/mcp-toolbox/internal/testutils"
)

func withResources(res map[string]resources.Resource, templates map[string]resources.Resource) func(*Server) {
	return func(s *Server) {
		s.PrimitiveMgr.SetPrimitives(
			s.PrimitiveMgr.GetSourcesMap(),
			s.PrimitiveMgr.GetAuthServiceMap(),
			s.PrimitiveMgr.GetEmbeddingModelMap(),
			s.PrimitiveMgr.GetToolsMap(),
			s.PrimitiveMgr.GetToolsetsMap(),
			s.PrimitiveMgr.GetPromptsMap(),
			s.PrimitiveMgr.GetPromptsetsMap(),
			res,
			templates,
		)
	}
}

func TestResources_Integration(t *testing.T) {
	ctx := context.Background()
	tempDir, err := os.MkdirTemp("", "mcp-resources-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Resolve physical path for tempDir to avoid symlink issues
	resolvedTempDir, err := filepath.EvalSymlinks(tempDir)
	if err != nil {
		t.Fatalf("failed to resolve symlinks for tempDir: %v", err)
	}

	// Set up a test file inside the sandbox
	testFilePath := filepath.Join(resolvedTempDir, "sub", "test.txt")
	if err := os.MkdirAll(filepath.Dir(testFilePath), 0755); err != nil {
		t.Fatalf("failed to create sub dir: %v", err)
	}
	if err := os.WriteFile(testFilePath, []byte("hello from file"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Set up a file OUTSIDE the sandbox (but inside our temp directory for safety/cleanup)
	outsideFilePath := filepath.Join(resolvedTempDir, "outside.txt")
	if err := os.WriteFile(outsideFilePath, []byte("outside sandbox content"), 0644); err != nil {
		t.Fatalf("failed to write outside file: %v", err)
	}

	// Create resources configs via YAML decoding (matches production boot)
	textYaml := `
name: static-doc
type: text
text: "hello from static text"
uri: "info://static-doc"
`
	textCfg, err := resources.DecodeConfig(ctx, "text", "static-doc", yaml.NewDecoder(strings.NewReader(textYaml)))
	if err != nil {
		t.Fatalf("failed to decode text config: %v", err)
	}
	textRes, err := textCfg.Initialize(ctx, "", false)
	if err != nil {
		t.Fatalf("failed to initialize text resource: %v", err)
	}

	sandboxSubDir := filepath.Join(resolvedTempDir, "sub")
	fileYaml := `
name: sandbox-template
type: file
uri: "file://` + sandboxSubDir + `/{path}"
allowedPaths:
  - "` + sandboxSubDir + `"
`
	fileCfg, err := resources.DecodeConfig(ctx, "file", "sandbox-template", yaml.NewDecoder(strings.NewReader(fileYaml)))
	if err != nil {
		t.Fatalf("failed to decode file template config: %v", err)
	}
	fileTemplate, err := fileCfg.Initialize(ctx, sandboxSubDir, false)
	if err != nil {
		t.Fatalf("failed to initialize file template: %v", err)
	}

	resMap := map[string]resources.Resource{"static-doc": textRes}
	tempMap := map[string]resources.Resource{"sandbox-template": fileTemplate}

	mockTools := []testutils.MockTool{testutils.MockTool1, testutils.MockTool2}
	mockPrompts := []testutils.MockPrompt{testutils.MockPrompt1}
	toolsMap, toolsets, promptsMap, promptsets := testutils.SetUpResources(t, mockTools, mockPrompts)

	// Start test server using our custom resources injector option (setUpServer is now visible!)
	r, shutdown := setUpServer(t, "mcp", toolsMap, toolsets, promptsMap, promptsets, withResources(resMap, tempMap))
	defer shutdown()

	ts := httptest.NewServer(r)
	defer ts.Close()

	t.Run("resources/list returns resources and templates", func(t *testing.T) {
		reqBody := jsonrpc.JSONRPCRequest{
			Jsonrpc: jsonrpc.JSONRPC_VERSION,
			Id:      1,
			Request: jsonrpc.Request{
				Method: "resources/list",
			},
		}
		bodyBytes, _ := json.Marshal(reqBody)

		res, err := http.Post(ts.URL+"/", "application/json", bytes.NewReader(bodyBytes))
		if err != nil {
			t.Fatalf("POST request failed: %v", err)
		}
		defer res.Body.Close()

		respBytes, _ := io.ReadAll(res.Body)
		if res.StatusCode != http.StatusOK {
			t.Fatalf("expected HTTP 200, got %d. Body: %s", res.StatusCode, respBytes)
		}

		var rpcResp struct {
			Result struct {
				Resources []struct {
					URI  string `json:"uri"`
					Name string `json:"name"`
				} `json:"resources"`
				ResourceTemplates []struct {
					URITemplate string `json:"uriTemplate"`
					Name        string `json:"name"`
				} `json:"resourceTemplates"`
			} `json:"result"`
		}
		if err := json.Unmarshal(respBytes, &rpcResp); err != nil {
			t.Fatalf("failed to unmarshal response: %v. Body: %s", err, respBytes)
		}

		if len(rpcResp.Result.Resources) != 1 || rpcResp.Result.Resources[0].Name != "static-doc" {
			t.Errorf("expected 1 static resource named 'static-doc', got: %v", rpcResp.Result.Resources)
		}
		if len(rpcResp.Result.ResourceTemplates) != 1 || rpcResp.Result.ResourceTemplates[0].Name != "sandbox-template" {
			t.Errorf("expected 1 template named 'sandbox-template', got: %v", rpcResp.Result.ResourceTemplates)
		}
	})

	t.Run("resources/read static text resource", func(t *testing.T) {
		reqBody := jsonrpc.JSONRPCRequest{
			Jsonrpc: jsonrpc.JSONRPC_VERSION,
			Id:      2,
			Request: jsonrpc.Request{
				Method: "resources/read",
			},
			Params: struct {
				URI string `json:"uri"`
			}{
				URI: "info://static-doc",
			},
		}
		bodyBytes, _ := json.Marshal(reqBody)

		res, err := http.Post(ts.URL+"/", "application/json", bytes.NewReader(bodyBytes))
		if err != nil {
			t.Fatalf("POST request failed: %v", err)
		}
		defer res.Body.Close()

		respBytes, _ := io.ReadAll(res.Body)
		var rpcResp struct {
			Result struct {
				Contents []struct {
					URI  string `json:"uri"`
					Text string `json:"text"`
				} `json:"contents"`
			} `json:"result"`
		}
		if err := json.Unmarshal(respBytes, &rpcResp); err != nil {
			t.Fatalf("failed to unmarshal response: %v. Body: %s", err, respBytes)
		}

		if len(rpcResp.Result.Contents) != 1 || rpcResp.Result.Contents[0].Text != "hello from static text" {
			t.Errorf("expected static text content, got: %v", rpcResp.Result.Contents)
		}
	})

	t.Run("resources/read file resource template success", func(t *testing.T) {
		reqBody := jsonrpc.JSONRPCRequest{
			Jsonrpc: jsonrpc.JSONRPC_VERSION,
			Id:      3,
			Request: jsonrpc.Request{
				Method: "resources/read",
			},
			Params: struct {
				URI string `json:"uri"`
			}{
				URI: "file://" + resolvedTempDir + "/sub/test.txt",
			},
		}
		bodyBytes, _ := json.Marshal(reqBody)

		res, err := http.Post(ts.URL+"/", "application/json", bytes.NewReader(bodyBytes))
		if err != nil {
			t.Fatalf("POST request failed: %v", err)
		}
		defer res.Body.Close()

		respBytes, _ := io.ReadAll(res.Body)
		var rpcResp struct {
			Result struct {
				Contents []struct {
					URI  string `json:"uri"`
					Text string `json:"text"`
				} `json:"contents"`
			} `json:"result"`
		}
		if err := json.Unmarshal(respBytes, &rpcResp); err != nil {
			t.Fatalf("failed to unmarshal response: %v. Body: %s", err, respBytes)
		}

		if len(rpcResp.Result.Contents) != 1 || rpcResp.Result.Contents[0].Text != "hello from file" {
			t.Errorf("expected file content, got: %v", rpcResp.Result.Contents)
		}
	})

	t.Run("resources/read directory traversal blocked", func(t *testing.T) {
		reqBody := jsonrpc.JSONRPCRequest{
			Jsonrpc: jsonrpc.JSONRPC_VERSION,
			Id:      4,
			Request: jsonrpc.Request{
				Method: "resources/read",
			},
			Params: struct {
				URI string `json:"uri"`
			}{
				URI: "file://" + sandboxSubDir + "/../outside.txt",
			},
		}
		bodyBytes, _ := json.Marshal(reqBody)

		res, err := http.Post(ts.URL+"/", "application/json", bytes.NewReader(bodyBytes))
		if err != nil {
			t.Fatalf("POST request failed: %v", err)
		}
		defer res.Body.Close()

		respBytes, _ := io.ReadAll(res.Body)
		var rpcResp struct {
			Error struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}
		json.Unmarshal(respBytes, &rpcResp)

		if rpcResp.Error.Code == 0 || !strings.Contains(rpcResp.Error.Message, "escapes the allowed sandbox roots") {
			t.Errorf("expected sandboxing traversal block error, got: %s", respBytes)
		}
	})
}
