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
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/google/uuid"
	"github.com/googleapis/genai-toolbox/internal/server/mcp"
	mcputil "github.com/googleapis/genai-toolbox/internal/server/mcp/util"
	"github.com/googleapis/genai-toolbox/internal/util"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
)

type sseSession struct {
	writer     http.ResponseWriter
	flusher    http.Flusher
	done       chan struct{}
	eventQueue chan string
}

// mcpSession represents each mcp session connected through initialize method.
type mcpSession struct {
	// protocol version negotiated during initialization
	protocol string
	// only available for connections that uses sse
	sseSession *sseSession
	// represent if the the initialization is successful
	initialized bool
}

// mcpManager manages and control access to mcp sesisons
type mcpManager struct {
	mu          sync.RWMutex
	mcpSessions map[string]*mcpSession
}

func (m *mcpManager) get(id string) (*mcpSession, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	session, ok := m.mcpSessions[id]
	return session, ok
}

func (m *mcpManager) add(id string, session *mcpSession) {
	m.mu.Lock()
	m.mcpSessions[id] = session
	m.mu.Unlock()
}

func (m *mcpManager) remove(id string) {
	m.mu.Lock()
	delete(m.mcpSessions, id)
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

	r.Route("/{toolsetName}", func(r chi.Router) {
		r.Get("/sse", func(w http.ResponseWriter, r *http.Request) { sseHandler(s, w, r) })
		r.Post("/", func(w http.ResponseWriter, r *http.Request) { mcpHandler(s, w, r) })
	})

	return r, nil
}

// sseHandler handles sse initialization and message.
func sseHandler(s *Server, w http.ResponseWriter, r *http.Request) {
	ctx, span := s.instrumentation.Tracer.Start(r.Context(), "toolbox/server/mcp/sse")
	r = r.WithContext(ctx)

	sessionId := uuid.New().String()
	toolsetName := chi.URLParam(r, "toolsetName")
	s.logger.DebugContext(ctx, fmt.Sprintf("toolset name: %s", toolsetName))
	span.SetAttributes(attribute.String("session_id", sessionId))
	span.SetAttributes(attribute.String("toolset_name", toolsetName))

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	var err error
	defer func() {
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
		status := "success"
		if err != nil {
			status = "error"
		}
		s.instrumentation.McpSse.Add(
			r.Context(),
			1,
			metric.WithAttributes(attribute.String("toolbox.toolset.name", toolsetName)),
			metric.WithAttributes(attribute.String("toolbox.sse.sessionId", sessionId)),
			metric.WithAttributes(attribute.String("toolbox.operation.status", status)),
		)
	}()

	flusher, ok := w.(http.Flusher)
	if !ok {
		err = fmt.Errorf("unable to retrieve flusher for sse")
		s.logger.DebugContext(ctx, err.Error())
		_ = render.Render(w, r, newErrResponse(err, http.StatusInternalServerError))
	}
	session := &sseSession{
		writer:     w,
		flusher:    flusher,
		done:       make(chan struct{}),
		eventQueue: make(chan string, 100),
	}
	mcpSession := &mcpSession{
		protocol:    "", // not yet negotiated protocol version
		sseSession:  session,
		initialized: false,
	}
	s.mcpManager.add(sessionId, mcpSession)
	defer s.mcpManager.remove(sessionId)

	// https scheme formatting if (forwarded) request is a TLS request
	proto := r.Header.Get("X-Forwarded-Proto")
	if proto == "" {
		if r.TLS == nil {
			proto = "http"
		} else {
			proto = "https"
		}
	}

	// send initial endpoint event
	toolsetURL := ""
	if toolsetName != "" {
		toolsetURL = fmt.Sprintf("/%s", toolsetName)
	}
	messageEndpoint := fmt.Sprintf("%s://%s/mcp%s?sessionId=%s", proto, r.Host, toolsetURL, sessionId)
	s.logger.DebugContext(ctx, fmt.Sprintf("sending endpoint event: %s", messageEndpoint))
	fmt.Fprintf(w, "event: endpoint\ndata: %s\n\n", messageEndpoint)
	flusher.Flush()

	clientClose := r.Context().Done()
	for {
		select {
		// Ensure that only a single responses are written at once
		case event := <-session.eventQueue:
			fmt.Fprint(w, event)
			s.logger.DebugContext(ctx, fmt.Sprintf("sending event: %s", event))
			flusher.Flush()
			// channel for client disconnection
		case <-clientClose:
			close(session.done)
			s.logger.DebugContext(ctx, "client disconnected")
			return
		}
	}
}

