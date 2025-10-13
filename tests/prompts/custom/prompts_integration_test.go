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

package custom_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"testing"
	"time"

	"github.com/googleapis/genai-toolbox/internal/server/mcp/jsonrpc"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/tests"
)

func getPromptsConfig() map[string]any {
	return map[string]any{
		"prompts": map[string]any{
			"prompt1": map[string]any{
				"description": "description1.",
				"arguments": []any{
					map[string]any{
						"name":        "arg1",
						"description": "Description of arg1 for prompt1",
					},
				},
				"messages": []any{
					map[string]any{
						"content": "content arg1: {{.arg1}}",
					},
				},
			},
			"prompt2": map[string]any{
				"description": "description2",
				"arguments": []any{
					map[string]any{
						"name":        "arg1",
						"description": "Description for arg1 for prompt2",
					},
					map[string]any{
						"name":        "arg2",
						"description": "Description for arg2 for prompt2",
					},
				},
				"messages": []any{
					map[string]any{
						"role":    "user",
						"content": "message1 args: {{.arg1}}, {{.arg2}}",
					},
					map[string]any{
						"role":    "assistant",
						"content": "{{.arg1}}",
					},
				},
			},
		},
	}
}

// TestMCPPromptsIntegration is the main entrypoint for the prompts integration tests.
func TestMCPPromptsIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	cmd, cleanup, err := tests.StartCmd(ctx, getPromptsConfig())
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

	runMCPListPromptsTest(t)
	runMCPGetPromptTest(t)
}

func runMCPListPromptsTest(t *testing.T) {
	t.Run("MCP_prompts/list", func(t *testing.T) {
		api := "http://127.0.0.1:5000/mcp"
		reqBody := jsonrpc.JSONRPCRequest{
			Jsonrpc: "2.0",
			Id:      "list-prompts-1",
			Request: jsonrpc.Request{Method: "prompts/list"},
		}
		jsonBody, _ := json.Marshal(reqBody)

		resp, err := http.Post(api, "application/json", bytes.NewBuffer(jsonBody))
		if err != nil {
			t.Fatalf("failed to send request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected status OK; got %s", resp.Status)
		}

		var result struct {
			Result struct {
				Prompts []map[string]any `json:"prompts"`
			} `json:"result"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if len(result.Result.Prompts) != 2 {
			t.Errorf("expected 2 prompts, got %d", len(result.Result.Prompts))
		}
	})
}

func runMCPGetPromptTest(t *testing.T) {
	t.Run("MCP_prompts/get", func(t *testing.T) {
		api := "http://127.0.0.1:5000/mcp"
		reqBody := jsonrpc.JSONRPCRequest{
			Jsonrpc: "2.0",
			Id:      "get-prompt-2",
			Request: jsonrpc.Request{Method: "prompts/get"},
			Params: map[string]any{
				"name": "prompt2",
				"arguments": map[string]any{
					"arg1": "value1",
					"arg2": "value2",
				},
			},
		}
		jsonBody, _ := json.Marshal(reqBody)

		resp, err := http.Post(api, "application/json", bytes.NewBuffer(jsonBody))
		if err != nil {
			t.Fatalf("failed to send request: %v", err)
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected status OK; got %s, body: %s", resp.Status, string(body))
		}

		var result map[string]any
		if err := json.Unmarshal(body, &result); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		messages := result["result"].(map[string]any)["messages"].([]any)
		if len(messages) != 2 {
			t.Fatalf("expected 2 messages, got %d", len(messages))
		}

		// Check first message
		msg1 := messages[0].(map[string]any)
		content1 := msg1["content"].(map[string]any)["text"].(string)
		expectedContent1 := "message1 args: value1, value2"
		if content1 != expectedContent1 {
			t.Errorf("unexpected content in message 1.\nGot: %q\nWant: %q", content1, expectedContent1)
		}

		// Check second message
		msg2 := messages[1].(map[string]any)
		content2 := msg2["content"].(map[string]any)["text"].(string)
		expectedContent2 := "value1"
		if content2 != expectedContent2 {
			t.Errorf("unexpected content in message 2.\nGot: %q\nWant: %q", content2, expectedContent2)
		}
	})
}
