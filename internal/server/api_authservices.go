// Copyright 2026 Google LLC
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
	"slices"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/googleapis/genai-toolbox/internal/tools"
)

type AuthServiceInfo struct {
	Name       string   `json:"name"`
	Kind       string   `json:"kind"`
	HeaderName string   `json:"headerName"`
	Tools      []string `json:"tools"`
}

type AuthServiceListResponse struct {
	AuthServices map[string]AuthServiceInfo `json:"authServices"`
}

// authServiceListHandler handles requests for listing all auth services.
func authServiceListHandler(s *Server, w http.ResponseWriter, r *http.Request) {
	ctx, span := s.instrumentation.Tracer.Start(r.Context(), "toolbox/server/authservice/list")
	r = r.WithContext(ctx)
	defer span.End()

	authServicesMap := s.ResourceMgr.GetAuthServiceMap()
	usageByAuthService := authServiceToolUsage(s.ResourceMgr.GetToolsMap())
	resp := AuthServiceListResponse{
		AuthServices: make(map[string]AuthServiceInfo, len(authServicesMap)),
	}
	for name, authService := range authServicesMap {
		resp.AuthServices[name] = AuthServiceInfo{
			Name:       authService.GetName(),
			Kind:       authService.AuthServiceKind(),
			HeaderName: authService.GetName(),
			Tools:      usageByAuthService[name],
		}
	}
	render.JSON(w, r, resp)
}

// authServiceGetHandler handles requests for a single auth service.
func authServiceGetHandler(s *Server, w http.ResponseWriter, r *http.Request) {
	ctx, span := s.instrumentation.Tracer.Start(r.Context(), "toolbox/server/authservice/get")
	r = r.WithContext(ctx)
	defer span.End()

	authServiceName := chi.URLParam(r, "authServiceName")
	authService, ok := s.ResourceMgr.GetAuthService(authServiceName)
	if !ok {
		err := fmt.Errorf("auth service %q does not exist", authServiceName)
		s.logger.DebugContext(ctx, err.Error())
		_ = render.Render(w, r, newErrResponse(err, http.StatusNotFound))
		return
	}
	usageByAuthService := authServiceToolUsage(s.ResourceMgr.GetToolsMap())
	resp := AuthServiceListResponse{
		AuthServices: map[string]AuthServiceInfo{
			authServiceName: {
				Name:       authService.GetName(),
				Kind:       authService.AuthServiceKind(),
				HeaderName: authService.GetName(),
				Tools:      usageByAuthService[authServiceName],
			},
		},
	}
	render.JSON(w, r, resp)
}

func authServiceToolUsage(toolsMap map[string]tools.Tool) map[string][]string {
	usage := make(map[string]map[string]bool)

	for toolName, tool := range toolsMap {
		manifest := tool.Manifest()
		for _, authName := range manifest.AuthRequired {
			addAuthServiceUsage(usage, authName, toolName)
		}
		for _, param := range manifest.Parameters {
			for _, authName := range param.AuthServices {
				addAuthServiceUsage(usage, authName, toolName)
			}
		}
	}

	out := make(map[string][]string, len(usage))
	for authName, toolsSet := range usage {
		toolsList := make([]string, 0, len(toolsSet))
		for toolName := range toolsSet {
			toolsList = append(toolsList, toolName)
		}
		slices.Sort(toolsList)
		out[authName] = toolsList
	}
	return out
}

func addAuthServiceUsage(usage map[string]map[string]bool, authName, toolName string) {
	if authName == "" {
		return
	}
	if usage[authName] == nil {
		usage[authName] = make(map[string]bool)
	}
	usage[authName][toolName] = true
}
