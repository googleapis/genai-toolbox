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

package mongodbaggregate_test

import (
	"strings"
	"testing"

	"github.com/googleapis/mcp-toolbox/internal/tools"
	"github.com/googleapis/mcp-toolbox/internal/tools/mongodb/mongodbaggregate"
	"github.com/googleapis/mcp-toolbox/internal/util/parameters"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/mcp-toolbox/internal/server"
	"github.com/googleapis/mcp-toolbox/internal/testutils"
)

func TestParseFromYamlMongoQuery(t *testing.T) {
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
            name: example_tool
            type: mongodb-aggregate
            source: my-instance
            description: some description
            database: test_db
            collection: test_coll
            readOnly: true
            pipelinePayload: |
                [{ $match: { name: {{json .name}} }}]
            pipelineParams:
                - name: name 
                  type: string
                  description: small description
			`,
			want: server.ToolConfigs{
				"example_tool": mongodbaggregate.Config{
					ConfigBase: tools.ConfigBase{
						Name:         "example_tool",
						AuthRequired: []string{},
						Description:  "some description",
					},
					Type:            "mongodb-aggregate",
					Source:          "my-instance",
					Database:        "test_db",
					Collection:      "test_coll",
					PipelinePayload: "[{ $match: { name: {{json .name}} }}]\n",
					PipelineParams: parameters.Parameters{
						&parameters.StringParameter{
							CommonParameter: parameters.CommonParameter{
								Name: "name",
								Type: "string",
								Desc: "small description",
							},
						},
					},
					ReadOnly: true,
				},
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			// Parse contents
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
	// Test default annotations for read-only tool
	t.Run("default annotations", func(t *testing.T) {
		annotations := tools.GetAnnotationsOrDefault(nil, tools.NewReadOnlyAnnotations)
		if annotations == nil {
			t.Fatal("expected non-nil annotations")
		}
		if annotations.ReadOnlyHint == nil || *annotations.ReadOnlyHint != true {
			t.Error("expected readOnlyHint to be true")
		}
	})

	// Test custom annotations override default
	t.Run("custom annotations", func(t *testing.T) {
		customReadOnly := false
		custom := &tools.ToolAnnotations{ReadOnlyHint: &customReadOnly}
		annotations := tools.GetAnnotationsOrDefault(custom, tools.NewReadOnlyAnnotations)
		if annotations.ReadOnlyHint == nil || *annotations.ReadOnlyHint != false {
			t.Error("expected custom readOnlyHint to be false")
		}
	})
}

func TestFailParseFromYamlMongoQuery(t *testing.T) {
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
			desc: "Invalid method",
			in: `
            kind: tool
            name: example_tool
            type: mongodb-aggregate
            source: my-instance
            description: some description
            collection: test_coll
            pipelinePayload: |
              [{ $match: { name : {{json .name}} }}]
			`,
			err: `unable to parse tool "example_tool" as type "mongodb-aggregate"`,
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			_, _, _, _, _, _, err := server.UnmarshalPrimitiveConfig(ctx, testutils.FormatYaml(tc.in))
			if err == nil {
				t.Fatalf("expect parsing to fail")
			}
			errStr := err.Error()
			if !strings.Contains(errStr, tc.err) {
				t.Fatalf("unexpected error string: got %q, want substring %q", errStr, tc.err)
			}
		})
	}

}

func findCollectionParam(params []parameters.ParameterManifest) *parameters.ParameterManifest {
	for i := range params {
		if params[i].Name == "collection" {
			return &params[i]
		}
	}
	return nil
}

func TestRuntimeCollectionParam(t *testing.T) {
	ctx, err := testutils.ContextWithNewLogger()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	// A config that omits collection should still parse, since it is no longer required.
	if _, _, _, _, _, _, err := server.UnmarshalPrimitiveConfig(ctx, testutils.FormatYaml(noCollectionConfig)); err != nil {
		t.Fatalf("expected config without collection to parse, got: %s", err)
	}

	// When collection is omitted, it is exposed as a required runtime parameter.
	runtimeCfg := mongodbaggregate.Config{
		ConfigBase: tools.ConfigBase{Name: "example_tool", Description: "some description"},
	}
	runtimeTool, err := runtimeCfg.Initialize(ctx)
	if err != nil {
		t.Fatalf("unable to initialize tool: %s", err)
	}
	if p := findCollectionParam(runtimeTool.StaticManifest().Parameters); p == nil {
		t.Error("expected a runtime collection parameter when collection is omitted from config")
	} else if !p.Required {
		t.Error("expected the runtime collection parameter to be required")
	}

	// When collection is set in the config, no runtime parameter is exposed.
	staticCfg := mongodbaggregate.Config{
		ConfigBase: tools.ConfigBase{Name: "example_tool", Description: "some description"},
		Collection: "test_coll",
	}
	staticTool, err := staticCfg.Initialize(ctx)
	if err != nil {
		t.Fatalf("unable to initialize tool: %s", err)
	}
	if p := findCollectionParam(staticTool.StaticManifest().Parameters); p != nil {
		t.Error("did not expect a collection parameter when collection is set in the config")
	}
}

var noCollectionConfig = `
            kind: tool
            name: example_tool
            type: mongodb-aggregate
            source: my-instance
            description: some description
            database: test_db
            pipelinePayload: |
                [{ $match: { name: {{json .name}} }}]
            pipelineParams:
                - name: name
                  type: string
                  description: small description
`

func TestRuntimeCollectionAllowedValues(t *testing.T) {
	ctx, err := testutils.ContextWithNewLogger()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	// collectionAllowedValues should restrict the injected runtime collection parameter.
	cfg := mongodbaggregate.Config{
		ConfigBase:              tools.ConfigBase{Name: "example_tool", Description: "some description"},
		CollectionAllowedValues: []string{"orders", "customers"},
	}
	tool, err := cfg.Initialize(ctx)
	if err != nil {
		t.Fatalf("unable to initialize tool: %s", err)
	}
	params, err := tool.GetParameters(nil)
	if err != nil {
		t.Fatalf("unable to get parameters: %s", err)
	}

	var collectionParam *parameters.StringParameter
	for _, p := range params {
		if sp, ok := p.(*parameters.StringParameter); ok && sp.GetName() == "collection" {
			collectionParam = sp
		}
	}
	if collectionParam == nil {
		t.Fatal("expected an injected collection parameter")
	}
	if len(collectionParam.AllowedValues) != 2 {
		t.Fatalf("expected 2 allowed values on the collection parameter, got %d", len(collectionParam.AllowedValues))
	}
}

func TestRuntimeCollectionDuplicateParam(t *testing.T) {
	ctx, err := testutils.ContextWithNewLogger()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	// When collection is omitted and a pipeline param is also named "collection",
	// initialization should fail on the duplicate parameter.
	collectionParam := parameters.NewStringParameter("collection", "duplicate", parameters.WithStringRequired(true))
	cfg := mongodbaggregate.Config{
		ConfigBase:     tools.ConfigBase{Name: "example_tool", Description: "some description"},
		PipelineParams: parameters.Parameters{collectionParam},
	}
	if _, err := cfg.Initialize(ctx); err == nil {
		t.Fatal("expected initialization to fail on duplicate collection parameter")
	}
}
