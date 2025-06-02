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
	"bufio"
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
	"github.com/googleapis/genai-toolbox/internal/server/mcp/jsonrpc"
	mcputil "github.com/googleapis/genai-toolbox/internal/server/mcp/util"
	v20241105 "github.com/googleapis/genai-toolbox/internal/server/mcp/v20241105"
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

// mcpSession represents each mcp session using the sse transport used to
// keep track if the session is initialized and associated protocol version.
type mcpSession struct {
	// protocol version negotiated during initialization
	protocol string
	// only available for connections that uses sse
	sseSession *sseSession
	// represent if the the initialization is successful
	initialized *bool
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

type stdioSession struct {
	protocol    string
	initialized *bool
	server      *Server
	reader      *bufio.Reader
	writer      io.Writer
}

func NewStdioSession(s *Server, stdin io.Reader, stdout io.Writer) *stdioSession {
	init := false
	stdioSession := &stdioSession{
		initialized: &init,
		server:      s,
		reader:      bufio.NewReader(stdin),
		writer:      stdout,
	}
	return stdioSession
}

func (s *stdioSession) Start(ctx context.Context) error {
	return s.readInputStream(ctx)
}

// readInputStream reads requests/notifications from MCP clients through stdin
func (s *stdioSession) readInputStream(ctx context.Context) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		line, err := s.readLine(ctx)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		v, res, err := processMcpMessage(ctx, []byte(line), s.server, s.protocol, s.initialized, true, "")
		if err != nil {
			// errors during the processing of message will generate a valid MCP Error response.
			// server can continue to run.
			s.server.logger.ErrorContext(ctx, err.Error())
		}
		if v != "" {
			s.protocol = v
		}
		// no responses for notifications
		if res != nil {
			if err = s.write(ctx, res); err != nil {
				return err
			}
        }
	}
}

// readLine process each line within the input stream.
func (s *stdioSession) readLine(ctx context.Context) (string, error) {
	readChan := make(chan string, 1)
	errChan := make(chan error, 1)
	done := make(chan struct{})
	defer close(done)
	defer close(readChan)
	defer close(errChan)

	go func() {
		select {
		case <-done:
			return
		default:
			line, err := s.reader.ReadString('\n')
			if err != nil {
				select {
				case errChan <- err:
				case <-done:
				}
				return
			}
			select {
			case readChan <- line:
			case <-done:
			}
			return
		}
	}()

	select {
	// if context is cancelled, return an empty string
	case <-ctx.Done():
		return "", ctx.Err()
	// return error if error is found
	case err := <-errChan:
		return "", err
	// return line if successful
	case line := <-readChan:
		return line, nil
	}
}

// write writes to stdout with response to client
func (s *stdioSession) write(ctx context.Context, response any) error {
	res, _ := json.Marshal(response)

	_, err := fmt.Fprintf(s.writer, "%s\n", res)
	return err
}

