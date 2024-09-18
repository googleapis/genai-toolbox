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
	"github.com/go-chi/render"
)

// apiRouter creates a router that represents the routes under /api
func apiRouter(s *Server) chi.Router {
	r := chi.NewRouter()

	r.Get("/toolset/{toolsetName}", toolsetHandler(s))

	// TOOD: make this POST
	r.Get("/tool/{toolName}", toolHandler(s))

	return r
}

func toolsetHandler(s *Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		toolsetName := chi.URLParam(r, "toolsetName")
		w.Write([]byte(fmt.Sprintf("Stub for toolset %s manifest!", toolsetName)))
	}
}

func toolHandler(s *Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		toolName := chi.URLParam(r, "toolName")
		tool, ok := s.tools[toolName]
		if !ok {
			render.Status(r, http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf("Tool %q does not exist", toolName)))
			return
		}

		res, err := tool.Invoke()
		if err != nil {
			render.Status(r, http.StatusInternalServerError)
			return
		}
		w.Write([]byte(fmt.Sprintf("Tool Result: %s", res)))
	}
}
