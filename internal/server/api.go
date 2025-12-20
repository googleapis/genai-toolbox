// Copyright 2024 Google LLC
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
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/util"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
)

// apiRouter creates a router that represents the routes under /api
func apiRouter(s *Server) (chi.Router, error) {
	r := chi.NewRouter()

	r.Use(middleware.AllowContentType("application/json"))
	r.Use(middleware.StripSlashes)
	r.Use(render.SetContentType(render.ContentTypeJSON))

	r.Get("/toolset", func(w http.ResponseWriter, r *http.Request) { toolsetHandler(s, w, r) })
	r.Get("/toolset/{toolsetName}", func(w http.ResponseWriter, r *http.Request) { toolsetHandler(s, w, r) })

	r.Route("/tool/{toolName}", func(r chi.Router) {
		r.Get("/", func(w http.ResponseWriter, r *http.Request) { toolGetHandler(s, w, r) })
		r.Post("/invoke", func(w http.ResponseWriter, r *http.Request) { toolInvokeHandler(s, w, r) })
	})

	return r, nil
}

// toolsetHandler handles the request for information about a Toolset.
func toolsetHandler(s *Server, w http.ResponseWriter, r *http.Request) {
	ctx, span := s.instrumentation.Tracer.Start(r.Context(), "toolbox/server/toolset/get")
	r = r.WithContext(ctx)

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
		s.instrumentation.ToolsetGet.Add(
			r.Context(),
			1,
			metric.WithAttributes(attribute.String("toolbox.name", toolsetName)),
			metric.WithAttributes(attribute.String("toolbox.operation.status", status)),
		)
	}()

	toolset, ok := s.ResourceMgr.GetToolset(toolsetName)
	if !ok {
		err = fmt.Errorf("toolset %q does not exist", toolsetName)
		s.logger.DebugContext(ctx, err.Error())
		_ = render.Render(w, r, newErrResponse(err, http.StatusNotFound))
		return
	}
	render.JSON(w, r, toolset.Manifest)
}

// toolGetHandler handles requests for a single Tool.
func toolGetHandler(s *Server, w http.ResponseWriter, r *http.Request) {
	ctx, span := s.instrumentation.Tracer.Start(r.Context(), "toolbox/server/tool/get")
	r = r.WithContext(ctx)

	toolName := chi.URLParam(r, "toolName")
	s.logger.DebugContext(ctx, fmt.Sprintf("tool name: %s", toolName))
	span.SetAttributes(attribute.String("tool_name", toolName))
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
		s.instrumentation.ToolGet.Add(
			r.Context(),
			1,
			metric.WithAttributes(attribute.String("toolbox.name", toolName)),
			metric.WithAttributes(attribute.String("toolbox.operation.status", status)),
		)
	}()
	tool, ok := s.ResourceMgr.GetTool(toolName)
	if !ok {
		err = fmt.Errorf("invalid tool name: tool with name %q does not exist", toolName)
		s.logger.DebugContext(ctx, err.Error())
		_ = render.Render(w, r, newErrResponse(err, http.StatusNotFound))
		return
	}
	// TODO: this can be optimized later with some caching
	m := tools.ToolsetManifest{
		ServerVersion: s.version,
		ToolsManifest: map[string]tools.Manifest{
			toolName: tool.Manifest(),
		},
	}

	render.JSON(w, r, m)
}

