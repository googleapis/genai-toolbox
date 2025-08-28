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
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

// adminRouter creates a router that represents the routes under /admin
func adminRouter(s *Server) (chi.Router, error) {
	r := chi.NewRouter()

	r.Use(middleware.AllowContentType("application/json"))
	r.Use(middleware.StripSlashes)
	r.Use(render.SetContentType(render.ContentTypeJSON))

	r.Get("/{resource}", func(w http.ResponseWriter, r *http.Request) { adminGetHandler(s, w, r) })

	return r, nil
}

// adminGetHandler handles requests for a list of specific resource
func adminGetHandler(s *Server, w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	resource := chi.URLParam(r, "resource")

	var resourceList []string
	switch resource {
	case "source":
		sourcesMap := s.ResourceMgr.GetSourcesMap()
		for n := range sourcesMap {
			resourceList = append(resourceList, n)
		}
	case "authservice":
		authServicesMap := s.ResourceMgr.GetAuthServiceMap()
		for n := range authServicesMap {
			resourceList = append(resourceList, n)
		}
	case "tool":
		toolsMap := s.ResourceMgr.GetToolsMap()
		for n := range toolsMap {
			resourceList = append(resourceList, n)
		}
	case "toolset":
		toolsetsMap := s.ResourceMgr.GetToolsetsMap()
		for n := range toolsetsMap {
			resourceList = append(resourceList, n)
		}
	default:
		err := fmt.Errorf(`invalid resource %s, please provide one of "source", "authservice", "tool", or "toolset"`, resource)
		s.logger.DebugContext(ctx, err.Error())
		_ = render.Render(w, r, newErrResponse(err, http.StatusNotFound))
        return
	}

    render.JSON(w, r, resourceList)
}
