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
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	yaml "github.com/goccy/go-yaml"
)

type SourceInfo struct {
	Name   string         `json:"name"`
	Kind   string         `json:"kind"`
	Config map[string]any `json:"config,omitempty"`
}

type SourceListResponse struct {
	Sources map[string]SourceInfo `json:"sources"`
}

// sourceListHandler handles requests for listing all sources.
func sourceListHandler(s *Server, w http.ResponseWriter, r *http.Request) {
	ctx, span := s.instrumentation.Tracer.Start(r.Context(), "toolbox/server/source/list")
	r = r.WithContext(ctx)
	defer span.End()

	sourcesMap := s.ResourceMgr.GetSourcesMap()
	resp := SourceListResponse{
		Sources: make(map[string]SourceInfo, len(sourcesMap)),
	}
	for name, source := range sourcesMap {
		resp.Sources[name] = SourceInfo{
			Name: name,
			Kind: source.SourceKind(),
		}
	}
	render.JSON(w, r, resp)
}

// sourceGetHandler handles requests for a single source.
func sourceGetHandler(s *Server, w http.ResponseWriter, r *http.Request) {
	ctx, span := s.instrumentation.Tracer.Start(r.Context(), "toolbox/server/source/get")
	r = r.WithContext(ctx)
	defer span.End()

	sourceName := chi.URLParam(r, "sourceName")
	source, ok := s.ResourceMgr.GetSource(sourceName)
	if !ok {
		err := fmt.Errorf("source %q does not exist", sourceName)
		s.logger.DebugContext(ctx, err.Error())
		_ = render.Render(w, r, newErrResponse(err, http.StatusNotFound))
		return
	}
	configMap, err := sourceConfigToMap(source.ToConfig())
	if err != nil {
		errMsg := fmt.Errorf("unable to serialize source %q config: %w", sourceName, err)
		s.logger.DebugContext(ctx, errMsg.Error())
		_ = render.Render(w, r, newErrResponse(errMsg, http.StatusInternalServerError))
		return
	}
	redactSensitiveValues(configMap)
	resp := SourceListResponse{
		Sources: map[string]SourceInfo{
			sourceName: {
				Name:   sourceName,
				Kind:   source.SourceKind(),
				Config: configMap,
			},
		},
	}
	render.JSON(w, r, resp)
}

func sourceConfigToMap(cfg any) (map[string]any, error) {
	raw, err := yaml.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	var configMap map[string]any
	if err := yaml.Unmarshal(raw, &configMap); err != nil {
		return nil, err
	}
	return configMap, nil
}

func redactSensitiveValues(v any) {
	switch typed := v.(type) {
	case map[string]any:
		for k, val := range typed {
			if isSensitiveKey(k) {
				typed[k] = "[REDACTED]"
				continue
			}
			redactSensitiveValues(val)
		}
	case []any:
		for i := range typed {
			redactSensitiveValues(typed[i])
		}
	}
}

func isSensitiveKey(key string) bool {
	lower := strings.ToLower(key)
	sensitive := []string{"password", "secret", "token", "key", "credential"}
	for _, keyword := range sensitive {
		if strings.Contains(lower, keyword) {
			return true
		}
	}
	return false
}