// mcpRouter creates a router that represents the routes under /mcp
func mcpRouter(s *Server) (chi.Router, error) {
	r := chi.NewRouter()

	r.Use(middleware.AllowContentType("application/json"))
	r.Use(middleware.StripSlashes)
	r.Use(render.SetContentType(render.ContentTypeJSON))

	r.Get("/sse", func(w http.ResponseWriter, r *http.Request) { sseHandler(s, w, r) })
	r.Post("/", func(w http.ResponseWriter, r *http.Request) { httpHandler(s, w, r) })

	r.Route("/{toolsetName}", func(r chi.Router) {
		r.Get("/sse", func(w http.ResponseWriter, r *http.Request) { sseHandler(s, w, r) })
		r.Post("/", func(w http.ResponseWriter, r *http.Request) { httpHandler(s, w, r) })
	})

	// TODO: If client send HTTP DELETE to MCP Endpoint with the `MCP-Session-Id`
	// header, remove the session from mcpManager.

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
	init := false
	mcpSession := &mcpSession{
		protocol:    v20241105.PROTOCOL_VERSION, // sse is only supported for v2024-11-05
		sseSession:  session,
		initialized: &init,
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

// httpHandler handles all mcp messages.
func httpHandler(s *Server, w http.ResponseWriter, r *http.Request) {
	ctx, span := s.instrumentation.Tracer.Start(r.Context(), "toolbox/server/mcp")
	r = r.WithContext(ctx)
	ctx = util.WithLogger(r.Context(), s.logger)

	var mcpSess *mcpSession
	var sessionId, protocolVersion string
	// indicate if initialize have to be enforced
	var initialized *bool
	var initEnforced bool

	// check url for sessionId (if user connect via sse)
	sessionId = r.URL.Query().Get("sessionId")
	if sessionId == "" {
		// check for sessionId in the header
		// client had done the initialization step if this exists
		sessionId = r.Header.Get("Mcp-Session-Id")
	}
	// sessionId present, retrieve mcpSess and protocolVersion
	if sessionId != "" {
		var ok bool
		mcpSess, ok = s.mcpManager.get(sessionId)
		if !ok {
			s.logger.DebugContext(ctx, "mcp session not available")
		}
		protocolVersion = mcpSess.protocol
		initialized = mcpSess.initialized
		initEnforced = true
	} else {
		init := false
		initialized = &init
	}

	toolsetName := chi.URLParam(r, "toolsetName")
	s.logger.DebugContext(ctx, fmt.Sprintf("toolset name: %s", toolsetName))
	span.SetAttributes(attribute.String("toolset_name", toolsetName))

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
			metric.WithAttributes(attribute.String("toolbox.sse.sessionId", sessionId)),
			metric.WithAttributes(attribute.String("toolbox.operation.status", status)),
		)
	}()

	// Read and returns a body from io.Reader
	body, err := io.ReadAll(r.Body)
	if err != nil {
		// Generate a new uuid if unable to decode
		id := uuid.New().String()
		s.logger.DebugContext(ctx, err.Error())
		render.JSON(w, r, jsonrpc.NewError(id, jsonrpc.PARSE_ERROR, err.Error(), nil))
	}

	_, res, err := processMcpMessage(ctx, body, s, protocolVersion, initialized, initEnforced, toolsetName)
	// notifications will return empty string
	if res == nil {
		// Notifications do not expect a response
		// Toolbox doesn't do anything with notifications yet
		w.WriteHeader(http.StatusAccepted)
		return
	}
	if err != nil {
		s.logger.DebugContext(ctx, err.Error())
	}
	// TODO: implement this during implementation of v2025-03-26 schema
	// Some clients might be using the HTTP without SSE transport. Initialization is not
	// enforced for that transport protocol.
	//if protocolVersion == v20250326.PROTOCOL_VERSION {
	//    sessionId = uuid.New().String()
	//    mcpSess = &mcpSession{
	//        protocol: protocolVersion,
	//    }
	//    s.mcpManager.add(sessionId, mcpSess)
	//}

	if mcpSess != nil {
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
	}

	// send HTTP response
	render.JSON(w, r, res)
}

// processMcpMessage process the messages received from clients
func processMcpMessage(ctx context.Context, body []byte, s *Server, protocolVersion string, initialized *bool, initEnforced bool, toolsetName string) (string, any, error) {
	logger, err := util.LoggerFromContext(ctx)
	if err != nil {
		return "", jsonrpc.NewError("", jsonrpc.INTERNAL_ERROR, err.Error(), nil), err
	}

	// Generic baseMessage could either be a JSONRPCNotification or JSONRPCRequest
	var baseMessage jsonrpc.BaseMessage
	if err = util.DecodeJSON(bytes.NewBuffer(body), &baseMessage); err != nil {
		// Generate a new uuid if unable to decode
		id := uuid.New().String()
		return "", jsonrpc.NewError(id, jsonrpc.PARSE_ERROR, err.Error(), nil), err
	}

	// Check if method is present
	if baseMessage.Method == "" {
		err = fmt.Errorf("method not found")
		return "", jsonrpc.NewError(baseMessage.Id, jsonrpc.METHOD_NOT_FOUND, err.Error(), nil), err
	}
	logger.DebugContext(ctx, fmt.Sprintf("method is: %s", baseMessage.Method))

	// Check for JSON-RPC 2.0
	if baseMessage.Jsonrpc != jsonrpc.JSONRPC_VERSION {
		err = fmt.Errorf("invalid json-rpc version")
		return "", jsonrpc.NewError(baseMessage.Id, jsonrpc.INVALID_REQUEST, err.Error(), nil), err
	}

	// Check if message is a notification
	if baseMessage.Id == nil {
		err := mcp.NotificationHandler(ctx, body, initialized)
		return "", nil, err
	}

	switch baseMessage.Method {
	case mcputil.INITIALIZE:
		res, v, err := mcp.InitializeResponse(ctx, baseMessage.Id, body, s.version)
		if err != nil {
			return "", res, err
		}
		return v, res, err
	default:
		if initEnforced && !*initialized {
			err = fmt.Errorf("session is not initialized")
			return "", jsonrpc.NewError(baseMessage.Id, jsonrpc.INVALID_REQUEST, err.Error(), nil), err
		}

		toolset, ok := s.toolsets[toolsetName]
		if !ok {
			err = fmt.Errorf("toolset does not exist")
			return "", jsonrpc.NewError(baseMessage.Id, jsonrpc.INVALID_REQUEST, err.Error(), nil), err
		}
		res, err := mcp.ProcessMethod(ctx, protocolVersion, baseMessage.Id, baseMessage.Method, toolset, s.tools, body)
		return "", res, err
	}
}
