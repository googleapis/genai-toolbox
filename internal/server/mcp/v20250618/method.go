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

package v20250618

import (
	"strings"
	"github.com/googleapis/mcp-toolbox/internal/resources"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"time"

	"github.com/googleapis/mcp-toolbox/internal/auth"
	"github.com/googleapis/mcp-toolbox/internal/prompts"
	"github.com/googleapis/mcp-toolbox/internal/server/mcp/jsonrpc"
	mcputil "github.com/googleapis/mcp-toolbox/internal/server/mcp/util"
	"github.com/googleapis/mcp-toolbox/internal/server/primitives"
	"github.com/googleapis/mcp-toolbox/internal/tools"
	"github.com/googleapis/mcp-toolbox/internal/util"
	"github.com/googleapis/mcp-toolbox/internal/util/parameters"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// ProcessMethod returns a response for the request.
func ProcessMethod(ctx context.Context, id jsonrpc.RequestId, method string, toolset tools.Toolset, promptset prompts.Promptset, resourceset resources.ResourceSet, primitiveMgr *primitives.PrimitiveManager, body []byte, header http.Header) (any, error) {
	switch method {
	case INITIALIZE:
		return initializeHandler(ctx, id, body)
	case PING:
		return pingHandler(id)
	case TOOLS_LIST:
		return toolsListHandler(ctx, id, primitiveMgr, toolset, body)
	case TOOLS_CALL:
		return toolsCallHandler(ctx, id, toolset, primitiveMgr, body, header)
	case PROMPTS_LIST:
		return promptsListHandler(ctx, id, primitiveMgr, promptset, body)
	case PROMPTS_GET:
		return promptsGetHandler(ctx, id, promptset, primitiveMgr, body)
	case RESOURCES_LIST:
		return resourcesListHandler(ctx, id, primitiveMgr, resourceset, body)
	case RESOURCES_READ:
		return resourcesReadHandler(ctx, id, resourceset, primitiveMgr, body)
	default:
		err := fmt.Errorf("invalid method %s", method)
		return jsonrpc.NewError(id, jsonrpc.METHOD_NOT_FOUND, err.Error(), nil), err
	}
}

// InitializeResponse runs capability negotiation and protocol version agreement.
// This is the Initialization phase of the lifecycle for MCP client-server connections.
// Always start with the latest protocol version supported.
func initializeHandler(ctx context.Context, id jsonrpc.RequestId, body []byte) (any, error) {
	v, err := util.ToolboxVersionFromContext(ctx)
	if err != nil {
		return jsonrpc.NewError(id, jsonrpc.INTERNAL_ERROR, err.Error(), nil), err
	}

	var req InitializeRequest
	if err := json.Unmarshal(body, &req); err != nil {
		err = fmt.Errorf("invalid mcp initialize request: %w", err)
		return jsonrpc.NewError(id, jsonrpc.INVALID_REQUEST, err.Error(), nil), err
	}

	toolsListChanged := false
	promptsListChanged := false
	result := InitializeResult{
		ProtocolVersion: PROTOCOL_VERSION,
		Capabilities: ServerCapabilities{
			Tools: &ListChanged{
				ListChanged: &toolsListChanged,
			},
			Prompts: &ListChanged{
				ListChanged: &promptsListChanged,
			},
		},
		ServerInfo: Implementation{
			BaseMetadata: BaseMetadata{
				Name: SERVER_NAME,
			},
			Version: v,
		},
	}
	res := jsonrpc.JSONRPCResponse{
		Jsonrpc: jsonrpc.JSONRPC_VERSION,
		Id:      id,
		Result:  result,
	}

	return res, nil
}

// pingHandler handles the "ping" method by returning an empty response.
func pingHandler(id jsonrpc.RequestId) (any, error) {
	return jsonrpc.JSONRPCResponse{
		Jsonrpc: jsonrpc.JSONRPC_VERSION,
		Id:      id,
		Result:  struct{}{},
	}, nil
}

func toolsListHandler(ctx context.Context, id jsonrpc.RequestId, primitiveMgr *primitives.PrimitiveManager, toolset tools.Toolset, body []byte) (any, error) {
	var req ListToolsRequest
	if err := json.Unmarshal(body, &req); err != nil {
		err = fmt.Errorf("invalid mcp tools list request: %w", err)
		return jsonrpc.NewError(id, jsonrpc.INVALID_REQUEST, err.Error(), nil), err
	}

	urlParams, _ := util.UrlParamsFromContext(ctx)
	toolsMap := primitiveMgr.GetToolsMap()
	listToolsResult, err := GenerateListToolsResult(primitiveMgr.GetSourcesMap(), toolset, toolsMap, urlParams)
	if err != nil {
		err = fmt.Errorf("error generating manifest: %w", err)
		return jsonrpc.NewError(id, jsonrpc.INTERNAL_ERROR, err.Error(), nil), err
	}
	return jsonrpc.JSONRPCResponse{
		Jsonrpc: jsonrpc.JSONRPC_VERSION,
		Id:      id,
		Result:  listToolsResult,
	}, nil
}

// toolsCallHandler generate a response for tools call.
func toolsCallHandler(ctx context.Context, id jsonrpc.RequestId, toolset tools.Toolset, primitiveMgr *primitives.PrimitiveManager, body []byte, header http.Header) (any, error) {
	if header != nil {
		if clientIP := util.ExtractClientIP(header); clientIP != "" {
			ctx = util.WithClientIP(ctx, clientIP)
		}
	}

	authServices := primitiveMgr.GetAuthServiceMap()

	// retrieve logger from context
	logger, err := util.LoggerFromContext(ctx)
	if err != nil {
		return jsonrpc.NewError(id, jsonrpc.INTERNAL_ERROR, err.Error(), nil), err
	}

	var req CallToolRequest
	if err = json.Unmarshal(body, &req); err != nil {
		err = fmt.Errorf("invalid mcp tools call request: %w", err)
		return jsonrpc.NewError(id, jsonrpc.INVALID_REQUEST, err.Error(), nil), err
	}

	toolName := req.Params.Name
	toolArgument := req.Params.Arguments
	logger.DebugContext(ctx, fmt.Sprintf("tool name: %s", toolName))

	// Update span name and set gen_ai attributes
	span := trace.SpanFromContext(ctx)
	span.SetName(fmt.Sprintf("%s %s", TOOLS_CALL, toolName))
	span.SetAttributes(
		attribute.String("gen_ai.tool.name", toolName),
		attribute.String("gen_ai.operation.name", "execute_tool"),
	)

	// Verify tool belongs to the current toolset before resolving globally.
	if !toolset.ContainsTool(toolName) {
		err = fmt.Errorf("invalid tool name: tool with name %q does not exist", toolName)
		return jsonrpc.NewError(id, jsonrpc.INVALID_PARAMS, err.Error(), nil), err
	}

	tool, ok := primitiveMgr.GetTool(toolName)
	if !ok {
		err = fmt.Errorf("invalid tool name: tool with name %q does not exist", toolName)
		return jsonrpc.NewError(id, jsonrpc.INVALID_PARAMS, err.Error(), nil), err
	}

	// Populate gen_ai attributes for operation duration metric
	if genAIAttrs := util.GenAIMetricAttrsFromContext(ctx); genAIAttrs != nil {
		genAIAttrs.OperationName = "execute_tool"
		genAIAttrs.ToolName = toolName
	}

	// Get access token
	authTokenHeadername, err := tool.GetAuthTokenHeaderName(primitiveMgr)
	if err != nil {
		errMsg := fmt.Errorf("error during invocation: %w", err)
		return jsonrpc.NewError(id, jsonrpc.INTERNAL_ERROR, errMsg.Error(), nil), errMsg
	}
	accessToken := tools.AccessToken(header.Get(authTokenHeadername))

	// Check if this specific tool requires the standard authorization header
	clientAuth, err := tool.RequiresClientAuthorization(primitiveMgr)
	if err != nil {
		errMsg := fmt.Errorf("error during invocation: %w", err)
		return jsonrpc.NewError(id, jsonrpc.INTERNAL_ERROR, errMsg.Error(), nil), errMsg
	}
	if clientAuth {
		if accessToken == "" {
			err := util.NewClientServerError(
				"missing access token in the 'Authorization' header",
				http.StatusUnauthorized,
				nil,
			)
			return jsonrpc.NewError(id, jsonrpc.INVALID_REQUEST, err.Error(), nil), err
		}
	}

	// marshal arguments and decode it using decodeJSON instead to prevent loss between floats/int.
	aMarshal, err := json.Marshal(toolArgument)
	if err != nil {
		err = fmt.Errorf("unable to marshal tools argument: %w", err)
		return jsonrpc.NewError(id, jsonrpc.INTERNAL_ERROR, err.Error(), nil), err
	}

	var data map[string]any
	if err = util.DecodeJSON(bytes.NewBuffer(aMarshal), &data); err != nil {
		err = fmt.Errorf("unable to decode tools argument: %w", err)
		return jsonrpc.NewError(id, jsonrpc.INTERNAL_ERROR, err.Error(), nil), err
	}

	// Tool authentication
	// claimsFromAuth maps the name of the authservice to the claims retrieved from it.
	claimsFromAuth := make(map[string]map[string]any)

	// if using stdio, header will be nil and auth will not be supported
	if header != nil {
		for _, aS := range authServices {
			var claims map[string]any
			var err error

			if mSvc, ok := aS.(auth.MCPAuthService); ok && mSvc.IsMCPEnabled() {
				claims = util.AuthTokenClaimsFromContext(ctx)
			} else {
				claims, err = aS.GetClaimsFromHeader(ctx, header)
				if err != nil {
					logger.DebugContext(ctx, err.Error())
					continue
				}
			}

			if claims == nil {
				// authService not present in header
				continue
			}
			claimsFromAuth[aS.GetName()] = claims
		}
	}

	// Tool authorization check
	verifiedAuthServices := make([]string, len(claimsFromAuth))
	i := 0
	for k := range claimsFromAuth {
		verifiedAuthServices[i] = k
		i++
	}

	// Check if any of the specified auth services is verified
	isAuthorized := tool.Authorized(verifiedAuthServices)
	if !isAuthorized {
		err = util.NewClientServerError(
			"unauthorized Tool call: Please make sure you specify correct auth headers",
			http.StatusUnauthorized,
			nil,
		)
		return jsonrpc.NewError(id, jsonrpc.INVALID_REQUEST, err.Error(), nil), err
	}
	logger.DebugContext(ctx, "tool invocation authorized")

	if err := mcputil.ValidateScopes(ctx, tool.GetScopesRequired(), authServices); err != nil {
		return jsonrpc.NewError(id, jsonrpc.INVALID_REQUEST, err.Error(), nil), err
	}

	toolParams, err := tool.GetParameters(primitiveMgr.GetSourcesMap())
	if err != nil {
		err = fmt.Errorf("error getting parameters for tool: %w", err)
		return jsonrpc.NewError(id, jsonrpc.INTERNAL_ERROR, err.Error(), nil), err
	}

	// Auto-populate arguments from URL parameters
	data = mcputil.PopulateUrlParams(ctx, data, toolParams)

	params, err := parameters.ParseParams(toolParams, data, claimsFromAuth)
	if err != nil {
		err = fmt.Errorf("provided parameters were invalid: %w", err)
		return jsonrpc.NewError(id, jsonrpc.INVALID_PARAMS, err.Error(), nil), err
	}
	logger.DebugContext(ctx, fmt.Sprintf("invocation params: %s", params))

	embeddingModels := primitiveMgr.GetEmbeddingModelMap()
	params, err = tool.EmbedParams(ctx, params, embeddingModels)
	if err != nil {
		err = fmt.Errorf("error embedding parameters: %w", err)
		return jsonrpc.NewError(id, jsonrpc.INVALID_PARAMS, err.Error(), nil), err
	}

	// Get instrumentation for recording tool execution duration
	instrumentation, instrumentationErr := util.InstrumentationFromContext(ctx)

	// run tool invocation and generate response.
	executionStart := time.Now()
	results, err := tool.Invoke(ctx, primitiveMgr, params, accessToken)
	executionDuration := time.Since(executionStart).Seconds()

	// Record tool execution duration metric
	if instrumentationErr == nil {
		execAttrs := []attribute.KeyValue{
			attribute.String("gen_ai.tool.name", toolName),
		}
		// Add network protocol attributes from context
		if genAIAttrs := util.GenAIMetricAttrsFromContext(ctx); genAIAttrs != nil {
			if genAIAttrs.NetworkProtocolName != "" {
				execAttrs = append(execAttrs, attribute.String("network.protocol.name", genAIAttrs.NetworkProtocolName))
			}
			if genAIAttrs.NetworkProtocolVersion != "" {
				execAttrs = append(execAttrs, attribute.String("network.protocol.version", genAIAttrs.NetworkProtocolVersion))
			}
		}
		if err != nil {
			execAttrs = append(execAttrs, attribute.String("error.type", err.Error()))
		}
		instrumentation.ToolExecutionDuration.Record(ctx, executionDuration, metric.WithAttributes(execAttrs...))
	}

	if err != nil {
		var tbErr util.ToolboxError

		if errors.As(err, &tbErr) {
			switch tbErr.Category() {
			case util.CategoryAgent:
				// MCP - Tool execution error
				// Return SUCCESS but with IsError: true
				text := TextContent{
					Type: "text",
					Text: err.Error(),
				}
				return jsonrpc.JSONRPCResponse{
					Jsonrpc: jsonrpc.JSONRPC_VERSION,
					Id:      id,
					Result:  CallToolResult{Content: []TextContent{text}, IsError: true},
				}, nil

			case util.CategoryServer:
				// MCP Spec - Protocol error
				// Return JSON-RPC ERROR
				var clientServerErr *util.ClientServerError
				rpcCode := jsonrpc.INTERNAL_ERROR // Default to Internal Error (-32603)

				if errors.As(err, &clientServerErr) {
					if clientServerErr.Code == http.StatusUnauthorized || clientServerErr.Code == http.StatusForbidden {
						if clientAuth {
							rpcCode = jsonrpc.INVALID_REQUEST
						} else {
							rpcCode = jsonrpc.INTERNAL_ERROR
						}
					}
				}
				return jsonrpc.NewError(id, rpcCode, err.Error(), nil), err
			}
		} else {
			// Unknown error -> 500
			return jsonrpc.NewError(id, jsonrpc.INTERNAL_ERROR, err.Error(), nil), err
		}
	}

	content := make([]TextContent, 0)

	sliceRes, ok := results.([]any)
	if !ok {
		sliceRes = []any{results}
	}

	for _, d := range sliceRes {
		text := TextContent{Type: "text"}
		dM, err := json.Marshal(d)
		if err != nil {
			text.Text = fmt.Sprintf("fail to marshal: %s, result: %s", err, d)
		} else {
			text.Text = string(dM)
		}
		content = append(content, text)
	}

	return jsonrpc.JSONRPCResponse{
		Jsonrpc: jsonrpc.JSONRPC_VERSION,
		Id:      id,
		Result:  CallToolResult{Content: content},
	}, nil
}

// promptsListHandler handles the "prompts/list" method.
func promptsListHandler(ctx context.Context, id jsonrpc.RequestId, primitiveMgr *primitives.PrimitiveManager, promptset prompts.Promptset, body []byte) (any, error) {
	// retrieve logger from context
	logger, err := util.LoggerFromContext(ctx)
	if err != nil {
		return jsonrpc.NewError(id, jsonrpc.INTERNAL_ERROR, err.Error(), nil), err
	}
	logger.DebugContext(ctx, "handling prompts/list request")

	var req ListPromptsRequest
	if err := json.Unmarshal(body, &req); err != nil {
		err = fmt.Errorf("invalid mcp prompts list request: %w", err)
		return jsonrpc.NewError(id, jsonrpc.INVALID_REQUEST, err.Error(), nil), err
	}

	promptsMap := primitiveMgr.GetPromptsMap()
	listPromptsResult, err := GenerateListPromptsResult(promptset, promptsMap)
	if err != nil {
		err = fmt.Errorf("error generating manifest: %w", err)
		return jsonrpc.NewError(id, jsonrpc.INTERNAL_ERROR, err.Error(), nil), err
	}
	logger.DebugContext(ctx, fmt.Sprintf("returning %d prompts", len(listPromptsResult.Prompts)))
	return jsonrpc.JSONRPCResponse{
		Jsonrpc: jsonrpc.JSONRPC_VERSION,
		Id:      id,
		Result:  listPromptsResult,
	}, nil
}

// promptsGetHandler handles the "prompts/get" method.
func promptsGetHandler(ctx context.Context, id jsonrpc.RequestId, promptset prompts.Promptset, primitiveMgr *primitives.PrimitiveManager, body []byte) (any, error) {
	// retrieve logger from context
	logger, err := util.LoggerFromContext(ctx)
	if err != nil {
		return jsonrpc.NewError(id, jsonrpc.INTERNAL_ERROR, err.Error(), nil), err
	}
	logger.DebugContext(ctx, "handling prompts/get request")

	var req GetPromptRequest
	if err := json.Unmarshal(body, &req); err != nil {
		err = fmt.Errorf("invalid mcp prompts/get request: %w", err)
		return jsonrpc.NewError(id, jsonrpc.INVALID_REQUEST, err.Error(), nil), err
	}

	promptName := req.Params.Name
	logger.DebugContext(ctx, fmt.Sprintf("prompt name: %s", promptName))

	// Update span name and set gen_ai attributes
	span := trace.SpanFromContext(ctx)
	span.SetName(fmt.Sprintf("%s %s", PROMPTS_GET, promptName))
	span.SetAttributes(attribute.String("gen_ai.prompt.name", promptName))

	// Verify prompt belongs to the current promptset before resolving globally.
	if !promptset.ContainsPrompt(promptName) {
		err := fmt.Errorf("prompt with name %q does not exist", promptName)
		return jsonrpc.NewError(id, jsonrpc.INVALID_PARAMS, err.Error(), nil), err
	}

	prompt, ok := primitiveMgr.GetPrompt(promptName)
	if !ok {
		err := fmt.Errorf("prompt with name %q does not exist", promptName)
		return jsonrpc.NewError(id, jsonrpc.INVALID_PARAMS, err.Error(), nil), err
	}

	// Populate gen_ai attributes for operation duration metric
	if genAIAttrs := util.GenAIMetricAttrsFromContext(ctx); genAIAttrs != nil {
		genAIAttrs.OperationName = "get_prompt"
		genAIAttrs.PromptName = promptName
	}

	// Parse the arguments provided in the request.
	argValues, err := prompt.ParseArgs(req.Params.Arguments, nil)
	if err != nil {
		err = fmt.Errorf("invalid arguments for prompt %q: %w", promptName, err)
		return jsonrpc.NewError(id, jsonrpc.INVALID_PARAMS, err.Error(), nil), err
	}
	logger.DebugContext(ctx, fmt.Sprintf("parsed args: %v", argValues))

	// Substitute the argument values into the prompt's messages.
	substituted, err := prompt.SubstituteParams(argValues)
	if err != nil {
		err = fmt.Errorf("error substituting params for prompt %q: %w", promptName, err)
		return jsonrpc.NewError(id, jsonrpc.INTERNAL_ERROR, err.Error(), nil), err
	}

	// Cast the result to the expected []prompts.Message type.
	substitutedMessages, ok := substituted.([]prompts.Message)
	if !ok {
		err = fmt.Errorf("internal error: SubstituteParams returned unexpected type")
		return jsonrpc.NewError(id, jsonrpc.INTERNAL_ERROR, err.Error(), nil), err
	}
	logger.DebugContext(ctx, "substituted params successfully")

	// Format the response messages into the required structure.
	promptMessages := make([]PromptMessage, len(substitutedMessages))
	for i, msg := range substitutedMessages {
		promptMessages[i] = PromptMessage{
			Role: msg.Role,
			Content: TextContent{
				Type: "text",
				Text: msg.Content,
			},
		}
	}

	result := GetPromptResult{
		Description: prompt.Manifest().Description,
		Messages:    promptMessages,
	}

	return jsonrpc.JSONRPCResponse{
		Jsonrpc: jsonrpc.JSONRPC_VERSION,
		Id:      id,
		Result:  result,
	}, nil
}


/* Resources Handlers */

func resourcesListHandler(ctx context.Context, id jsonrpc.RequestId, primitiveMgr *primitives.PrimitiveManager, resourceset resources.ResourceSet, body []byte) (any, error) {
	allRes := primitiveMgr.ListResources()
	allTemps := primitiveMgr.ListResourceTemplates()

	var resList []Resource
	for _, r := range allRes {
		if resourceset.ContainsResource(r.ResourceURI()) {
			resList = append(resList, Resource{
				URI:         r.ResourceURI(),
				Name:        r.ResourceName(),
				Description: r.ResourceDescription(),
				MimeType:    r.ResourceMimeType(),
				Annotations: translateAnnotations(r.ResourceAnnotations()),
			})
		}
	}

	var tempList []ResourceTemplate
	for _, r := range allTemps {
		if resourceset.ContainsResource(r.ResourceURI()) {
			tempList = append(tempList, ResourceTemplate{
				URITemplate: r.ResourceURI(),
				Name:        r.ResourceName(),
				Description: r.ResourceDescription(),
				MimeType:    r.ResourceMimeType(),
				Annotations: translateAnnotations(r.ResourceAnnotations()),
			})
		}
	}

	return jsonrpc.JSONRPCResponse{
		Jsonrpc: jsonrpc.JSONRPC_VERSION,
		Id:      id,
		Result: ResourcesListResult{
			Resources:         resList,
			ResourceTemplates: tempList,
		},
	}, nil
}

func resourcesReadHandler(ctx context.Context, id jsonrpc.RequestId, resourceset resources.ResourceSet, primitiveMgr *primitives.PrimitiveManager, body []byte) (any, error) {
	var req ResourcesReadRequest
	if err := json.Unmarshal(body, &req); err != nil {
		err = fmt.Errorf("invalid resources/read request: %w", err)
		return jsonrpc.NewError(id, jsonrpc.INVALID_REQUEST, err.Error(), nil), err
	}

	uri := req.Params.URI
	if uri == "" {
		err := fmt.Errorf("missing 'uri' parameter")
		return jsonrpc.NewError(id, jsonrpc.INVALID_PARAMS, err.Error(), nil), err
	}

	// Resolve resource
	res, found := primitiveMgr.ResolveResource(uri)
	if !found {
		err := fmt.Errorf("resource not found: %q", uri)
		return jsonrpc.NewError(id, jsonrpc.INVALID_REQUEST, err.Error(), nil), err
	}

	// Verify resourceset membership
	if !resourceset.ContainsResource(res.ResourceURI()) {
		err := fmt.Errorf("permission denied: resource %q is not in the active resourceset", uri)
		return jsonrpc.NewError(id, jsonrpc.INVALID_REQUEST, err.Error(), nil), err
	}

	// Extract template parameters
	params := extractTemplateParams(res.ResourceURI(), uri)

	// Read resource content
	contents, err := res.Read(ctx, params)
	if err != nil {
		return jsonrpc.NewError(id, jsonrpc.INTERNAL_ERROR, err.Error(), nil), err
	}

	var rpcContents []ResourceContent
	for _, c := range contents {
		rpcContents = append(rpcContents, ResourceContent{
			URI:      c.URI,
			MimeType: c.MimeType,
			Text:     c.Text,
		})
	}

	return jsonrpc.JSONRPCResponse{
		Jsonrpc: jsonrpc.JSONRPC_VERSION,
		Id:      id,
		Result: ResourcesReadResult{
			Contents: rpcContents,
		},
	}, nil
}

func translateAnnotations(ann *resources.Annotations) *ResourceAnnotations {
	if ann == nil {
		return nil
	}
	prio := 1.0
	if ann.Priority != nil {
		prio = *ann.Priority
	}
	return &ResourceAnnotations{
		Audience: ann.Audience,
		Priority: prio,
	}
}

func extractTemplateParams(templateURI, requestedURI string) map[string]any {
	params := make(map[string]any)
	if strings.Contains(templateURI, "{path}") {
		prefix := strings.Split(templateURI, "{path}")[0]
		if strings.HasPrefix(requestedURI, prefix) {
			val := strings.TrimPrefix(requestedURI, prefix)
			params["path"] = val
		}
	}
	return params
}
