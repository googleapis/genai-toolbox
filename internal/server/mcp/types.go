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
