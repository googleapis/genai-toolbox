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

package firestoreexecutemql_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	firestoreapi "cloud.google.com/go/firestore"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/mcp-toolbox/internal/server"
	"github.com/googleapis/mcp-toolbox/internal/sources"
	"github.com/googleapis/mcp-toolbox/internal/testutils"
	"github.com/googleapis/mcp-toolbox/internal/tools"
	"github.com/googleapis/mcp-toolbox/internal/tools/firestoremongodb/firestoreexecutemql"
	"github.com/googleapis/mcp-toolbox/internal/util/parameters"
)

type mockSource struct {
	sources.SourceConfig
	executePipelineFunc func(ctx context.Context, query string) (any, error)
}

func (m *mockSource) SourceType() string {
	return "firestore"
}

func (m *mockSource) ToConfig() sources.SourceConfig {
	return m.SourceConfig
}

func (m *mockSource) FirestoreClient() *firestoreapi.Client {
	return nil
}

func (m *mockSource) ExecutePipeline(ctx context.Context, query string) (any, error) {
	if m.executePipelineFunc != nil {
		return m.executePipelineFunc(ctx, query)
	}
	return nil, nil
}

type incompatibleSource struct {
	sources.SourceConfig
}

func (i *incompatibleSource) SourceType() string {
	return "incompatible"
}

func (i *incompatibleSource) ToConfig() sources.SourceConfig {
	return i.SourceConfig
}

func TestParseFromYamlFirestoreExecuteMQL(t *testing.T) {
	ctx, err := testutils.ContextWithNewLogger()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	tcs := []struct {
		desc string
		in   string
		want server.ToolConfigs
	}{
		{
			desc: "basic example",
			in: `
			kind: tool
			name: execute_mql_tool
			type: firestore-execute-mql
			source: my-firestore-instance
			description: Execute MQL query in Firestore
			`,
			want: server.ToolConfigs{
				"execute_mql_tool": firestoreexecutemql.Config{
					ConfigBase: tools.ConfigBase{
						Name:         "execute_mql_tool",
						Description:  "Execute MQL query in Firestore",
						AuthRequired: []string{},
					},
					Type:   "firestore-execute-mql",
					Source: "my-firestore-instance",
				},
			},
		},
		{
			desc: "with auth requirements",
			in: `
			kind: tool
			name: secure_execute_mql
			type: firestore-execute-mql
			source: prod-firestore
			description: Execute MQL with authentication
			authRequired:
				- google-auth-service
			`,
			want: server.ToolConfigs{
				"secure_execute_mql": firestoreexecutemql.Config{
					ConfigBase: tools.ConfigBase{
						Name:         "secure_execute_mql",
						Description:  "Execute MQL with authentication",
						AuthRequired: []string{"google-auth-service"},
					},
					Type:   "firestore-execute-mql",
					Source: "prod-firestore",
				},
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			_, _, _, got, _, _, err := server.UnmarshalPrimitiveConfig(ctx, testutils.FormatYaml(tc.in))
			if err != nil {
				t.Fatalf("unable to unmarshal: %s", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("incorrect parse: diff %v", diff)
			}
		})
	}
}

func TestAnnotations(t *testing.T) {
	t.Run("default destructive annotations", func(t *testing.T) {
		annotations := tools.GetAnnotationsOrDefault(nil, tools.NewDestructiveAnnotations)
		if annotations == nil {
			t.Fatal("expected non-nil annotations")
		}
		if annotations.ReadOnlyHint == nil || *annotations.ReadOnlyHint != false {
			t.Error("expected readOnlyHint to be false")
		}
		if annotations.DestructiveHint == nil || *annotations.DestructiveHint != true {
			t.Error("expected destructiveHint to be true")
		}
	})

	t.Run("custom annotations override", func(t *testing.T) {
		readOnly := true
		custom := &tools.ToolAnnotations{ReadOnlyHint: &readOnly}
		annotations := tools.GetAnnotationsOrDefault(custom, tools.NewDestructiveAnnotations)
		if annotations.ReadOnlyHint == nil || *annotations.ReadOnlyHint != true {
			t.Error("expected custom readOnlyHint to be true")
		}
	})
}

func TestFailParseFromYaml(t *testing.T) {
	ctx, err := testutils.ContextWithNewLogger()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	tcs := []struct {
		desc string
		in   string
		err  string
	}{
		{
			desc: "missing source",
			in: `
			kind: tool
			name: execute_mql_tool
			type: firestore-execute-mql
			description: Execute MQL query
			`,
			err: "Field validation for 'Source' failed on the 'required' tag",
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			_, _, _, _, _, _, err := server.UnmarshalPrimitiveConfig(ctx, testutils.FormatYaml(tc.in))
			if err == nil {
				t.Fatalf("expect parsing to fail")
			}
			if !strings.Contains(err.Error(), tc.err) {
				t.Fatalf("unexpected error: got %q, want substring %q", err.Error(), tc.err)
			}
		})
	}
}

