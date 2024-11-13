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
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/googleapis/genai-toolbox/internal/tools"
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

// toolInvokeHandler handles the request for information about a Toolset.
func toolsetHandler(s *Server, w http.ResponseWriter, r *http.Request) {
	toolsetName := chi.URLParam(r, "toolsetName")
	toolset, ok := s.toolsets[toolsetName]
	if !ok {
		err := fmt.Errorf("Toolset %q does not exist", toolsetName)
		_ = render.Render(w, r, newErrResponse(err, http.StatusNotFound))
		return
	}
	render.JSON(w, r, toolset.Manifest)
}

// toolGetHandler handles requests for a single Tool.
func toolGetHandler(s *Server, w http.ResponseWriter, r *http.Request) {
	toolName := chi.URLParam(r, "toolName")
	tool, ok := s.tools[toolName]
	if !ok {
		err := fmt.Errorf("invalid tool name: tool with name %q does not exist", toolName)
		_ = render.Render(w, r, newErrResponse(err, http.StatusNotFound))
		return
	}
	// TODO: this can be optimized later with some caching
	m := tools.ToolsetManifest{
		ServerVersion: s.conf.Version,
		ToolsManifest: map[string]tools.Manifest{
			toolName: tool.Manifest(),
		},
	}

	render.JSON(w, r, m)
}

// toolInvokeHandler handles the API request to invoke a specific Tool.
func toolInvokeHandler(s *Server, w http.ResponseWriter, r *http.Request) {
	toolName := chi.URLParam(r, "toolName")
	tool, ok := s.tools[toolName]
	if !ok {
		err := fmt.Errorf("invalid tool name: tool with name %q does not exist", toolName)
		_ = render.Render(w, r, newErrResponse(err, http.StatusNotFound))
		return
	}

	// Tool authentication
	// claimsFromAuth maps the name of the authsource to the claims retrieved from it.
	claimsFromAuth := make(map[string]map[string]any)
	for _, aS := range s.authSources {
		claims, err := aS.GetClaimsFromHeader(r.Header)
		if err != nil {
			err := fmt.Errorf("failure getting claims from header: %w", err)
			_ = render.Render(w, r, newErrResponse(err, http.StatusBadRequest))
			return
		}
		if claims == nil {
			// authSource not present in header
			continue
		}
		claimsFromAuth[aS.GetName()] = claims
	}

	// Tool authorization check
	verifiedAuthSources := make([]string, len(claimsFromAuth))
	i := 0
	for k := range claimsFromAuth {
		verifiedAuthSources[i] = k
		i++
	}
	// Check if any of the specified auth sources is verified
	isAuthorized := tool.Authorized(verifiedAuthSources)
	if !isAuthorized {
		err := fmt.Errorf("tool invocation not authorized. Please make sure your specify correct auth headers")
		_ = render.Render(w, r, newErrResponse(err, http.StatusUnauthorized))
		return
	}

	var data map[string]any
	if err := render.DecodeJSON(r.Body, &data); err != nil {
		render.Status(r, http.StatusBadRequest)
		err := fmt.Errorf("request body was invalid JSON: %w", err)
		_ = render.Render(w, r, newErrResponse(err, http.StatusBadRequest))
		return
	}

	params, err := tool.ParseParams(data, claimsFromAuth)
	if err != nil {
		err := fmt.Errorf("provided parameters were invalid: %w", err)
		_ = render.Render(w, r, newErrResponse(err, http.StatusBadRequest))
		return
	}

	res, err := tool.Invoke(params)
	if err != nil {
		err := fmt.Errorf("error while invoking tool: %w", err)
		_ = render.Render(w, r, newErrResponse(err, http.StatusInternalServerError))
		return
	}

	_ = render.Render(w, r, &resultResponse{Result: res})
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

// newErrResponse is a helper function initalizing an ErrResponse
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
