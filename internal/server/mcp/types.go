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

package mcp

import (
	"github.com/googleapis/genai-toolbox/internal/tools"
)

// SERVER_NAME is the server name used in Implementation.
const SERVER_NAME = "Toolbox"

// LATEST_PROTOCOL_VERSION is the most recent version of the MCP protocol.
const LATEST_PROTOCOL_VERSION = "2024-11-05"

// JSONRPC_VERSION is the version of JSON-RPC used by MCP.
const JSONRPC_VERSION = "2.0"

// Standard JSON-RPC error codes
const (
	PARSE_ERROR      = -32700
	INVALID_REQUEST  = -32600
	METHOD_NOT_FOUND = -32601
	INVALID_PARAMS   = -32602
	INTERNAL_ERROR   = -32603
)

// JSONRPCMessage represents either a JSONRPCRequest, JSONRPCNotification, JSONRPCResponse, or JSONRPCError.
type JSONRPCMessage interface{}

// ProgressToken is used to associate progress notifications with the original request.
type ProgressToken interface{}

// Request represents a bidirectional message with method and parameters expecting a response.
type Request struct {
	Method string `json:"method"`
	Params struct {
		Meta struct {
			// If specified, the caller is requesting out-of-band progress
			// notifications for this request (as represented by
			// notifications/progress). The value of this parameter is an
			// opaque token that will be attached to any subsequent
			// notifications. The receiver is not obligated to provide these
			// notifications.
			ProgressToken ProgressToken `json:"progressToken,omitempty"`
		} `json:"_meta,omitempty"`
	} `json:"params,omitempty"`
}

// Notification is a one-way message requiring no response.
type Notification struct {
	Method string `json:"method"`
	Params struct {
		Meta map[string]interface{} `json:"_meta,omitempty"`
	} `json:"params,omitempty"`
}

// Result represents a response for the request query.
type Result struct {
	// This result property is reserved by the protocol to allow clients and
	// servers to attach additional metadata to their responses.
	Meta map[string]interface{} `json:"_meta,omitempty"`
}

// RequestId is a uniquely identifying ID for a request in JSON-RPC.
// It can be any JSON-serializable value, typically a number or string.
type RequestId interface{}

// JSONRPCRequest represents a request that expects a response.
type JSONRPCRequest struct {
	Jsonrpc string    `json:"jsonrpc"`
	Id      RequestId `json:"id"`
	Request
	Params any `json:"params,omitempty"`
}

// JSONRPCNotification represents a notification which does not expect a response.
type JSONRPCNotification struct {
	Jsonrpc string `json:"jsonrpc"`
	Notification
}

// JSONRPCResponse represents a successful (non-error) response to a request.
type JSONRPCResponse struct {
	Jsonrpc string      `json:"jsonrpc"`
	Id      RequestId   `json:"id"`
	Result  interface{} `json:"result"`
}

// McpError represents the error content.
type McpError struct {
	// The error type that occurred.
	Code int `json:"code"`
	// A short description of the error. The message SHOULD be limited
	// to a concise single sentence.
	Message string `json:"message"`
	// Additional information about the error. The value of this member
	// is defined by the sender (e.g. detailed error information, nested errors etc.).
	Data interface{} `json:"data,omitempty"`
}

// JSONRPCError represents a non-successful (error) response to a request.
type JSONRPCError struct {
	Jsonrpc string    `json:"jsonrpc"`
	Id      RequestId `json:"id"`
	Error   McpError  `json:"error"`
}

/* Empty result */

// EmptyResult represents a response that indicates success but carries no data.
type EmptyResult Result

/* Initialization */

// Params to define MCP Client during initialize request.
type InitializeParams struct {
	// The latest version of the Model Context Protocol that the client supports.
	// The client MAY decide to support older versions as well.
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ClientCapabilities `json:"capabilities"`
	ClientInfo      Implementation     `json:"clientInfo"`
}

// InitializeRequest is sent from the client to the server when it first
// connects, asking it to begin initialization.
type InitializeRequest struct {
	Request
	Params InitializeParams `json:"params"`
}

