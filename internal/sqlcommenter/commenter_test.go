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
	"strings"
	"testing"
)

func testContext() context.Context {
	ctx := context.Background()
	ctx = WithToolName(ctx, "testTool")
	ctx = WithAgentName(ctx, "testAgent")
	return ctx
}

func TestAppendComment_Postgres(t *testing.T) {
	ctx := testContext()
	ctx = WithDBDriver(ctx, "postgres")
	
	sql := "SELECT * FROM users WHERE id = 1"
	got := AppendComment(ctx, sql)
	
	// The query should start with the original sql and end with the comment block,
	// or the comment block should be prepended before the semicolon/end.
	// We'll just verify the comment string is present.
	expectedComment := "/*action='testTool',application='mcp-toolbox',controller='testAgent',db_driver='postgres',framework='mcp-toolbox'*/"
	
	if !strings.Contains(got, expectedComment) {
		t.Fatalf("expected comment %s not found in got=%s", expectedComment, got)
	}
}

func TestAppendComment_NoSQL_Cassandra(t *testing.T) {
	ctx := testContext()
	ctx = WithDBDriver(ctx, "cassandra")
	
	sql := "SELECT * FROM keyspace.table"
	got := AppendComment(ctx, sql)
	
	expectedComment := "/*action='testTool',application='mcp-toolbox',controller='testAgent',db_driver='cassandra',framework='mcp-toolbox'*/"
	
	if !strings.Contains(got, expectedComment) {
		t.Fatalf("expected comment %s not found in got=%s", expectedComment, got)
	}
}

func TestAppendComment_NoMetadata(t *testing.T) {
	// Need to check what happens when only appName and framework are present (they are defaults).
	ctx := context.Background()
	sql := "SELECT 1"
	got := AppendComment(ctx, sql)
	
	if !strings.Contains(got, "/*application='mcp-toolbox',framework='mcp-toolbox'*/") {
		t.Fatalf("expected basic default comment, got=%s", got)
	}
}


