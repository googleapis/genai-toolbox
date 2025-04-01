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
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/google/uuid"
	"github.com/googleapis/genai-toolbox/internal/server/mcp"
)

type sseSession struct {
	sessionId  string
	writer     http.ResponseWriter
	flusher    http.Flusher
	done       chan struct{}
	eventQueue chan string
}

// sseManager manages and control access to sse sessions
type sseManager struct {
	mu          sync.RWMutex
	sseSessions map[string]*sseSession
}

func (m *sseManager) get(id string) (*sseSession, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	session, ok := m.sseSessions[id]
	return session, ok
}

func (m *sseManager) add(id string, session *sseSession) {
	m.mu.Lock()
	m.sseSessions[id] = session
	m.mu.Unlock()
}

func (m *sseManager) remove(id string) {
	m.mu.Lock()
	delete(m.sseSessions, id)
	m.mu.Unlock()
}

// mcpRouter creates a router that represents the routes under /mcp
func mcpRouter(s *Server) (chi.Router, error) {
	r := chi.NewRouter()

	r.Use(middleware.AllowContentType("application/json"))
	r.Use(middleware.StripSlashes)
	r.Use(render.SetContentType(render.ContentTypeJSON))

	r.Get("/sse", func(w http.ResponseWriter, r *http.Request) { sseHandler(s, w, r) })
	r.Post("/", func(w http.ResponseWriter, r *http.Request) { mcpHandler(s, w, r) })

	return r, nil
}

// sseHandler handles sse initialization and message.
func sseHandler(s *Server, w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		err := fmt.Errorf("unable to retrieve flusher for sse")
		_ = render.Render(w, r, newErrResponse(err, http.StatusInternalServerError))
	}
	sessionId := uuid.New().String()
	session := &sseSession{
		sessionId:  sessionId,
		writer:     w,
		flusher:    flusher,
		done:       make(chan struct{}),
		eventQueue: make(chan string, 100),
	}
	s.sseManager.add(sessionId, session)
	defer s.sseManager.remove(sessionId)

	// send initil endpoint event
	messageEndpoint := fmt.Sprintf("http://127.0.0.1:5000/mcp?sessionId=%s", sessionId)
	fmt.Fprintf(w, "event: endpoint\ndata: %s\n\n", messageEndpoint)
	flusher.Flush()

	clientClose := r.Context().Done()
	for {
		select {
		// Ensure that only a single responses are written at once
		case event := <-session.eventQueue:
			fmt.Fprint(w, event)
			flusher.Flush()
			// channel for client disconnection
		case <-clientClose:
			close(session.done)
			s.logger.DebugContext(context.Background(), "client disconnected")
			return
		}
	}
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

	// retrieve sse session
	sessionId := r.URL.Query().Get("sessionId")
	session, ok := s.sseManager.get(sessionId)
	if !ok {
		s.logger.DebugContext(context.Background(), "sse session not available")
	} else {
		// queue sse event
		eventData, _ := json.Marshal(res)
		select {
		case session.eventQueue <- fmt.Sprintf("event: message\ndata: %s\n\n", eventData):
			s.logger.DebugContext(context.Background(), "event queue successful")
		case <-session.done:
			s.logger.DebugContext(context.Background(), "session is close")
		default:
			s.logger.DebugContext(context.Background(), "unable to add to event queue")
		}
	}

	// send HTTP response
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
