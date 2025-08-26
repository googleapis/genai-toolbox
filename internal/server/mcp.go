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
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/google/uuid"
	"github.com/googleapis/genai-toolbox/internal/server/mcp"
	"github.com/googleapis/genai-toolbox/internal/server/mcp/jsonrpc"
	mcputil "github.com/googleapis/genai-toolbox/internal/server/mcp/util"
	v20241105 "github.com/googleapis/genai-toolbox/internal/server/mcp/v20241105"
	v20250326 "github.com/googleapis/genai-toolbox/internal/server/mcp/v20250326"
	"github.com/googleapis/genai-toolbox/internal/tools"
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
	lastActive time.Time
}

// sseManager manages and control access to sse sessions
type sseManager struct {
	mu          sync.Mutex
	sseSessions map[string]*sseSession
}

func (m *sseManager) get(id string) (*sseSession, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	session, ok := m.sseSessions[id]
	session.lastActive = time.Now()
	return session, ok
}

func newSseManager(ctx context.Context) *sseManager {
	sseM := &sseManager{
		mu:          sync.Mutex{},
		sseSessions: make(map[string]*sseSession),
	}
	go sseM.cleanupRoutine(ctx)
	return sseM
}

func (m *sseManager) add(id string, session *sseSession) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sseSessions[id] = session
	session.lastActive = time.Now()
}

func (m *sseManager) remove(id string) {
	m.mu.Lock()
	delete(m.sseSessions, id)
	m.mu.Unlock()
}

func (m *sseManager) cleanupRoutine(ctx context.Context) {
	timeout := 10 * time.Minute
	ticker := time.NewTicker(timeout)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			func() {
				m.mu.Lock()
				defer m.mu.Unlock()
				now := time.Now()
				for id, sess := range m.sseSessions {
					if now.Sub(sess.lastActive) > timeout {
						delete(m.sseSessions, id)
					}
				}
			}()
		}
	}
}

type stdioSession struct {
	protocol string
	server   *Server
	reader   *bufio.Reader
	writer   io.Writer
}

