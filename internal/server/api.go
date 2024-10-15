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
	toolsManifest := make(map[string]tools.ToolManifest)
	for _, name := range c.ToolNames {
		manifest := s.conf.ToolConfigs[name].Manifest()
		toolsManifest[name] = manifest
	}
	return tools.ToolsetManifest{ServerVersion: s.conf.Version, ToolsManifest: toolsManifest}
}

// apiRouter creates a router that represents the routes under /api
func apiRouter(s *Server) (chi.Router, error) {
	r := chi.NewRouter()

	r.Get("/toolset/{toolsetName}", toolsetHandler(s))

	r.Route("/tool/{toolName}", func(r chi.Router) {
		r.Use(middleware.AllowContentType("application/json"))
		r.Post("/", toolHandler(s))
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

func toolHandler(s *Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		toolName := chi.URLParam(r, "toolName")
		tool, ok := s.tools[toolName]
		if !ok {
			render.Status(r, http.StatusInternalServerError)
			_, _ = w.Write([]byte(fmt.Sprintf("Tool %q does not exist", toolName)))
			return
		}

		var data map[string]interface{}
		if err := render.DecodeJSON(r.Body, &data); err != nil {
			render.Status(r, http.StatusBadRequest)
			return
		}

		params, err := tool.ParseParams(data)
		if err != nil {
			render.Status(r, http.StatusBadRequest)
			// TODO: More robust error formatting (probably JSON)
			render.PlainText(w, r, err.Error())
			return
		}

		res, err := tool.Invoke(params)
		if err != nil {
			render.Status(r, http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte(fmt.Sprintf("Tool Result: %s", res)))
	}
}
