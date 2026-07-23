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
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/googleapis/mcp-toolbox/internal/util"
	"github.com/googleapis/mcp-toolbox/internal/util/parameters"
)

func TestLoadEmulatorMocks(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	mockFile := filepath.Join(dir, "mocks.json")
	content := `{
  "mocks": [
    {"tool_name": "search_users_bq", "parameters": {"id": 123}, "response": [{"id":123}]},
    {"tool_name": "search_users_bq", "parameters": {"email": "alice@example.com"}, "response": []}
  ]
}`
	if err := os.WriteFile(mockFile, []byte(content), 0o600); err != nil {
		t.Fatalf("failed writing mock file: %v", err)
	}

	mocks, err := loadEmulatorMocks(mockFile)
	if err != nil {
		t.Fatalf("loadEmulatorMocks returned error: %v", err)
	}
	if got := len(mocks["search_users_bq"]); got != 2 {
		t.Fatalf("expected 2 mocks for search_users_bq, got %d", got)
	}
}

func TestEmulatorToolInvoke_MatchReturnsMock(t *testing.T) {
	t.Parallel()
	tool := emulatorTool{
		name: "search_users_bq",
		mocks: []emulatorMock{{
			ToolName:   "search_users_bq",
			Parameters: map[string]any{"id": float64(123)},
			Response:   []map[string]any{{"id": float64(123), "name": "Alice"}},
		}},
	}
	params := parameters.ParamValues{{Name: "id", Value: int64(123)}}

	got, err := tool.Invoke(context.Background(), nil, params, "")
	if err != nil {
		t.Fatalf("Invoke returned error: %v", err)
	}
	rows, ok := got.([]map[string]any)
	if !ok || len(rows) != 1 || rows[0]["name"] != "Alice" {
		t.Fatalf("unexpected mock response: %#v", got)
	}
}

func TestEmulatorToolInvoke_NoMatchReturnsAgentError(t *testing.T) {
	t.Parallel()
	tool := emulatorTool{
		name: "search_users_bq",
		mocks: []emulatorMock{{
			ToolName:   "search_users_bq",
			Parameters: map[string]any{"id": float64(123)},
			Response:   []map[string]any{{"id": float64(123)}},
		}},
	}
	params := parameters.ParamValues{{Name: "id", Value: int64(999)}}

	_, err := tool.Invoke(context.Background(), nil, params, "")
	if err == nil {
		t.Fatal("expected error for unmatched mock")
	}
	if _, ok := err.(*util.AgentError); !ok {
		t.Fatalf("expected AgentError, got %T", err)
	}
}