// toolInvokeHandler handles the API request to invoke a specific Tool.
func toolInvokeHandler(s *Server, w http.ResponseWriter, r *http.Request) {
	ctx, span := s.instrumentation.Tracer.Start(r.Context(), "toolbox/server/tool/invoke")
	r = r.WithContext(ctx)
	ctx = util.WithLogger(r.Context(), s.logger)

	toolName := chi.URLParam(r, "toolName")
	s.logger.DebugContext(ctx, fmt.Sprintf("tool name: %s", toolName))
	span.SetAttributes(attribute.String("tool_name", toolName))
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
		s.instrumentation.ToolInvoke.Add(
			r.Context(),
			1,
			metric.WithAttributes(attribute.String("toolbox.name", toolName)),
			metric.WithAttributes(attribute.String("toolbox.operation.status", status)),
		)
	}()

	tool, ok := s.ResourceMgr.GetTool(toolName)
	if !ok {
		err = fmt.Errorf("invalid tool name: tool with name %q does not exist", toolName)
		s.logger.DebugContext(ctx, err.Error())
		_ = render.Render(w, r, newErrResponse(err, http.StatusNotFound))
		return
	}

	// Extract OAuth access token from the "Authorization" header (currently for
	// BigQuery end-user credentials usage only)
	accessToken := tools.AccessToken(r.Header.Get("Authorization"))

	// Check if this specific tool requires the standard authorization header
	clientAuth, err := tool.RequiresClientAuthorization(s.ResourceMgr)
	if err != nil {
		errMsg := fmt.Errorf("error during invocation: %w", err)
		s.logger.DebugContext(ctx, errMsg.Error())
		_ = render.Render(w, r, newErrResponse(errMsg, http.StatusNotFound))
		return
	}
	if clientAuth {
		if accessToken == "" {
			err = fmt.Errorf("tool requires client authorization but access token is missing from the request header")
			s.logger.DebugContext(ctx, err.Error())
			_ = render.Render(w, r, newErrResponse(err, http.StatusUnauthorized))
			return
		}
	}

	// Tool authentication
	// claimsFromAuth maps the name of the authservice to the claims retrieved from it.
	claimsFromAuth := make(map[string]map[string]any)
	for _, aS := range s.ResourceMgr.GetAuthServiceMap() {
		claims, err := aS.GetClaimsFromHeader(ctx, r.Header)
		if err != nil {
			s.logger.DebugContext(ctx, err.Error())
			continue
		}
		if claims == nil {
			// authService not present in header
			continue
		}
		claimsFromAuth[aS.GetName()] = claims
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
		err = fmt.Errorf("tool invocation not authorized. Please make sure your specify correct auth headers")
		s.logger.DebugContext(ctx, err.Error())
		_ = render.Render(w, r, newErrResponse(err, http.StatusUnauthorized))
		return
	}
	s.logger.DebugContext(ctx, "tool invocation authorized")

	var data map[string]any
	if err = util.DecodeJSON(r.Body, &data); err != nil {
		render.Status(r, http.StatusBadRequest)
		err = fmt.Errorf("request body was invalid JSON: %w", err)
		s.logger.DebugContext(ctx, err.Error())
		_ = render.Render(w, r, newErrResponse(err, http.StatusBadRequest))
		return
	}

	params, err := tool.ParseParams(data, claimsFromAuth)
	if err != nil {
		// If auth error, return 401
		if errors.Is(err, util.ErrUnauthorized) {
			s.logger.DebugContext(ctx, fmt.Sprintf("error parsing authenticated parameters from ID token: %s", err))
			_ = render.Render(w, r, newErrResponse(err, http.StatusUnauthorized))
			return
		}
		err = fmt.Errorf("provided parameters were invalid: %w", err)
		s.logger.DebugContext(ctx, err.Error())
		_ = render.Render(w, r, newErrResponse(err, http.StatusBadRequest))
		return
	}
	s.logger.DebugContext(ctx, fmt.Sprintf("invocation params: %s", params))

	res, err := tool.Invoke(ctx, s.ResourceMgr, params, accessToken)

	// Determine what error to return to the users.
	if err != nil {
		errStr := err.Error()
		var statusCode int

		// Upstream API auth error propagation
		switch {
		case strings.Contains(errStr, "Error 401"):
			statusCode = http.StatusUnauthorized
		case strings.Contains(errStr, "Error 403"):
			statusCode = http.StatusForbidden
		}

		if statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden {
			if clientAuth {
				// Propagate the original 401/403 error.
				s.logger.DebugContext(ctx, fmt.Sprintf("error invoking tool. Client credentials lack authorization to the source: %v", err))
				_ = render.Render(w, r, newErrResponse(err, statusCode))
				return
			}
			// ADC lacking permission or credentials configuration error.
			internalErr := fmt.Errorf("unexpected auth error occured during Tool invocation: %w", err)
			s.logger.ErrorContext(ctx, internalErr.Error())
			_ = render.Render(w, r, newErrResponse(internalErr, http.StatusInternalServerError))
			return
		}
		err = fmt.Errorf("error while invoking tool: %w", err)
		s.logger.DebugContext(ctx, err.Error())
		_ = render.Render(w, r, newErrResponse(err, http.StatusBadRequest))
		return
	}

	resMarshal, err := json.Marshal(res)
	if err != nil {
		err = fmt.Errorf("unable to marshal result: %w", err)
		s.logger.DebugContext(ctx, err.Error())
		_ = render.Render(w, r, newErrResponse(err, http.StatusInternalServerError))
		return
	}

	_ = render.Render(w, r, &resultResponse{Result: string(resMarshal)})
}

var _ render.Renderer = &resultResponse{} // Renderer interface for managing response payloads.

// resultResponse is the response sent back when the tool was invocated successfully.
type resultResponse struct {
	Result string `json:"result"` // result of tool invocation
}

// Render renders a single payload and respond to the client request.
func (rr resultResponse) Render(w http.ResponseWriter, r *http.Request) error {
	render.Status(r, http.StatusOK)
	return nil
}

var _ render.Renderer = &errResponse{} // Renderer interface for managing response payloads.

// newErrResponse is a helper function initializing an ErrResponse
func newErrResponse(err error, code int) *errResponse {
	return &errResponse{
		Err:            err,
		HTTPStatusCode: code,

		StatusText: http.StatusText(code),
		ErrorText:  err.Error(),
	}
}

// errResponse is the response sent back when an error has been encountered.
type errResponse struct {
	Err            error `json:"-"` // low-level runtime error
	HTTPStatusCode int   `json:"-"` // http response status code

	StatusText string `json:"status"`          // user-level status message
	ErrorText  string `json:"error,omitempty"` // application-level error message, for debugging
}

func (e *errResponse) Render(w http.ResponseWriter, r *http.Request) error {
	render.Status(r, e.HTTPStatusCode)
	return nil
}
