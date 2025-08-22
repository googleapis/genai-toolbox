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

package util

import (
	"errors"

	"github.com/googleapis/genai-toolbox/internal/server/mcp/jsonrpc"
)

const (
	// SERVER_NAME is the server name used in Implementation.
	SERVER_NAME = "Toolbox"
	// methods that are supported
	INITIALIZE = "initialize"
)

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
	jsonrpc.Request
	Params InitializeParams `json:"params"`
}

// InitializeResult is sent after receiving an initialize request from the
// client.
type InitializeResult struct {
	jsonrpc.Result
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
	jsonrpc.Notification
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

// Base interface for metadata with name (identifier) and title (display name) properties.
type BaseMetadata struct {
	// Intended for programmatic or logical use, but used as a display name in past specs
	// or fallback (if title isn't present).
	Name string `json:"name"`
	// Intended for UI and end-user contexts — optimized to be human-readable and easily understood,
	//even by those unfamiliar with domain-specific terminology.
	//
	// If not provided, the name should be used for display (except for Tool,
	// where `annotations.title` should be given precedence over using `name`,
	// if present).
	Title string `json:"title,omitempty"`
}

// Implementation describes the name and version of an MCP implementation.
type Implementation struct {
	BaseMetadata
	Version string `json:"version"`
}

// ErrUnauthorizedRequest is thrown when a tool call request is unauthorized
var ErrUnauthorizedRequest = errors.New("authorization header required but not provided")
