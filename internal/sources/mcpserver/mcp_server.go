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
	"log"
	"net/url"

	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/tools"
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
	SSE   TransportType = "sse" // Deprecated
	HTTP  TransportType = "http"
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
	actual := Config{Name: name, AuthMethod: None} // Default auth method
	if err := decoder.DecodeContext(ctx, &actual); err != nil {
		return nil, err
	}
	return actual, nil
}

type Config struct {
	Name        string        `yaml:"name" validate:"required"`
	Kind        string        `yaml:"kind" validate:"required"`
	Endpoint    string        `yaml:"endpoint" validate:"required"`
	SpecVersion SpecVersion   `yaml:"specVersion" validate:"required"`
	Transport   TransportType `yaml:"transport" validate:"required"`
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
		return nil, fmt.Errorf("failed to parse endpoint %v", err)
	}

	// Validate the spec version is supported
	if c.SpecVersion != NOV_2024 && c.SpecVersion != MAR_2025 && c.SpecVersion != JUN_2025 {
		return nil, fmt.Errorf("unsupported specVersion: %s", c.SpecVersion)
	}

	// Validate the transport type is supported
	// TODO: support stdio
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

	// TODO: Hook into ToolsListHandler option
	var client *mcp.Client = mcp.NewClient("TODO: Toolbox Client Name", "TODO: Toolbox Client version", &mcp.ClientOptions{})

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
	var transport mcp.Transport
	switch s.Transport {
	case SSE:
		transport = mcp.NewSSEClientTransport(s.Endpoint, &mcp.SSEClientTransportOptions{})
	case HTTP:
		transport = mcp.NewStreamableClientTransport(s.Endpoint, &mcp.StreamableClientTransportOptions{})
	default:
		transport = mcp.NewStdioTransport()
	}
	return s.Client.Connect(ctx, transport)
}

func (s *Source) GetTools(ctx context.Context) ([]tools.Tool, error) {
	fmt.Printf("Attempting to connect? to endpoint %s\n", s.Endpoint)
	session, err := s.getSession(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MCP server: %w", err)
	}
	defer session.Close()
	fmt.Println("Connecting?")

	remoteServerTools, err := session.ListTools(ctx, &mcp.ListToolsParams{})
	if err != nil {
		return nil, fmt.Errorf("failed to list/tools on MCP server of %s: %w", s.Name, err)
	}

	// Required args is empty when there are none defined as required
	var requiredArgs = []string{}
	var mcpTools []MCPServerTool = make([]MCPServerTool, len(remoteServerTools.Tools))
	for i, tool := range remoteServerTools.Tools {
		// TODO: Refactor or reuse the model jsonschema.Schema shape
		fmt.Println(tool.Name)
		fmt.Println(tool.InputSchema.Type)
		fmt.Println(tool.OutputSchema)

		var toolProperties = map[string]tools.ParameterMcpManifest{}
		for toolArgKey, toolArgumentValue := range tool.InputSchema.Properties {
			toolProperties[toolArgKey] = tools.ParameterMcpManifest{
				Type:        toolArgumentValue.Type,
				Description: toolArgumentValue.Description,
			}
		}

		if tool.InputSchema.Required != nil {
			requiredArgs = tool.InputSchema.Required
		}
		var toolCallParameters = make(tools.Parameters, len(toolProperties))

		mcpTools[i] = MCPServerTool{
			Source: s,
			Name:   tool.Name,
			manifest: tools.Manifest{
				Description: tool.Description,
				Parameters:  []tools.ParameterManifest{},
			},
			mcpManifest: tools.McpManifest{
				Name:        tool.Name,
				Description: tool.Description,
				InputSchema: tools.McpToolsSchema{
					Type:       tool.InputSchema.Type,
					Properties: toolProperties,
					Required:   requiredArgs,
				},
			},
			// Parameters: tool.InputSchema.ContentSchema,
			Parameters: toolCallParameters,
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

// remote-mcp-a:
//     kind: mcp-server
//     endpoint: https://mcp-a.example.com
//     version: "2024-11-05"
//     apiKey: "secret"

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
		log.Fatal(err)
	}
	defer session.Close()

	toolCallRequest := &mcp.CallToolParams{
		Name:      t.Name,
		Arguments: params.AsMap(),
	}
	res, err := session.CallTool(ctx, toolCallRequest)
	if err != nil {
		log.Fatalf("CallTool failed: %v", err)
	}
	if res.IsError {
		log.Fatal("tool failed")
	}

	var r = make([]any, 0, len(res.Content))
	for _, c := range res.Content {
		log.Print(c.(*mcp.TextContent).Text)
		r = append(r, res.Content)
	}

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
	// TODO:
	return true
}