func NewStdioSession(s *Server, stdin io.Reader, stdout io.Writer) *stdioSession {
	stdioSession := &stdioSession{
		server: s,
		reader: bufio.NewReader(stdin),
		writer: stdout,
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
		v, res, err := processMcpMessage(ctx, []byte(line), s.server, s.protocol, "", "")
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

	r.Use(middleware.AllowContentType("application/json", "application/json-rpc", "application/jsonrequest"))
	r.Use(middleware.StripSlashes)
	r.Use(render.SetContentType(render.ContentTypeJSON))

	r.Get("/sse", func(w http.ResponseWriter, r *http.Request) { sseHandler(s, w, r) })
	r.Get("/", func(w http.ResponseWriter, r *http.Request) { methodNotAllowed(s, w, r) })
	r.Post("/", func(w http.ResponseWriter, r *http.Request) { httpHandler(s, w, r) })
	r.Delete("/", func(w http.ResponseWriter, r *http.Request) {})

	r.Route("/{toolsetName}", func(r chi.Router) {
		r.Get("/sse", func(w http.ResponseWriter, r *http.Request) { sseHandler(s, w, r) })
		r.Get("/", func(w http.ResponseWriter, r *http.Request) { methodNotAllowed(s, w, r) })
		r.Post("/", func(w http.ResponseWriter, r *http.Request) { httpHandler(s, w, r) })
		r.Delete("/", func(w http.ResponseWriter, r *http.Request) {})
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
	s.sseManager.add(sessionId, session)
	defer s.sseManager.remove(sessionId)

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

// methodNotAllowed handles all mcp messages.
func methodNotAllowed(s *Server, w http.ResponseWriter, r *http.Request) {
	err := fmt.Errorf("toolbox does not support streaming in streamable HTTP transport")
	s.logger.DebugContext(r.Context(), err.Error())
	_ = render.Render(w, r, newErrResponse(err, http.StatusMethodNotAllowed))
}

// httpHandler handles all mcp messages.
func httpHandler(s *Server, w http.ResponseWriter, r *http.Request) {
	ctx, span := s.instrumentation.Tracer.Start(r.Context(), "toolbox/server/mcp")
	r = r.WithContext(ctx)
	ctx = util.WithLogger(r.Context(), s.logger)

	var sessionId, protocolVersion string
	var session *sseSession

	// check if client connects via sse
	// v2024-11-05 supports http with sse
	paramSessionId := r.URL.Query().Get("sessionId")
	if paramSessionId != "" {
		sessionId = paramSessionId
		protocolVersion = v20241105.PROTOCOL_VERSION
		var ok bool
		session, ok = s.sseManager.get(sessionId)
		if !ok {
			s.logger.DebugContext(ctx, "sse session not available")
		}
	}

	// check if client have `Mcp-Session-Id` header
	// if `Mcp-Session-Id` header is set, we are using v2025-03-26 since
	// previous version doesn't use this header.
	headerSessionId := r.Header.Get("Mcp-Session-Id")
	if headerSessionId != "" {
		protocolVersion = v20250326.PROTOCOL_VERSION
	}

	// check if client have `MCP-Protocol-Version` header
	headerProtocolVersion := r.Header.Get("MCP-Protocol-Version")
	if headerProtocolVersion != "" {
		if !mcp.VerifyProtocolVersion(headerProtocolVersion) {
			err := fmt.Errorf("invalid protocol version: %s", headerProtocolVersion)
			_ = render.Render(w, r, newErrResponse(err, http.StatusBadRequest))
			return
		}
		protocolVersion = headerProtocolVersion
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
		return
	}

	accessToken := tools.AccessToken(r.Header.Get("Authorization"))

	v, res, err := processMcpMessage(ctx, body, s, protocolVersion, toolsetName, accessToken)

	if err != nil {
		s.logger.DebugContext(ctx, err.Error())

		// Check for specific authorization errors to return an HTTP 401.
		errStr := err.Error()
		if errors.Is(err, tools.ErrUnauthorized) || strings.Contains(errStr, "Error 401") {
			_ = render.Render(w, r, newErrResponse(err, http.StatusUnauthorized))
			return
		}
		if strings.Contains(errStr, "Error 403") {
			_ = render.Render(w, r, newErrResponse(err, http.StatusForbidden))
			return
		}
	}

	// notifications will return empty string
	if res == nil {
		// Notifications do not expect a response
		// Toolbox doesn't do anything with notifications yet
		w.WriteHeader(http.StatusAccepted)
		return
	}

	// for v20250326, add the `Mcp-Session-Id` header
	if v == v20250326.PROTOCOL_VERSION {
		sessionId = uuid.New().String()
		w.Header().Set("Mcp-Session-Id", sessionId)
	}

	if session != nil {
		// queue sse event
		eventData, _ := json.Marshal(res)
		select {
		case session.eventQueue <- fmt.Sprintf("event: message\ndata: %s\n\n", eventData):
			s.logger.DebugContext(ctx, "event queue successful")
		case <-session.done:
			s.logger.DebugContext(ctx, "session is close")
		default:
			s.logger.DebugContext(ctx, "unable to add to event queue")
		}
	}

	// send HTTP response
	render.JSON(w, r, res)
}

// processMcpMessage process the messages received from clients
func processMcpMessage(ctx context.Context, body []byte, s *Server, protocolVersion string, toolsetName string, accessToken tools.AccessToken) (string, any, error) {
	logger, err := util.LoggerFromContext(ctx)
	if err != nil {
		return "", jsonrpc.NewError("", jsonrpc.INTERNAL_ERROR, err.Error(), nil), err
	}

	// Generic baseMessage could either be a JSONRPCNotification or JSONRPCRequest
	var baseMessage jsonrpc.BaseMessage
	if err = util.DecodeJSON(bytes.NewBuffer(body), &baseMessage); err != nil {
		// Generate a new uuid if unable to decode
		id := uuid.New().String()

		// check if user is sending a batch request
		var a []any
		unmarshalErr := json.Unmarshal(body, &a)
		if unmarshalErr == nil {
			err = fmt.Errorf("not supporting batch requests")
			return "", jsonrpc.NewError(id, jsonrpc.INVALID_REQUEST, err.Error(), nil), err
		}

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
		err := mcp.NotificationHandler(ctx, body)
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
		toolset, ok := s.ResourceMgr.GetToolset(toolsetName)
		if !ok {
			err = fmt.Errorf("toolset does not exist")
			return "", jsonrpc.NewError(baseMessage.Id, jsonrpc.INVALID_REQUEST, err.Error(), nil), err
		}
		res, err := mcp.ProcessMethod(ctx, protocolVersion, baseMessage.Id, baseMessage.Method, toolset, s.ResourceMgr.GetToolsMap(), body, accessToken)
		return "", res, err
	}
}