func TestInvoke(t *testing.T) {
	ctx, err := testutils.ContextWithNewLogger()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	cfg := firestoreexecutemql.Config{
		ConfigBase: tools.ConfigBase{
			Name:        "execute_mql",
			Description: "Executes MQL query",
		},
		Type:   "firestore-execute-mql",
		Source: "my-firestore",
	}

	tool, err := cfg.Initialize(ctx)
	if err != nil {
		t.Fatalf("failed to initialize tool: %v", err)
	}

	t.Run("successful invocation find query", func(t *testing.T) {
		wantResult := []map[string]any{{"_id": "doc1", "value": 100}}
		mock := &mockSource{
			executePipelineFunc: func(ctx context.Context, query string) (any, error) {
				if query != "db.orders.find({})" {
					return nil, errors.New("unexpected query")
				}
				return wantResult, nil
			},
		}

		params := parameters.ParamValues{"query": "db.orders.find({})"}
		got, toolErr := tool.Invoke(ctx, mock, params, "")
		if toolErr != nil {
			t.Fatalf("unexpected invoke error: %v", toolErr)
		}
		if diff := cmp.Diff(wantResult, got); diff != "" {
			t.Fatalf("unexpected result: diff %v", diff)
		}
	})

	t.Run("successful invocation aggregation pipeline array", func(t *testing.T) {
		wantResult := []map[string]any{{"_id": "item1", "total": 250}}
		pipelineQuery := `[{"$match": {"status": "completed"}}, {"$group": {"_id": "$item", "total": {"$sum": "$amount"}}}]`
		mock := &mockSource{
			executePipelineFunc: func(ctx context.Context, query string) (any, error) {
				if query != pipelineQuery {
					return nil, errors.New("unexpected query")
				}
				return wantResult, nil
			},
		}

		params := parameters.ParamValues{"query": pipelineQuery}
		got, toolErr := tool.Invoke(ctx, mock, params, "")
		if toolErr != nil {
			t.Fatalf("unexpected invoke error: %v", toolErr)
		}
		if diff := cmp.Diff(wantResult, got); diff != "" {
			t.Fatalf("unexpected result: diff %v", diff)
		}
	})

	t.Run("successful invocation aggregation pipeline object", func(t *testing.T) {
		wantResult := []map[string]any{{"_id": "item1", "total": 250}}
		pipelineObj := `{"pipeline": [{"$match": {"status": "completed"}}], "explain": false}`
		mock := &mockSource{
			executePipelineFunc: func(ctx context.Context, query string) (any, error) {
				if query != pipelineObj {
					return nil, errors.New("unexpected query")
				}
				return wantResult, nil
			},
		}

		params := parameters.ParamValues{"query": pipelineObj}
		got, toolErr := tool.Invoke(ctx, mock, params, "")
		if toolErr != nil {
			t.Fatalf("unexpected invoke error: %v", toolErr)
		}
		if diff := cmp.Diff(wantResult, got); diff != "" {
			t.Fatalf("unexpected result: diff %v", diff)
		}
	})

	t.Run("missing query parameter", func(t *testing.T) {
		mock := &mockSource{}
		params := parameters.ParamValues{}
		_, toolErr := tool.Invoke(ctx, mock, params, "")
		if toolErr == nil {
			t.Fatal("expected error for missing query parameter")
		}
	})

	t.Run("incompatible source", func(t *testing.T) {
		incompat := &incompatibleSource{}
		params := parameters.ParamValues{"query": "db.orders.find({})"}
		_, toolErr := tool.Invoke(ctx, incompat, params, "")
		if toolErr == nil {
			t.Fatal("expected error for incompatible source")
		}
	})
}

func TestValidateSource(t *testing.T) {
	ctx := context.Background()
	cfg := firestoreexecutemql.Config{
		ConfigBase: tools.ConfigBase{
			Name:        "execute_mql",
			Description: "Executes MQL query",
		},
		Type:   "firestore-execute-mql",
		Source: "my-firestore",
	}
	tool, err := cfg.Initialize(ctx)
	if err != nil {
		t.Fatalf("failed to initialize tool: %v", err)
	}

	mock := &mockSource{}
	if err := tool.ValidateSource(mock); err != nil {
		t.Errorf("expected valid source, got: %v", err)
	}

	incompat := &incompatibleSource{}
	if err := tool.ValidateSource(incompat); err == nil {
		t.Error("expected error for incompatible source")
	}
}
