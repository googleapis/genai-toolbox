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
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

// apiRouter creates a router that represents the routes under /api
func apiRouter(s *Server) (chi.Router, error) {
	r := chi.NewRouter()

	r.Use(middleware.AllowContentType("application/json"))
	r.Use(middleware.StripSlashes)
	r.Use(render.SetContentType(render.ContentTypeJSON))

	r.Get("/authservice", func(w http.ResponseWriter, r *http.Request) { authServiceListHandler(s, w, r) })
	r.Get("/authservice/{authServiceName}", func(w http.ResponseWriter, r *http.Request) { authServiceGetHandler(s, w, r) })

	r.Get("/source", func(w http.ResponseWriter, r *http.Request) { sourceListHandler(s, w, r) })
	r.Get("/source/{sourceName}", func(w http.ResponseWriter, r *http.Request) { sourceGetHandler(s, w, r) })

	r.Get("/toolset", func(w http.ResponseWriter, r *http.Request) { toolsetHandler(s, w, r) })
	r.Get("/toolset/{toolsetName}", func(w http.ResponseWriter, r *http.Request) { toolsetHandler(s, w, r) })

	r.Route("/tool/{toolName}", func(r chi.Router) {
		r.Get("/", func(w http.ResponseWriter, r *http.Request) { toolGetHandler(s, w, r) })
		r.Post("/invoke", func(w http.ResponseWriter, r *http.Request) { toolInvokeHandler(s, w, r) })
	})

	return r, nil
}
