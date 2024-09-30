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
	"github.com/go-chi/render"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/toolsets"
)

// apiRouter creates a router that represents the routes under /api
func apiRouter(s *Server) chi.Router {
	r := chi.NewRouter()

	r.Get("/toolset/{toolsetName}", toolsetHandler(s))

	// TODO: make this POST
	r.Get("/tool/{toolName}", toolHandler(s))

	return r
}

func toolsetHandler(s *Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		toolsetName := chi.URLParam(r, "toolsetName")
		if toolsetName == "" {
			// return all the tools if no toolset name is provided
			var toolsetManifest toolsets.ToolsetManifest
			toolsetManifest.Tools = make(map[string]tools.ToolManifest)
			toolsetManifest.ServerVersion = s.version
			for name, tool := range s.tools {
				toolManifest, err := tool.Describe()
				if err != nil {
					fmt.Errorf("Error describing tool %s", err)
					return
				}
				toolsetManifest.Tools[name] = toolManifest
			}
			json.NewEncoder(w).Encode(toolsetManifest)
		} else {
			// Describe the tools of the request toolset
			var toolsetManifest toolsets.ToolsetManifest
			toolsetManifest.Tools = make(map[string]tools.ToolManifest)
			toolsetManifest.ServerVersion = s.version
			if toolset, ok := s.toolsets[toolsetName]; ok {
				toolset.Describe()
				json.NewEncoder(w).Encode(toolsetManifest)
			}
			fmt.Errorf("toolset not found: %s", toolsetName)
		}
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

		res, err := tool.Invoke()
		if err != nil {
			render.Status(r, http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte(fmt.Sprintf("Tool Result: %s", res)))
	}
}
