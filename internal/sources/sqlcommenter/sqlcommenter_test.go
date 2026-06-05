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

package sqlcommenter

import (
	"context"
	"net/url"
	"strings"
	"testing"

	"github.com/googleapis/mcp-toolbox/internal/util"
)

// sqlCommenterCtx returns a context with sql-commenter enabled.
func sqlCommenterCtx() context.Context {
	return util.WithSQLCommenterEnabled(context.Background(), true)
}

func TestAppendComment_SQLCommenterDisabled(t *testing.T) {
	// SQL commenter not enabled in context — statement should be unchanged
	ctx := context.Background()
	ctx = util.WithUserAgent(ctx, "1.1.0")
	ctx = util.WithGenAIMetricAttrs(ctx, &util.GenAIMetricAttrs{
		ToolName: "search_hotels",
	})

	stmt := "SELECT * FROM users"
	result := AppendComment(ctx, stmt, "postgresql")

	if result != stmt {
		t.Errorf("expected unchanged statement when sql-commenter disabled, got: %s", result)
	}
}

func TestAppendComment_EmptyContext(t *testing.T) {
	ctx := sqlCommenterCtx()
	stmt := "SELECT * FROM users"
	result := AppendComment(ctx, stmt, "")

	// No attributes available, statement should be unchanged
	if result != stmt {
		t.Errorf("expected unchanged statement, got: %s", result)
	}
}

func TestAppendComment_OnlyDbSystemName(t *testing.T) {
	ctx := sqlCommenterCtx()
	stmt := "SELECT * FROM users"
	result := AppendComment(ctx, stmt, "postgresql")

	expected := "/*db.system.name='postgresql'*/ SELECT * FROM users"
	if result != expected {
		t.Errorf("expected %s, got: %s", expected, result)
	}
}

func TestAppendComment_ServerSideAttributes(t *testing.T) {
	ctx := sqlCommenterCtx()
	ctx = util.WithUserAgent(ctx, "1.1.0")
	ctx = util.WithGenAIMetricAttrs(ctx, &util.GenAIMetricAttrs{
		ToolName: "search_hotels",
	})

	stmt := "SELECT * FROM hotels"
	result := AppendComment(ctx, stmt, "postgresql")

	// Should contain server, tool.name, db.system.name
	if !strings.Contains(result, "/*") || !strings.Contains(result, "*/") {
		t.Errorf("expected SQL comment, got: %s", result)
	}
	if !strings.Contains(result, "db.system.name='postgresql'") {
		t.Errorf("missing db.system.name, got: %s", result)
	}
	if !strings.Contains(result, "server='"+url.QueryEscape("genai-toolbox/1.1.0")+"'") {
		t.Errorf("missing server, got: %s", result)
	}
	if !strings.Contains(result, "tool.name='search_hotels'") {
		t.Errorf("missing tool.name, got: %s", result)
	}
	// Comment should be prepended
	if !strings.HasPrefix(result, "/*") {
		t.Errorf("expected comment prepended to statement, got: %s", result)
	}
}

func TestAppendComment_FullAttributes(t *testing.T) {
	ctx := sqlCommenterCtx()
	ctx = util.WithUserAgent(ctx, "1.1.0")
	ctx = util.WithGenAIMetricAttrs(ctx, &util.GenAIMetricAttrs{
		ToolName: "search_user",
	})
	ctx = util.WithTelemetryAttributes(ctx, &util.TelemetryAttributes{
		ClientName:    "toolbox-langchain-python",
		ClientVersion: "v0.1.0",
		ClientModel:   "gemini-2.5-flash",
		ClientUserID:  "user-123",
		ClientAgentID: "agent-456",
	})

	stmt := "SELECT * FROM users"
	result := AppendComment(ctx, stmt, "postgresql")

	// Verify all expected key='value' pairs are present
	expectedPairs := []string{
		"client='" + url.QueryEscape("toolbox-langchain-python/v0.1.0") + "'",
		"client.agent.id='agent-456'",
		"client.model='gemini-2.5-flash'",
		"client.user.id='user-123'",
		"db.system.name='postgresql'",
		"server='" + url.QueryEscape("genai-toolbox/1.1.0") + "'",
		"tool.name='search_user'",
	}
	for _, pair := range expectedPairs {
		if !strings.Contains(result, pair) {
			t.Errorf("missing pair %q in: %s", pair, result)
		}
	}
}

func TestAppendComment_AlphabeticalOrder(t *testing.T) {
	ctx := sqlCommenterCtx()
	ctx = util.WithUserAgent(ctx, "1.0.0")
	ctx = util.WithGenAIMetricAttrs(ctx, &util.GenAIMetricAttrs{
		ToolName: "my_tool",
	})
	ctx = util.WithTelemetryAttributes(ctx, &util.TelemetryAttributes{
		ClientName:    "test-client",
		ClientVersion: "v1",
		ClientModel:   "model-x",
	})

	stmt := "SELECT 1"
	result := AppendComment(ctx, stmt, "postgresql")

	// Extract the comment part
	commentStart := strings.Index(result, "/*")
	commentEnd := strings.Index(result, "*/")
	if commentStart == -1 || commentEnd == -1 {
		t.Fatalf("no comment found in: %s", result)
	}
	comment := result[commentStart+2 : commentEnd]
	parts := strings.Split(comment, ",")

	// Verify keys are sorted
	for i := 1; i < len(parts); i++ {
		prevKey := strings.SplitN(parts[i-1], "=", 2)[0]
		currKey := strings.SplitN(parts[i], "=", 2)[0]
		if prevKey > currKey {
			t.Errorf("keys not sorted: %s comes before %s", prevKey, currKey)
		}
	}
}

func TestAppendComment_URLEncoding(t *testing.T) {
	ctx := sqlCommenterCtx()
	ctx = util.WithTelemetryAttributes(ctx, &util.TelemetryAttributes{
		ClientName:    "my client/special",
		ClientVersion: "v1.0",
	})

	stmt := "SELECT 1"
	result := AppendComment(ctx, stmt, "")

	// The client value "my client/special/v1.0" should be URL-encoded
	if !strings.Contains(result, "client='"+url.QueryEscape("my client/special/v1.0")+"'") {
		t.Errorf("expected URL-encoded client, got: %s", result)
	}
}

func TestAppendComment_PartialClientAttributes(t *testing.T) {
	ctx := sqlCommenterCtx()
	ctx = util.WithTelemetryAttributes(ctx, &util.TelemetryAttributes{
		ClientName: "test-client",
		// No version
	})

	stmt := "SELECT 1"
	result := AppendComment(ctx, stmt, "")

	if !strings.Contains(result, "client='test-client'") {
		t.Errorf("expected client with name only, got: %s", result)
	}
}

func TestAppendComment_EmptyTelemetryAttributes(t *testing.T) {
	ctx := sqlCommenterCtx()
	ctx = util.WithTelemetryAttributes(ctx, &util.TelemetryAttributes{})

	stmt := "SELECT 1"
	result := AppendComment(ctx, stmt, "postgresql")

	// Should only have db.system.name since all telemetry attrs are empty
	if !strings.Contains(result, "db.system.name='postgresql'") {
		t.Errorf("expected db.system.name, got: %s", result)
	}
}
