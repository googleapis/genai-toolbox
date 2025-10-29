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

package server

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/googleapis/genai-toolbox/internal/log"
	"github.com/googleapis/genai-toolbox/internal/prompts"
	"github.com/googleapis/genai-toolbox/internal/telemetry"
	"github.com/googleapis/genai-toolbox/internal/tools"
)

// fakeVersionString is used as a temporary version string in tests
const fakeVersionString = "0.0.0"

var (
	_ tools.Tool     = &MockTool{}
	_ prompts.Prompt = &MockPrompt{}
)

// MockTool is used to mock tools in tests
type MockTool struct {
	Name                         string
	Description                  string
	Params                       []tools.Parameter
	manifest                     tools.Manifest
	unauthorized                 bool
	requiresClientAuthrorization bool
}

func (t MockTool) Invoke(context.Context, tools.ParamValues, tools.AccessToken) (any, error) {
	mock := []any{t.Name}
	return mock, nil
}

// claims is a map of user info decoded from an auth token
func (t MockTool) ParseParams(data map[string]any, claimsMap map[string]map[string]any) (tools.ParamValues, error) {
	return tools.ParseParams(t.Params, data, claimsMap)
}

func (t MockTool) Manifest() tools.Manifest {
	pMs := make([]tools.ParameterManifest, 0, len(t.Params))
	for _, p := range t.Params {
		pMs = append(pMs, p.Manifest())
	}
	return tools.Manifest{Description: t.Description, Parameters: pMs}
}

func (t MockTool) Authorized(verifiedAuthServices []string) bool {
	// defaulted to true
	return !t.unauthorized
}

func (t MockTool) RequiresClientAuthorization() bool {
	// defaulted to false
	return t.requiresClientAuthrorization
}

func (t MockTool) McpManifest() tools.McpManifest {
	properties := make(map[string]tools.ParameterMcpManifest)
	required := make([]string, 0)
	authParams := make(map[string][]string)

	for _, p := range t.Params {
		name := p.GetName()
		paramManifest, authParamList := p.McpManifest()
		properties[name] = paramManifest
		required = append(required, name)

		if len(authParamList) > 0 {
			authParams[name] = authParamList
		}
	}

	toolsSchema := tools.McpToolsSchema{
		Type:       "object",
		Properties: properties,
		Required:   required,
	}

	mcpManifest := tools.McpManifest{
		Name:        t.Name,
		Description: t.Description,
		InputSchema: toolsSchema,
	}

	if len(authParams) > 0 {
		mcpManifest.Metadata = map[string]any{
			"toolbox/authParams": authParams,
		}
	}

	return mcpManifest
}

// MockPrompt is used to mock prompts in tests
type MockPrompt struct {
	Name        string
	Description string
	Args        prompts.Arguments
}

func (p MockPrompt) SubstituteParams(vals tools.ParamValues) (any, error) {
	return []prompts.Message{
		{
			Role:    "user",
			Content: fmt.Sprintf("substituted %s", p.Name),
		},
	}, nil
}

func (p MockPrompt) ParseArgs(data map[string]any, claimsMap map[string]map[string]any) (tools.ParamValues, error) {
	var parameters tools.Parameters
	for _, arg := range p.Args {
		parameters = append(parameters, arg.Parameter)
	}
	return tools.ParseParams(parameters, data, claimsMap)
}

func (p MockPrompt) Manifest() prompts.Manifest {
	var argManifests []tools.ParameterManifest
	for _, arg := range p.Args {
		argManifests = append(argManifests, arg.Manifest())
	}
	return prompts.Manifest{
		Description: p.Description,
		Arguments:   argManifests,
	}
}

func (p MockPrompt) McpManifest() prompts.McpManifest {
	return prompts.GetMcpManifest(p.Name, p.Description, p.Args)
}

var tool1 = MockTool{
	Name:   "no_params",
	Params: []tools.Parameter{},
}

var tool2 = MockTool{
	Name: "some_params",
	Params: tools.Parameters{
		tools.NewIntParameter("param1", "This is the first parameter."),
		tools.NewIntParameter("param2", "This is the second parameter."),
	},
}

var tool3 = MockTool{
	Name:        "array_param",
	Description: "some description",
	Params: tools.Parameters{
		tools.NewArrayParameter("my_array", "this param is an array of strings", tools.NewStringParameter("my_string", "string item")),
	},
}

var tool4 = MockTool{
	Name:         "unauthorized_tool",
	Params:       []tools.Parameter{},
	unauthorized: true,
}

