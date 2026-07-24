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

package vdraft

import (
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
func ProcessMethod(ctx context.Context, id jsonrpc.RequestId, method string, toolset tools.Toolset, promptset prompts.Promptset, primitiveMgr *primitives.PrimitiveManager, body []byte, header http.Header) (any, error) {
	switch method {
	case SERVER_DISCOVER:
		return serverDiscoverHandler(ctx, id, body, header)
	case TOOLS_LIST:
		return toolsListHandler(ctx, id, primitiveMgr, toolset, body, header)
	case TOOLS_CALL:
		return toolsCallHandler(ctx, id, toolset, primitiveMgr, body, header)
	case PROMPTS_LIST:
		return promptsListHandler(ctx, id, primitiveMgr, promptset, body, header)
	case PROMPTS_GET:
		return promptsGetHandler(ctx, id, promptset, primitiveMgr, body, header)
	default:
		err := fmt.Errorf("invalid method %s", method)
		return jsonrpc.NewError(id, jsonrpc.METHOD_NOT_FOUND, err.Error(), nil), err
	}
}

// validateMetadata checks the metadata of every requests
func validateMetadata(id jsonrpc.RequestId, params RequestParams, stdio bool) (any, error) {
	// Check _meta field for protocol version
	if params.Meta != nil {
		// check for protocol version metadata
		v := params.Meta.ProtocolVersion
		if v == "" {
			metaErr := fmt.Errorf("_meta error: missing io.modelcontextprotocol/protocolVersion")
			return jsonrpc.NewError(id, jsonrpc.INVALID_PARAMS, metaErr.Error(), nil), metaErr
		}
		// do not have to verify header for stdio; already verified during stdio
		// message processing
		if !stdio {
			if PROTOCOL_VERSION != v {
				metaErr := fmt.Errorf("_meta error: header mismatch: MCP-Protocol-Version header value '%s' does not match body value '%s'", PROTOCOL_VERSION, v)
				return jsonrpc.NewHeaderMismatchedError(id, metaErr), metaErr
			}
		}

		// check for clientInfo
		clientInfo := params.Meta.ClientInfo
		if clientInfo.Version == "" || clientInfo.Name == "" {
			metaErr := fmt.Errorf("_meta error: missing field from io.modelcontextprotocol/clientInfo")
			return jsonrpc.NewError(id, jsonrpc.INVALID_PARAMS, metaErr.Error(), nil), metaErr
		}
		// check for clientCapabilities
		clientCapabilities := params.Meta.MetaClientCapabilities
		if clientCapabilities == nil {
			metaErr := fmt.Errorf("_meta error: missing field from io.modelcontextprotocol/clientCapabilities")
			return jsonrpc.NewError(id, jsonrpc.INVALID_PARAMS, metaErr.Error(), nil), metaErr
		}
		// skip checking clientCapabilities since Toolbox do not utilize any of those
	} else {
		metaErr := fmt.Errorf("_meta error: missing required fields in request metadata")
		return jsonrpc.NewError(id, jsonrpc.INVALID_PARAMS, metaErr.Error(), nil), metaErr
	}
	return nil, nil
}

// validateHeader checks the header of every requests
// Toolbox do not check for `Mcp-Param-{Name}` header since we are not
// implementing custom headers from parameters
// Do not need to check for Base64-encoding or invalid characters since we are
// only checking `mcp-method` and `mcp-name`
func validateHeader(id jsonrpc.RequestId, header http.Header, method, name string) (any, error) {
	// stdio transport will not have header
	if header == nil {
		return nil, nil
	}
	headerMethod := header.Get("mcp-method")
	if headerMethod != method {
		err := fmt.Errorf("Mcp-Method header value '%s' does not match body value '%s'", headerMethod, method)
		return jsonrpc.NewHeaderMismatchedError(id, err), err
	}
	headerName := header.Get("mcp-name")
	if headerName != name {
		err := fmt.Errorf("Mcp-Name header value '%s' does not match body value '%s'", headerName, name)
		return jsonrpc.NewHeaderMismatchedError(id, err), err
	}
	return nil, nil
}

// getResultMetadata append the resultMetaObject on existing metadata
func getResultMetadata(ctx context.Context, curMeta map[string]any) (map[string]any, error) {
	v, err := util.ToolboxVersionFromContext(ctx)
	if err != nil {
		return nil, err
	}

	resMetaObj := ResultMetaObject{
		ServerInfo: Implementation{
			BaseMetadata: BaseMetadata{
				Name: SERVER_NAME,
			},
			Version: v,
		},
	}
	jsonData, err := json.Marshal(resMetaObj)
	if err != nil {
		return nil, err
	}
	resMeta := make(map[string]any)
	if err := json.Unmarshal(jsonData, &resMeta); err != nil {
		return nil, err
	}
	// if there are no existing metadata field, we can just return the
	// ResultMetaObject
	if curMeta == nil {
		return resMeta, nil
	}

	newMeta := make(map[string]any)
	// copy current Metadata into the new meta obj
	for k, v := range curMeta {
		newMeta[k] = v
	}
	// copy the ResultMetaObject items over
	for k, v := range resMeta {
		newMeta[k] = v
	}
	return newMeta, nil
}

func serverDiscoverHandler(ctx context.Context, id jsonrpc.RequestId, body []byte, header http.Header) (any, error) {
	enableDraft, ok := util.EnableDraftSpecsFromContext(ctx)
	if !ok {
		err := fmt.Errorf("unable to retrieve enableDraftSpecs from context")
		return jsonrpc.NewError(id, jsonrpc.INTERNAL_ERROR, err.Error(), nil), err
	}

	var req DiscoverRequest
	if err := json.Unmarshal(body, &req); err != nil {
		err = fmt.Errorf("invalid server discover request: %w", err)
		return jsonrpc.NewError(id, jsonrpc.INVALID_REQUEST, err.Error(), nil), err
	}
	validateHeaderErr, err := validateHeader(id, header, SERVER_DISCOVER, "")
	if err != nil {
		return validateHeaderErr, err
	}
	validateErr, err := validateMetadata(id, req.Params, header == nil)
	if err != nil {
		return validateErr, err
	}

	toolsListChanged := false
	promptsListChanged := false
	meta, err := getResultMetadata(ctx, nil)
	if err != nil {
		return jsonrpc.NewError(id, jsonrpc.INTERNAL_ERROR, err.Error(), nil), err
	}
	result := DiscoverResult{
		Result: Result{
			ResultType: resultTypeComplete,
			Result: jsonrpc.Result{
				Meta: meta,
			},
		},
		SupportedVersions: mcputil.GetSupportedVersions(enableDraft),
		Capabilities: ServerCapabilities{
			Tools: &ListChanged{
				ListChanged: &toolsListChanged,
			},
			Prompts: &ListChanged{
				ListChanged: &promptsListChanged,
			},
		},
	}
	res := jsonrpc.JSONRPCResponse{
		Jsonrpc: jsonrpc.JSONRPC_VERSION,
		Id:      id,
		Result:  result,
	}
	return res, nil
}

func toolsListHandler(ctx context.Context, id jsonrpc.RequestId, primitiveMgr *primitives.PrimitiveManager, toolset tools.Toolset, body []byte, header http.Header) (any, error) {
	var req ListToolsRequest
	if err := json.Unmarshal(body, &req); err != nil {
		err = fmt.Errorf("invalid mcp tools list request: %w", err)
		return jsonrpc.NewError(id, jsonrpc.INVALID_REQUEST, err.Error(), nil), err
	}
	validateHeaderErr, err := validateHeader(id, header, TOOLS_LIST, "")
	if err != nil {
		return validateHeaderErr, err
	}
	validateErr, err := validateMetadata(id, req.Params.RequestParams, header == nil)
	if err != nil {
		return validateErr, err
	}

	urlParams, _ := util.UrlParamsFromContext(ctx)
	toolsMap := primitiveMgr.GetToolsMap()
	listToolsResult, err := GenerateListToolsResult(primitiveMgr.GetSourcesMap(), toolset, toolsMap, urlParams)
	if err != nil {
		err = fmt.Errorf("error generating manifest: %w", err)
		return jsonrpc.NewError(id, jsonrpc.INTERNAL_ERROR, err.Error(), nil), err
	}
	meta, err := getResultMetadata(ctx, listToolsResult.Meta)
	if err != nil {
		return jsonrpc.NewError(id, jsonrpc.INTERNAL_ERROR, err.Error(), nil), err
	}
	listToolsResult.Meta = meta
	return jsonrpc.JSONRPCResponse{
		Jsonrpc: jsonrpc.JSONRPC_VERSION,
		Id:      id,
		Result:  listToolsResult,
	}, nil
}

// toolsCallHandler generate a response for tools call.
func toolsCallHandler(ctx context.Context, id jsonrpc.RequestId, toolset tools.Toolset, primitiveMgr *primitives.PrimitiveManager, body []byte, header http.Header) (any, error) {
	authServices := primitiveMgr.GetAuthServiceMap()

	// retrieve logger from context
	logger, err := util.LoggerFromContext(ctx)
	if err != nil {
		return jsonrpc.NewError(id, jsonrpc.INTERNAL_ERROR, err.Error(), nil), err
	}
	meta, err := getResultMetadata(ctx, nil)
	if err != nil {
		return jsonrpc.NewError(id, jsonrpc.INTERNAL_ERROR, err.Error(), nil), err
	}

	var req CallToolRequest
	if err = json.Unmarshal(body, &req); err != nil {
		err = fmt.Errorf("invalid mcp tools call request: %w", err)
		return jsonrpc.NewError(id, jsonrpc.INVALID_REQUEST, err.Error(), nil), err
	}
	validateHeaderErr, err := validateHeader(id, header, TOOLS_CALL, req.Params.Name)
	if err != nil {
		return validateHeaderErr, err
	}
	validateErr, err := validateMetadata(id, req.Params.RequestParams, header == nil)
	if err != nil {
		return validateErr, err
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
	var accessToken tools.AccessToken
	if header != nil {
		accessToken = tools.AccessToken(header.Get(authTokenHeadername))
	}

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
	var data map[string]any
	if toolArgument != nil {
		aMarshal, err := json.Marshal(toolArgument)
		if err != nil {
			err = fmt.Errorf("unable to marshal tools argument: %w", err)
			return jsonrpc.NewError(id, jsonrpc.INTERNAL_ERROR, err.Error(), nil), err
		}

		if err = util.DecodeJSON(bytes.NewBuffer(aMarshal), &data); err != nil {
			err = fmt.Errorf("unable to decode tools argument: %w", err)
			return jsonrpc.NewError(id, jsonrpc.INTERNAL_ERROR, err.Error(), nil), err
		}
	}
	if data == nil {
		data = make(map[string]any)
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
					Result: CallToolResult{
						Result: Result{
							ResultType: resultTypeComplete,
							Result: jsonrpc.Result{
								Meta: meta,
							},
						},
						Content: []TextContent{text},
						IsError: true,
					},
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
		Result: CallToolResult{
			Result: Result{
				ResultType: resultTypeComplete,
				Result: jsonrpc.Result{
					Meta: meta,
				},
			},
			Content: content,
		},
	}, nil
}

// promptsListHandler handles the "prompts/list" method.
func promptsListHandler(ctx context.Context, id jsonrpc.RequestId, primitiveMgr *primitives.PrimitiveManager, promptset prompts.Promptset, body []byte, header http.Header) (any, error) {
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
	validateHeaderErr, err := validateHeader(id, header, PROMPTS_LIST, "")
	if err != nil {
		return validateHeaderErr, err
	}
	validateErr, err := validateMetadata(id, req.Params.RequestParams, header == nil)
	if err != nil {
		return validateErr, err
	}

	promptsMap := primitiveMgr.GetPromptsMap()
	listPromptsResult, err := GenerateListPromptsResult(promptset, promptsMap)
	if err != nil {
		err = fmt.Errorf("error generating manifest: %w", err)
		return jsonrpc.NewError(id, jsonrpc.INTERNAL_ERROR, err.Error(), nil), err
	}
	logger.DebugContext(ctx, fmt.Sprintf("returning %d prompts", len(listPromptsResult.Prompts)))
	meta, err := getResultMetadata(ctx, listPromptsResult.Meta)
	if err != nil {
		return jsonrpc.NewError(id, jsonrpc.INTERNAL_ERROR, err.Error(), nil), err
	}
	listPromptsResult.Meta = meta
	return jsonrpc.JSONRPCResponse{
		Jsonrpc: jsonrpc.JSONRPC_VERSION,
		Id:      id,
		Result:  listPromptsResult,
	}, nil
}

// promptsGetHandler handles the "prompts/get" method.
func promptsGetHandler(ctx context.Context, id jsonrpc.RequestId, promptset prompts.Promptset, primitiveMgr *primitives.PrimitiveManager, body []byte, header http.Header) (any, error) {
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
	validateHeaderErr, err := validateHeader(id, header, PROMPTS_GET, req.Params.Name)
	if err != nil {
		return validateHeaderErr, err
	}
	validateErr, err := validateMetadata(id, req.Params.RequestParams, header == nil)
	if err != nil {
		return validateErr, err
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
	meta, err := getResultMetadata(ctx, nil)
	if err != nil {
		return jsonrpc.NewError(id, jsonrpc.INTERNAL_ERROR, err.Error(), nil), err
	}
	result := GetPromptResult{
		Result: Result{
			ResultType: resultTypeComplete,
			Result: jsonrpc.Result{
				Meta: meta,
			},
		},
		Description: prompt.Manifest().Description,
		Messages:    promptMessages,
	}
	return jsonrpc.JSONRPCResponse{
		Jsonrpc: jsonrpc.JSONRPC_VERSION,
		Id:      id,
		Result:  result,
	}, nil
}
