// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package mcpserver

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/modelcontextprotocol/go-sdk/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.opentelemetry.io/otel/trace"
)

const SourceKind string = "mcp-server"

type SpecVersion string

const (
	NOV_2024 SpecVersion = "2024-11-05"
	MAR_2025 SpecVersion = "2025-03-26"
	JUN_2025 SpecVersion = "2025-06-18"
)

type TransportType string

const (
	STDIO TransportType = "stdio"
	// MCP Specification deprecated this transport
	SSE  TransportType = "sse"
	HTTP TransportType = "http"
)

type AuthMethod string

const (
	None   AuthMethod = "none"
	ApiKey AuthMethod = "apiKey"
	Bearer AuthMethod = "bearer"
)

// validate interface
var _ sources.SourceConfig = Config{}

func init() {
	if !sources.Register(SourceKind, newConfig) {
		panic(fmt.Sprintf("source kind %q already registered", SourceKind))
	}
}

func newConfig(ctx context.Context, name string, decoder *yaml.Decoder) (sources.SourceConfig, error) {
	// Default auth method -> None, Transport -> http, SpecVersion -> MARCH_25
	actual := Config{
		Name:        name,
		AuthMethod:  None,
		Transport:   HTTP,
		SpecVersion: MAR_2025,
	}
	if err := decoder.DecodeContext(ctx, &actual); err != nil {
		return nil, err
	}
	return actual, nil
}

type Config struct {
	Name        string        `yaml:"name" validate:"required"`
	Kind        string        `yaml:"kind" validate:"required"`
	Endpoint    string        `yaml:"endpoint" validate:"required"`
	SpecVersion SpecVersion   `yaml:"specVersion"`
	Transport   TransportType `yaml:"transport"`
	AuthMethod  AuthMethod    `yaml:"authMethod"`
	AuthSecret  string        `yaml:"authSecret"`
}

func (c Config) SourceConfigKind() string {
	return SourceKind
}

func (c Config) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {
	// Validate the endpoint is valid uri
	_, err := url.ParseRequestURI(c.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed when parsing endpoint: %v", err)
	}

	// Validate the spec version is supported
	if c.SpecVersion != NOV_2024 && c.SpecVersion != MAR_2025 && c.SpecVersion != JUN_2025 {
		return nil, fmt.Errorf("unsupported specVersion: %s", c.SpecVersion)
	}

	// TODO: support stdio -- can we only have one max stdio server?
	// Validate the transport type is supported
	if c.Transport != SSE && c.Transport != HTTP {
		return nil, fmt.Errorf("unsupported transport type: %s", c.Transport)
	}

	// Validate the auth method is supported
	if c.AuthMethod != None && c.AuthMethod != ApiKey && c.AuthMethod != Bearer {
		return nil, fmt.Errorf("unsupported authMethod: %s", c.AuthMethod)
	}

	// Validate the auth secret is provided if required
	if c.AuthMethod != None && c.AuthSecret == "" {
		return nil, fmt.Errorf("authSecret is required when authMethod is set")
	}

	// TODO: Add the genai toolbox version info here
	var mcpImplementation = mcp.Implementation{
		Name:    "genai-toolbox-client",
		Version: "0.1.0",
	}
	// TODO: Hook into ToolListChangedHandler option for refresh
	var client *mcp.Client = mcp.NewClient(&mcpImplementation, &mcp.ClientOptions{})

	// client := mcp.NewClient()
	src := &Source{
		Name:        c.Name,
		Kind:        c.Kind,
		Endpoint:    c.Endpoint,
		SpecVersion: c.SpecVersion,
		Transport:   c.Transport,
		AuthMethod:  c.AuthMethod,
		AuthSecret:  c.AuthSecret,
		Client:      client,
	}
	return src, nil
}

var _ sources.Source = &Source{}

type Source struct {
	Name        string
	Kind        string
	Endpoint    string
	SpecVersion SpecVersion
	Transport   TransportType
	AuthMethod  AuthMethod
	AuthSecret  string
	Client      *mcp.Client
}

func (s *Source) SourceKind() string {
	return SourceKind
}

// TODO: run gofunc and maintain session + add auth
func (s *Source) getSession(ctx context.Context) (*mcp.ClientSession, error) {
	var defaultHeaders = map[string]string{}
	switch s.AuthMethod {
	case Bearer:
		defaultHeaders["Authorization"] = fmt.Sprintf("Bearer %s", s.AuthSecret)
	case ApiKey:
		defaultHeaders["X-API-KEY"] = s.AuthSecret
	}
	var httpTransport = &CustomAuthTransport{
		RoundTripper: http.DefaultTransport,
		Headers:      defaultHeaders,
	}
	var client *http.Client = &http.Client{
		Transport: httpTransport,
	}

	var transport mcp.Transport
	switch s.Transport {
	case SSE:
		transport = mcp.NewSSEClientTransport(s.Endpoint, &mcp.SSEClientTransportOptions{HTTPClient: client})
	case HTTP:
		transport = mcp.NewStreamableClientTransport(s.Endpoint, &mcp.StreamableClientTransportOptions{HTTPClient: client})
	default:
		transport = mcp.NewStdioTransport()
	}
	return s.Client.Connect(ctx, transport)
}