// mcpHandler handles all mcp messages.
func mcpHandler(s *Server, w http.ResponseWriter, r *http.Request) {
	ctx := util.WithLogger(r.Context(), s.logger)

	ctx, span := s.instrumentation.Tracer.Start(ctx, "toolbox/server/mcp")
	r = r.WithContext(ctx)

	var mcpSess *mcpSession
	var sessionId, protocolVersion string

	// session Id could present in either the header or the URL link (via sse)
	sessionId = r.Header.Get("Mcp-Session-Id")
	if sessionId == "" {
		// if session id is not in header, try checking the url query
		sessionId = r.URL.Query().Get("sessionId")
	}
	// sessionId is received during sse or initialization
	if sessionId != "" {
		var ok bool
		mcpSess, ok = s.mcpManager.get(sessionId)
		if !ok {
			s.logger.DebugContext(ctx, "mcp session not available")
		}
		protocolVersion = mcpSess.protocol
	} else {
		sessionId = uuid.New().String()
		mcpSess = &mcpSession{}
		s.mcpManager.add(sessionId, mcpSess)
	}
	// TODO: If client send HTTP DELETE to MCP Endpoint with the `MCP-Session-Id`
	// header, remove the session from mcpManager.

	toolsetName := chi.URLParam(r, "toolsetName")
	s.logger.DebugContext(ctx, fmt.Sprintf("toolset name: %s", toolsetName))
	span.SetAttributes(attribute.String("toolset_name", toolsetName))

	var id, toolName, method string
	var err error
	defer func() {
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()

		status := "success"
		if err != nil {
			status = "error"
		}
		s.instrumentation.McpPost.Add(
			r.Context(),
			1,
			metric.WithAttributes(attribute.String("toolbox.sse.messageId", id)),
			metric.WithAttributes(attribute.String("toolbox.sse.sessionId", sessionId)),
			metric.WithAttributes(attribute.String("toolbox.tool.name", toolName)),
			metric.WithAttributes(attribute.String("toolbox.toolset.name", toolsetName)),
			metric.WithAttributes(attribute.String("toolbox.method", method)),
			metric.WithAttributes(attribute.String("toolbox.operation.status", status)),
		)
	}()

	// Read and returns a body from io.Reader
	body, err := io.ReadAll(r.Body)
	if err != nil {
		// Generate a new uuid if unable to decode
		id = uuid.New().String()
		s.logger.DebugContext(ctx, err.Error())
		render.JSON(w, r, mcputil.NewError(id, mcputil.PARSE_ERROR, err.Error(), nil))
	}

	// Generic baseMessage could either be a JSONRPCNotification or JSONRPCRequest
	var baseMessage struct {
		Jsonrpc string            `json:"jsonrpc"`
		Method  string            `json:"method"`
		Id      mcputil.RequestId `json:"id,omitempty"`
	}
	if err = util.DecodeJSON(bytes.NewBuffer(body), &baseMessage); err != nil {
		// Generate a new uuid if unable to decode
		id := uuid.New().String()
		s.logger.DebugContext(ctx, err.Error())
		render.JSON(w, r, mcputil.NewError(id, mcputil.PARSE_ERROR, err.Error(), nil))
		return
	}

	// Check if method is present
	if baseMessage.Method == "" {
		err = fmt.Errorf("method not found")
		s.logger.DebugContext(ctx, err.Error())
		render.JSON(w, r, mcputil.NewError(baseMessage.Id, mcputil.METHOD_NOT_FOUND, err.Error(), nil))
		return
	}

	// Check for JSON-RPC 2.0
	if baseMessage.Jsonrpc != mcputil.JSONRPC_VERSION {
		err = fmt.Errorf("invalid json-rpc version")
		s.logger.DebugContext(ctx, err.Error())
		render.JSON(w, r, mcputil.NewError(baseMessage.Id, mcputil.INVALID_REQUEST, err.Error(), nil))
		return
	}

	// Check if message is a notification
	if baseMessage.Id == nil {
		mcp.NotificationHandler(ctx, baseMessage.Method, body)
		if baseMessage.Method == mcputil.INITIALIZE_NOTIFICATIONS {
			mcpSess.initialized = true
		}
		// Notifications do not expect a response
		// Toolbox doesn't do anything with notifications yet
		w.WriteHeader(http.StatusAccepted)
		return
	}
	id = fmt.Sprintf("%s", baseMessage.Id)
	method = baseMessage.Method
	s.logger.DebugContext(ctx, fmt.Sprintf("method is: %s", method))

	var res any
	switch method {
	case mcputil.INITIALIZE:
		res, protocolVersion = mcp.InitializeResponse(ctx, baseMessage.Id, body, s.version)
		mcpSess.protocol = protocolVersion
		w.Header().Add("Mcp-Session-Id", sessionId)
	default:
		if !mcpSess.initialized {
			err = fmt.Errorf("session is not initialized")
			s.logger.DebugContext(ctx, err.Error())
			res = mcputil.NewError(baseMessage.Id, mcputil.INVALID_REQUEST, err.Error(), nil)
			break
		}
		toolset, ok := s.toolsets[toolsetName]
		if !ok {
			err = fmt.Errorf("toolset does not exist")
			s.logger.DebugContext(ctx, err.Error())
			res = mcputil.NewError(baseMessage.Id, mcputil.INVALID_REQUEST, err.Error(), nil)
			break
		}
		res, toolName = mcp.McpHandler(ctx, protocolVersion, baseMessage.Id, baseMessage.Method, toolset, s.tools, body)
	}

	// retrieve sse session
	sseSess := mcpSess.sseSession
	if sseSess == nil {
		s.logger.DebugContext(ctx, "sse session not available")
	} else {
		// queue sse event
		eventData, _ := json.Marshal(res)
		select {
		case sseSess.eventQueue <- fmt.Sprintf("event: message\ndata: %s\n\n", eventData):
			s.logger.DebugContext(ctx, "event queue successful")
		case <-sseSess.done:
			s.logger.DebugContext(ctx, "session is close")
		default:
			s.logger.DebugContext(ctx, "unable to add to event queue")
		}
	}

	// send HTTP response
	render.JSON(w, r, res)
}