// InitializeResult is sent after receiving an initialize request from the
// client.
type InitializeResult struct {
	Result
	// The version of the Model Context Protocol that the server wants to use.
	// This may not match the version that the client requested. If the client cannot
	// support this version, it MUST disconnect.
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ServerCapabilities `json:"capabilities"`
	ServerInfo      Implementation     `json:"serverInfo"`
	// Instructions describing how to use the server and its features.
	//
	// This can be used by clients to improve the LLM's understanding of
	// available tools, resources, etc. It can be thought of like a "hint" to the model.
	// For example, this information MAY be added to the system prompt.
	Instructions string `json:"instructions,omitempty"`
}

// InitializedNotification is sent from the client to the server after
// initialization has finished.
type InitializedNotification struct {
	Notification
}

// ListChange represents whether the server supports notification for changes to the capabilities.
type ListChanged struct {
	ListChanged *bool `json:"listChanged,omitempty"`
}

// ClientCapabilities represents capabilities a client may support. Known
// capabilities are defined here, in this schema, but this is not a closed set: any
// client can define its own, additional capabilities.
type ClientCapabilities struct {
	// Experimental, non-standard capabilities that the client supports.
	Experimental map[string]interface{} `json:"experimental,omitempty"`
	// Present if the client supports listing roots.
	Roots *ListChanged `json:"roots,omitempty"`
	// Present if the client supports sampling from an LLM.
	Sampling struct{} `json:"sampling,omitempty"`
}

// ServerCapabilities represents capabilities that a server may support. Known
// capabilities are defined here, in this schema, but this is not a closed set: any
// server can define its own, additional capabilities.
type ServerCapabilities struct {
	Tools *ListChanged `json:"tools,omitempty"`
}

// Implementation describes the name and version of an MCP implementation.
type Implementation struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

/* Pagination */

// Cursor is an opaque token used to represent a cursor for pagination.
type Cursor string

type PaginatedRequest struct {
	Request
	Params struct {
		// An opaque token representing the current pagination position.
		// If provided, the server should return results starting after this cursor.
		Cursor Cursor `json:"cursor,omitempty"`
	} `json:"params,omitempty"`
}

type PaginatedResult struct {
	Result
	// An opaque token representing the pagination position after the last returned result.
	// If present, there may be more results available.
	NextCursor Cursor `json:"nextCursor,omitempty"`
}

/* Tools */

// Sent from the client to request a list of tools the server has.
type ListToolsRequest struct {
	PaginatedRequest
}

// The server's response to a tools/list request from the client.
type ListToolsResult struct {
	PaginatedResult
	Tools []tools.McpManifest `json:"tools"`
}

// Used by the client to invoke a tool provided by the server.
type CallToolRequest struct {
	Request
	Params struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments,omitempty"`
	} `json:"params,omitempty"`
}

// The sender or recipient of messages and data in a conversation.
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

// Base for objects that include optional annotations for the client.
// The client can use annotations to inform how objects are used or displayed
type Annotated struct {
	Annotations *struct {
		// Describes who the intended customer of this object or data is.
		// It can include multiple entries to indicate content useful for multiple
		// audiences (e.g., `["user", "assistant"]`).
		Audience []Role `json:"audience,omitempty"`
		// Describes how important this data is for operating the server.
		//
		// A value of 1 means "most important," and indicates that the data is
		// effectively required, while 0 means "least important," and indicates that
		// the data is entirely optional.
		//
		// @TJS-type number
		// @minimum 0
		// @maximum 1
		Priority float64 `json:"priority,omitempty"`
	} `json:"annotations,omitempty"`
}

// TextContent represents text provided to or from an LLM.
type TextContent struct {
	Annotated
	Type string `json:"type"`
	// The text content of the message.
	Text string `json:"text"`
}

// The server's response to a tool call.
//
// Any errors that originate from the tool SHOULD be reported inside the result
// object, with `isError` set to true, _not_ as an MCP protocol-level error
// response. Otherwise, the LLM would not be able to see that an error occurred
// and self-correct.
//
// However, any errors in _finding_ the tool, an error indicating that the
// server does not support tool calls, or any other exceptional conditions,
// should be reported as an MCP error response.
type CallToolResult struct {
	Result
	// Could be either a TextContent, ImageContent, or EmbeddedResources
	// For Toolbox, we will only be sending TextContent
	Content []TextContent `json:"content"`
	// Whether the tool call ended in an error.
	// If not set, this is assumed to be false (the call was successful).
	IsError bool                `json:"isError,omitempty"`
}
