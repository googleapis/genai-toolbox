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
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/googleapis/genai-toolbox/internal/tools"
)

func createToolsetManifest(s *Server, c tools.ToolsetConfig) tools.ToolsetManifest {
	toolsManifest := make([]tools.ToolManifest, 0, len(c.ToolNames))
	for _, name := range c.ToolNames {
		manifest := s.conf.ToolConfigs[name].Manifest()
		toolsManifest = append(toolsManifest, manifest)
	}
	return tools.ToolsetManifest{ServerVersion: s.conf.Version, ToolsManifest: toolsManifest}
}

// apiRouter creates a router that represents the routes under /api
func apiRouter(s *Server) (chi.Router, error) {
	r := chi.NewRouter()

	r.Get("/toolset/{toolsetName}", toolsetHandler(s))

	r.Route("/tool/{toolName}", func(r chi.Router) {
		r.Use(middleware.AllowContentType("application/json"))
		r.Post("/invoke", newToolHandler(s))
	})

	// Convert tool configs to JSON for manifest
	for name, c := range s.conf.ToolsetConfigs {
		manifest, err := json.Marshal(createToolsetManifest(s, c))
		if err != nil {
			return nil, fmt.Errorf("unable to marshal toolset: %w", err)
		}
		s.manifests[name] = manifest
	}
	return r, nil
}

func toolsetHandler(s *Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		toolsetName := chi.URLParam(r, "toolsetName")
		manifest, ok := s.manifests[toolsetName]
		if !ok {
			http.Error(w, fmt.Sprintf("Toolset %q does not exist", toolsetName), http.StatusNotFound)
			return
		}
		_, _ = w.Write([]byte(manifest))

	}
}

// resultResponse is the response sent back when the tool was invocated succesffully.
type resultResponse struct {
	Result string `json:"result"` // result of tool invocation
}

func (rr resultResponse) Render(w http.ResponseWriter, r *http.Request) error {
	render.Status(r, http.StatusOK)
	return nil
}

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

func newToolHandler(s *Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		toolName := chi.URLParam(r, "toolName")
		tool, ok := s.tools[toolName]
		if !ok {
			err := fmt.Errorf("Invalid tool name. Tool with name %q does not exist", toolName)
			_ = render.Render(w, r, newErrResponse(err, http.StatusNotFound))
			return
		}

		var data map[string]interface{}
		if err := render.DecodeJSON(r.Body, &data); err != nil {
			render.Status(r, http.StatusBadRequest)
			err := fmt.Errorf("Request body was invalid JSON: %w", err)
			_ = render.Render(w, r, newErrResponse(err, http.StatusBadRequest))
			return
		}

		params, err := tool.ParseParams(data)
		if err != nil {
			err := fmt.Errorf("Provided parameters were invalid: %w", err)
			_ = render.Render(w, r, newErrResponse(err, http.StatusBadRequest))
			return
		}

		res, err := tool.Invoke(params)
		if err != nil {
			err := fmt.Errorf("Error while invoking tool: %w", err)
			_ = render.Render(w, r, newErrResponse(err, http.StatusInternalServerError))
			return
		}

		_ = render.Render(w, r, &resultResponse{Result: res})
	}
}