var tool5 = MockTool{
	Name:                         "require_client_auth_tool",
	Params:                       []tools.Parameter{},
	requiresClientAuthrorization: true,
}

var prompt1 = MockPrompt{
	Name: "prompt1",
	Args: prompts.Arguments{},
}

var prompt2 = MockPrompt{
	Name: "prompt2",
	Args: prompts.Arguments{
		{Parameter: tools.NewStringParameter("arg1", "This is the first argument.")},
	},
}

// setUpResources setups resources to test against
func setUpResources(t *testing.T, mockTools []MockTool, mockPrompts []MockPrompt) (map[string]tools.Tool, map[string]tools.Toolset, map[string]prompts.Prompt, map[string]prompts.Promptset) {
	toolsMap := make(map[string]tools.Tool)
	var allTools []string
	for _, tool := range mockTools {
		tool.manifest = tool.Manifest()
		toolsMap[tool.Name] = tool
		allTools = append(allTools, tool.Name)
	}

	toolsets := make(map[string]tools.Toolset)
	for name, l := range map[string][]string{
		"":           allTools,
		"tool1_only": {allTools[0]},
		"tool2_only": {allTools[1]},
	} {
		tc := tools.ToolsetConfig{Name: name, ToolNames: l}
		m, err := tc.Initialize(fakeVersionString, toolsMap)
		if err != nil {
			t.Fatalf("unable to initialize toolset %q: %s", name, err)
		}
		toolsets[name] = m
	}

	promptsMap := make(map[string]prompts.Prompt)
	var allPrompts []string
	for _, prompt := range mockPrompts {
		promptsMap[prompt.Name] = prompt
		allPrompts = append(allPrompts, prompt.Name)
	}

	promptsets := make(map[string]prompts.Promptset)
	if len(allPrompts) > 0 {
		psc := prompts.PromptsetConfig{Name: "", PromptNames: allPrompts}
		ps, err := psc.Initialize(fakeVersionString, promptsMap)
		if err != nil {
			t.Fatalf("unable to initialize default promptset: %s", err)
		}
		promptsets[""] = ps
	}

	return toolsMap, toolsets, promptsMap, promptsets
}

// setUpServer create a new server with tools, toolsets, prompts, and promptsets.
func setUpServer(t *testing.T, router string, tools map[string]tools.Tool, toolsets map[string]tools.Toolset, prompts map[string]prompts.Prompt, promptsets map[string]prompts.Promptset) (chi.Router, func()) {
	ctx, cancel := context.WithCancel(context.Background())

	testLogger, err := log.NewStdLogger(os.Stdout, os.Stderr, "info")
	if err != nil {
		t.Fatalf("unable to initialize logger: %s", err)
	}

	otelShutdown, err := telemetry.SetupOTel(ctx, fakeVersionString, "", false, "toolbox")
	if err != nil {
		t.Fatalf("unable to setup otel: %s", err)
	}

	instrumentation, err := telemetry.CreateTelemetryInstrumentation(fakeVersionString)
	if err != nil {
		t.Fatalf("unable to create custom metrics: %s", err)
	}

	sseManager := newSseManager(ctx)

	resourceManager := NewResourceManager(nil, nil, tools, toolsets, prompts, promptsets)

	server := Server{
		version:         fakeVersionString,
		logger:          testLogger,
		instrumentation: instrumentation,
		sseManager:      sseManager,
		ResourceMgr:     resourceManager,
	}

	var r chi.Router
	switch router {
	case "api":
		r, err = apiRouter(&server)
		if err != nil {
			t.Fatalf("unable to initialize api router: %s", err)
		}
	case "mcp":
		r, err = mcpRouter(&server)
		if err != nil {
			t.Fatalf("unable to initialize mcp router: %s", err)
		}
	default:
		t.Fatalf("unknown router")
	}
	shutdown := func() {
		// cancel context
		cancel()
		// shutdown otel
		err := otelShutdown(ctx)
		if err != nil {
			t.Fatalf("error shutting down OpenTelemetry: %s", err)
		}
	}

	return r, shutdown
}

func runServer(r chi.Router, tls bool) *httptest.Server {
	var ts *httptest.Server
	if tls {
		ts = httptest.NewTLSServer(r)
	} else {
		ts = httptest.NewServer(r)
	}
	return ts
}

func runRequest(ts *httptest.Server, method, path string, body io.Reader, header map[string]string) (*http.Response, []byte, error) {
	req, err := http.NewRequest(method, ts.URL+path, body)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range header {
		req.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to send request: %w", err)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to read request body: %w", err)
	}
	defer resp.Body.Close()

	return resp, respBody, nil
}
