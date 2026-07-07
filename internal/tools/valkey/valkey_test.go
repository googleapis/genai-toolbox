// Copyright 2024 Google LLC
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

package valkey_test

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/mcp-toolbox/internal/embeddingmodels"
	"github.com/googleapis/mcp-toolbox/internal/server"
	"github.com/googleapis/mcp-toolbox/internal/sources"
	"github.com/googleapis/mcp-toolbox/internal/testutils"
	"github.com/googleapis/mcp-toolbox/internal/tools"
	"github.com/googleapis/mcp-toolbox/internal/tools/valkey"
	"github.com/googleapis/mcp-toolbox/internal/util/parameters"
	valkeyio "github.com/valkey-io/valkey-go"
)

func TestParseFromYamlvalkey(t *testing.T) {
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
			name: valkey_tool
			type: valkey
			source: my-valkey-instance
			description: some description
			commands:
				- [SET, greeting, "hello, {{.name}}"]
				- [GET, id]
			parameters:
				- name: name
				  type: string
				  description: user name
			`,
			want: server.ToolConfigs{
				"valkey_tool": valkey.Config{
					ConfigBase: tools.ConfigBase{
						Name:         "valkey_tool",
						Description:  "some description",
						AuthRequired: []string{},
					},
					Type:     "valkey",
					Source:   "my-valkey-instance",
					Commands: [][]string{{"SET", "greeting", "hello, {{.name}}"}, {"GET", "id"}},
					Parameters: []parameters.Parameter{
						parameters.NewStringParameter("name", "user name"),
					},
				},
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			_, _, _, got, _, _, err := server.UnmarshalResourceConfig(ctx, testutils.FormatYaml(tc.in))
			if err != nil {
				t.Fatalf("unable to unmarshal: %s", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("incorrect parse: diff %v", diff)
			}
		})
	}

}

type mockEmbeddingModel struct {
	embedFn func(context.Context, []string) ([][]float32, error)
}

func (m mockEmbeddingModel) EmbeddingModelType() string {
	return "mock"
}

func (m mockEmbeddingModel) ToConfig() embeddingmodels.EmbeddingModelConfig {
	return nil
}

func (m mockEmbeddingModel) EmbedParameters(ctx context.Context, texts []string) ([][]float32, error) {
	return m.embedFn(ctx, texts)
}

func TestValkeyEmbedParams(t *testing.T) {
	ctx := context.Background()

	cfg := valkey.Config{
		ConfigBase: tools.ConfigBase{
			Name:        "valkey_tool",
			Description: "some description",
		},
		Type:     "valkey",
		Source:   "my-source",
		Commands: [][]string{{"HSET", "doc:1", "vec", "$query"}},
		Parameters: parameters.Parameters{
			&parameters.StringParameter{
				CommonParameter: parameters.CommonParameter{
					Name:       "query",
					Type:       parameters.TypeString,
					Desc:       "some description",
					EmbeddedBy: "my_model",
				},
			},
		},
	}

	tool, err := cfg.Initialize(nil)
	if err != nil {
		t.Fatalf("failed to initialize tool: %v", err)
	}

	mockModel := mockEmbeddingModel{
		embedFn: func(ctx context.Context, texts []string) ([][]float32, error) {
			if len(texts) != 1 || texts[0] != "hello" {
				return nil, fmt.Errorf("unexpected texts: %v", texts)
			}
			return [][]float32{{0.1, -0.2, 0.3}}, nil
		},
	}

	embeddingModelsMap := map[string]embeddingmodels.EmbeddingModel{
		"my_model": mockModel,
	}

	paramValues := parameters.ParamValues{
		{
			Name:  "query",
			Value: "hello",
		},
	}

	gotParams, err := tool.EmbedParams(ctx, paramValues, embeddingModelsMap)
	if err != nil {
		t.Fatalf("EmbedParams failed: %v", err)
	}

	if len(gotParams) != 1 {
		t.Fatalf("expected 1 param value, got %d", len(gotParams))
	}

	gotVal, ok := gotParams[0].Value.(string)
	if !ok {
		t.Fatalf("expected string value, got %T", gotParams[0].Value)
	}

	if len(gotVal) != 12 { // 3 floats * 4 bytes each
		t.Fatalf("expected binary string length of 12, got %d", len(gotVal))
	}

	floats := make([]float32, 3)
	for i := 0; i < 3; i++ {
		bits := binary.LittleEndian.Uint32([]byte(gotVal[i*4 : (i+1)*4]))
		floats[i] = math.Float32frombits(bits)
	}

	wantFloats := []float32{0.1, -0.2, 0.3}
	if diff := cmp.Diff(wantFloats, floats); diff != "" {
		t.Fatalf("incorrect floats: diff %s", diff)
	}
}

type mockSource struct {
	executedCmds [][]string
}

func (m *mockSource) SourceType() string {
	return "valkey"
}

func (m *mockSource) ToConfig() sources.SourceConfig {
	return nil
}

func (m *mockSource) ValkeyClient() valkeyio.Client {
	return nil
}

func (m *mockSource) RunCommand(ctx context.Context, cmds [][]string) (any, error) {
	m.executedCmds = cmds
	return "ok", nil
}

type mockSourceProvider struct {
	src sources.Source
}

func (p mockSourceProvider) GetSource(name string) (sources.Source, bool) {
	return p.src, true
}

func TestValkeyToolInvokeWithEmbedding(t *testing.T) {
	ctx := context.Background()

	cfg := valkey.Config{
		ConfigBase: tools.ConfigBase{
			Name:        "search_products",
			Description: "Search products using vector similarity",
		},
		Type:     "valkey",
		Source:   "my-valkey",
		Commands: [][]string{{"FT.SEARCH", "idx:products", "(*)=>[KNN 5 @vector_field $query_vec]", "PARAMS", "2", "query_vec", "$query_vec", "DIALECT", "2"}},
		Parameters: parameters.Parameters{
			&parameters.StringParameter{
				CommonParameter: parameters.CommonParameter{
					Name:       "query_vec",
					Type:       parameters.TypeString,
					Desc:       "The query text to embed",
					EmbeddedBy: "gemini_model",
				},
			},
		},
	}

	tool, err := cfg.Initialize(nil)
	if err != nil {
		t.Fatalf("failed to initialize tool: %v", err)
	}

	mockModel := mockEmbeddingModel{
		embedFn: func(ctx context.Context, texts []string) ([][]float32, error) {
			return [][]float32{{0.1, -0.2, 0.3}}, nil
		},
	}
	embeddingModelsMap := map[string]embeddingmodels.EmbeddingModel{
		"gemini_model": mockModel,
	}

	mSrc := &mockSource{}
	provider := mockSourceProvider{src: mSrc}

	rawParams := map[string]any{
		"query_vec": "hello",
	}
	parsedParams, err := parameters.ParseParams(tool.GetParameters(), rawParams, nil)
	if err != nil {
		t.Fatalf("failed to parse params: %v", err)
	}

	embeddedParams, err := tool.EmbedParams(ctx, parsedParams, embeddingModelsMap)
	if err != nil {
		t.Fatalf("failed to embed params: %v", err)
	}

	_, tbErr := tool.Invoke(ctx, provider, embeddedParams, "")
	if tbErr != nil {
		t.Fatalf("tool invocation failed: %v", tbErr)
	}

	if len(mSrc.executedCmds) != 1 {
		t.Fatalf("expected 1 command, got %d", len(mSrc.executedCmds))
	}
	cmd := mSrc.executedCmds[0]

	if len(cmd) != 9 {
		t.Fatalf("expected 9 args, got %d", len(cmd))
	}

	binStr := cmd[6]
	if len(binStr) != 12 {
		t.Fatalf("expected binary string length 12, got %d", len(binStr))
	}

	floats := make([]float32, 3)
	for i := 0; i < 3; i++ {
		bits := binary.LittleEndian.Uint32([]byte(binStr[i*4 : (i+1)*4]))
		floats[i] = math.Float32frombits(bits)
	}
	wantFloats := []float32{0.1, -0.2, 0.3}
	if diff := cmp.Diff(wantFloats, floats); diff != "" {
		t.Fatalf("incorrect floats in executed command: diff %s", diff)
	}

	wantCmd := []string{"FT.SEARCH", "idx:products", "(*)=>[KNN 5 @vector_field $query_vec]", "PARAMS", "2", "query_vec", binStr, "DIALECT", "2"}
	if diff := cmp.Diff(wantCmd, cmd); diff != "" {
		t.Fatalf("command mismatch (-want +got):\n%s", diff)
	}
}