func (s *Source) GetTools(ctx context.Context) ([]tools.Tool, error) {
	// fmt.Printf("Attempting to connect? to endpoint %s\n", s.Endpoint)
	session, err := s.getSession(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MCP server: %w", err)
	}
	defer session.Close()
	// fmt.Println("Connecting?")

	remoteServerTools, err := session.ListTools(ctx, &mcp.ListToolsParams{})
	if err != nil {
		return nil, fmt.Errorf("failed to list/tools on MCP server of %s: %w", s.Name, err)
	}

	var mcpTools []MCPServerTool = make([]MCPServerTool, len(remoteServerTools.Tools))
	for i, tool := range remoteServerTools.Tools {
		var inputSchema tools.McpToolsSchema = GetInputSchema(tool)
		var outputSchema tools.McpToolsSchema = GetOutputSchema(tool)
		var toolCallParameters = make(tools.Parameters, len(inputSchema.Properties))

		mcpTools[i] = MCPServerTool{
			Source: s,
			Name:   tool.Name,
			manifest: tools.Manifest{
				Description: tool.Description,
				Parameters:  []tools.ParameterManifest{},
			},
			mcpManifest: tools.McpManifest{
				Name:         tool.Name,
				Description:  tool.Description,
				InputSchema:  inputSchema,
				OutputSchema: outputSchema,
			},
			// Parameters: tool.InputSchema.ContentSchema,
			Parameters: toolCallParameters,
			// TODO: Support output shape
		}
	}

	return toToolboxTools(mcpTools), nil
}

// GetMyInterfaces returns a slice of MyInterface from a slice of MyStruct.
func toToolboxTools(structs []MCPServerTool) []tools.Tool {
	interfaces := make([]tools.Tool, len(structs))
	for i, s := range structs {
		interfaces[i] = s // Assigning a concrete struct to an interface element.
	}
	return interfaces
}

var _ tools.Tool = MCPServerTool{}

type MCPServerTool struct {
	Source      *Source
	Name        string
	Parameters  tools.Parameters
	manifest    tools.Manifest
	mcpManifest tools.McpManifest
}

func (t MCPServerTool) Invoke(ctx context.Context, params tools.ParamValues) ([]any, error) {
	// Call a tool on the server.
	session, err := t.Source.getSession(ctx)
	if err != nil {
		return nil, err
	}
	defer session.Close()

	toolCallRequest := &mcp.CallToolParams{
		Name:      t.Name,
		Arguments: params.AsMap(),
	}
	res, err := session.CallTool(ctx, toolCallRequest)
	if err != nil || res.IsError {
		return nil, fmt.Errorf("call mcp tool failed: %v", err)
	}

	// TODO: Work around..Make Invoke call return []mcp.Content
	var r = make([]any, 0, len(res.Content))
	r = append(r, res.Content)

	return r, nil
}

func (t MCPServerTool) ParseParams(data map[string]any, claimsMap map[string]map[string]any) (tools.ParamValues, error) {
	params := make([]tools.ParamValue, 0, len(data))
	for k, v := range data {
		// paramAuthServices := p.GetAuthServices()
		name := k
		// if len(paramAuthServices) == 0 {
		// 	// parse non auth-required parameter
		// 	var ok bool
		// 	v, ok = data[name]
		// 	if !ok {
		// 		v = p.GetDefault()
		// 		if v == nil {
		// 			return nil, fmt.Errorf("parameter %q is required", name)
		// 		}
		// 	}
		// } else {
		// 	// parse authenticated parameter
		// 	var err error
		// 	v, err = parseFromAuthService(paramAuthServices, claimsMap)
		// 	if err != nil {
		// 		return nil, fmt.Errorf("error parsing authenticated parameter %q: %w", name, err)
		// 	}
		// }
		// newV, err := p.Parse(v)
		// if err != nil {
		// 	return nil, fmt.Errorf("unable to parse value for %q: %w", name, err)
		// }
		params = append(params, tools.ParamValue{Name: name, Value: v})
	}
	return params, nil
	// return tools.ParseParams(t.Parameters, data, claimsMap)
}

func (t MCPServerTool) Manifest() tools.Manifest {
	return t.manifest
}

func (t MCPServerTool) McpManifest() tools.McpManifest {
	return t.mcpManifest
}

func (t MCPServerTool) Authorized(verifiedAuthServices []string) bool {
	// TODO: Add Authorized feature
	return true
}

// http utilities

// CustomAuthTransport implements http.RoundTripper and adds the custom headers.
type CustomAuthTransport struct {
	// Embed http.RoundTripper to delegate the actual request execution.
	// This allows chaining with other transports or using the default.
	http.RoundTripper
	Headers map[string]string
}

func (t *CustomAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Update the http headers to support auth, do not need to clone the request
	for k, v := range t.Headers {
		req.Header.Add(k, v)
	}

	// Call the underlying RoundTripper to execute the request.
	return t.RoundTripper.RoundTrip(req)
}

func GetInputSchema(tool *mcp.Tool) tools.McpToolsSchema {
	return readMcpSchema(tool.InputSchema)
}

func GetOutputSchema(tool *mcp.Tool) tools.McpToolsSchema {
	return readMcpSchema(tool.OutputSchema)
}

func readMcpSchema(mcpToolSchema *jsonschema.Schema) tools.McpToolsSchema {
	// Required args is empty when there are none defined as required
	var requiredArgs = []string{}

	// Convert the MCP tool schema to the toolbox schema
	var properties = make(map[string]tools.ParameterMcpManifest, len(mcpToolSchema.Properties))
	for k, v := range mcpToolSchema.Properties {
		properties[k] = tools.ParameterMcpManifest{
			Type:        v.Type,
			Description: v.Description,
		}
	}
	if mcpToolSchema.Required != nil {
		requiredArgs = mcpToolSchema.Required
	}

	return tools.McpToolsSchema{
		Type:       mcpToolSchema.Type,
		Properties: properties,
		Required:   requiredArgs,
	}
}
