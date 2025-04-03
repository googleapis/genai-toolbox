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
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/google/uuid"
	"github.com/googleapis/genai-toolbox/internal/server/mcp"
)

// mcpRouter creates a router that represents the routes under /mcp
func mcpRouter(s *Server) (chi.Router, error) {
	r := chi.NewRouter()

	r.Use(middleware.AllowContentType("application/json"))
	r.Use(middleware.StripSlashes)
	r.Use(render.SetContentType(render.ContentTypeJSON))

	r.Post("/", func(w http.ResponseWriter, r *http.Request) { mcpHandler(s, w, r) })

	return r, nil
}

// mcpHandler handles all mcp messages.
func mcpHandler(s *Server, w http.ResponseWriter, r *http.Request) {
	// Read and returns a body from io.Reader
	body, err := io.ReadAll(r.Body)
	if err != nil {
		// Generate a new uuid if unable to decode
		id := uuid.New().String()
		render.JSON(w, r, newJSONRPCError(id, mcp.PARSE_ERROR, err.Error(), nil))
	}

	// Generic baseMessage could either be a JSONRPCNotification or JSONRPCRequest
	var baseMessage struct {
		Jsonrpc string        `json:"jsonrpc"`
		Method  string        `json:"method"`
		Id      mcp.RequestId `json:"id,omitempty"`
	}
	if err := decodeJSON(bytes.NewBuffer(body), &baseMessage); err != nil {
		// Generate a new uuid if unable to decode
		id := uuid.New().String()
		render.JSON(w, r, newJSONRPCError(id, mcp.PARSE_ERROR, err.Error(), nil))
		return
	}

	// Check if method is present
	if baseMessage.Method == "" {
		err := fmt.Errorf("method not found")
		render.JSON(w, r, newJSONRPCError(baseMessage.Id, mcp.METHOD_NOT_FOUND, err.Error(), nil))
		return
	}

	// Check for JSON-RPC 2.0
	if baseMessage.Jsonrpc != mcp.JSONRPC_VERSION {
		err := fmt.Errorf("invalid json-rpc version")
		render.JSON(w, r, newJSONRPCError(baseMessage.Id, mcp.INVALID_REQUEST, err.Error(), nil))
		return
	}

	// Check if message is a notification
	if baseMessage.Id == nil {
		var notification mcp.JSONRPCNotification
		if err := json.Unmarshal(body, &notification); err != nil {
			err := fmt.Errorf("invalid notification request: %w", err)
			render.JSON(w, r, newJSONRPCError(baseMessage.Id, mcp.PARSE_ERROR, err.Error(), nil))
		}
		// Notifications do not expect a response
		// Toolbox doesn't do anything with notifications yet
		return
	}

	var res mcp.JSONRPCMessage
	switch baseMessage.Method {
	case "initialize":
		var req mcp.InitializeRequest
		if err := json.Unmarshal(body, &req); err != nil {
			err := fmt.Errorf("invalid mcp initialize request: %w", err)
			res = newJSONRPCError(baseMessage.Id, mcp.INVALID_REQUEST, err.Error(), nil)
			break
		}
		result := mcp.Initialize(s.version)
		res = mcp.JSONRPCResponse{
			Jsonrpc: mcp.JSONRPC_VERSION,
			Id:      baseMessage.Id,
			Result:  result,
		}
	case "tools/list":
		var req mcp.ListToolsRequest
		if err := json.Unmarshal(body, &req); err != nil {
			err := fmt.Errorf("invalid mcp tools list request: %w", err)
			res = newJSONRPCError(baseMessage.Id, mcp.INVALID_REQUEST, err.Error(), nil)
			break
		}
		toolset, ok := s.toolsets[""]
		if !ok {
			err := fmt.Errorf("toolset does not exist")
			res = newJSONRPCError(baseMessage.Id, mcp.INVALID_REQUEST, err.Error(), nil)
			break
		}
		result := mcp.ToolsList(toolset)
		res = mcp.JSONRPCResponse{
			Jsonrpc: mcp.JSONRPC_VERSION,
			Id:      baseMessage.Id,
			Result:  result,
		}
	case "tools/call":
		var req mcp.CallToolRequest
		if err := json.Unmarshal(body, &req); err != nil {
			err := fmt.Errorf("invalid mcp tools call request: %w", err)
			res = newJSONRPCError(baseMessage.Id, mcp.INVALID_REQUEST, err.Error(), nil)
			break
		}
		toolName := req.Params.Name
		toolArgument := req.Params.Arguments
		tool, ok := s.tools[toolName]
		if !ok {
			err := fmt.Errorf("invalid tool name: tool with name %q does not exist", toolName)
			res = newJSONRPCError(baseMessage.Id, mcp.INVALID_PARAMS, err.Error(), nil)
			break
		}

		// marshal arguments and decode it using decodeJSON instead to prevent loss between floats/int.
		aMarshal, err := json.Marshal(toolArgument)
		if err != nil {
			err := fmt.Errorf("unable to marshal tools argument: %w", err)
			res = newJSONRPCError(baseMessage.Id, mcp.INTERNAL_ERROR, err.Error(), nil)
			break
		}
		var data map[string]any
		if err = decodeJSON(bytes.NewBuffer(aMarshal), &data); err != nil {
			err := fmt.Errorf("unable to decode tools argument: %w", err)
			res = newJSONRPCError(baseMessage.Id, mcp.INTERNAL_ERROR, err.Error(), nil)
			break
		}

		// claimsFromAuth maps the name of the authservice to the claims retrieved from it.
		// Since MCP doesn't support auth, an empty map will be use every time.
		claimsFromAuth := make(map[string]map[string]any)

		params, err := tool.ParseParams(data, claimsFromAuth)
		if err != nil {
			err = fmt.Errorf("provided parameters were invalid: %w", err)
			res = newJSONRPCError(baseMessage.Id, mcp.INVALID_PARAMS, err.Error(), nil)
			break
		}

		result := mcp.ToolCall(tool, params)
		res = mcp.JSONRPCResponse{
			Jsonrpc: mcp.JSONRPC_VERSION,
			Id:      baseMessage.Id,
			Result:  result,
		}
	default:
		res = newJSONRPCError(baseMessage.Id, mcp.METHOD_NOT_FOUND, fmt.Sprintf("invalid method %s", baseMessage.Method), nil)
	}

	render.JSON(w, r, res)
}

// newJSONRPCError is the response sent back when an error has been encountered in mcp.
func newJSONRPCError(id mcp.RequestId, code int, message string, data any) mcp.JSONRPCError {
	return mcp.JSONRPCError{
		Jsonrpc: mcp.JSONRPC_VERSION,
		Id:      id,
		Error: mcp.McpError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
}
