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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/googleapis/genai-toolbox/internal/util"
)

// adminRouter creates a router that represents the routes under /admin
func adminRouter(s *Server) (chi.Router, error) {
	r := chi.NewRouter()

	r.Use(middleware.AllowContentType("application/json", "application/json-rpc", "application/jsonrequest"))
	r.Use(middleware.StripSlashes)
	r.Use(render.SetContentType(render.ContentTypeJSON))

	r.Put("/{kind}/{name}", func(w http.ResponseWriter, r *http.Request) { createOrUpdatePrimitives(s, w, r) })

	return r, nil
}

// createOrUpdatePrimitives handles the creation or updating of primitives
// changing name will result in creation of a new primitive instead of replacing
// existing primitive
// Invalid primitive kind will result in http.StatusBadRequest
// Other errors will result in http.StatusInternalServerError
func createOrUpdatePrimitives(s *Server, w http.ResponseWriter, r *http.Request) {
	kind := chi.URLParam(r, "kind")
	name := chi.URLParam(r, "name")
	ctx := r.Context()
	ctx = util.WithInstrumentation(ctx, s.instrumentation)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		err = fmt.Errorf("unable to read request body: %w", err)
		s.logger.DebugContext(ctx, err.Error())
		_ = render.Render(w, r, newErrResponse(err, http.StatusBadRequest))
		return
	}
	var req struct {
		Config map[string]any `json:"config"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		err = fmt.Errorf("unable to unmarshal request body: %w", err)
		s.logger.DebugContext(ctx, err.Error())
		_ = render.Render(w, r, newErrResponse(err, http.StatusBadRequest))
		return
	}

	// Attempt to create or update the primitive based on its kind and name
	var updateErr error
	switch strings.ToLower(kind) {
	case "source":
		c, err := UnmarshalYAMLSourceConfig(ctx, name, req.Config)
		if err != nil {
			updateErr = fmt.Errorf("unable to unmarshal source config: %w", err)
			break
		}
		updateErr = s.ResourceMgr.UpdateSource(ctx, name, c)
	case "authservice":
		c, err := UnmarshalYAMLAuthServiceConfig(ctx, name, req.Config)
		if err != nil {
			updateErr = fmt.Errorf("unable to unmarshal auth service config: %w", err)
			break
		}
		updateErr = s.ResourceMgr.UpdateAuthService(ctx, name, c)
	case "embeddingmodel":
		c, err := UnmarshalYAMLEmbeddingModelConfig(ctx, name, req.Config)
		if err != nil {
			updateErr = fmt.Errorf("unable to unmarshal embedding model config: %w", err)
			break
		}
		updateErr = s.ResourceMgr.UpdateEmbeddingModel(ctx, name, c)
	case "tool":
		c, err := UnmarshalYAMLToolConfig(ctx, name, req.Config)
		if err != nil {
			updateErr = fmt.Errorf("unable to unmarshal tool config: %w", err)
			break
		}
		updateErr = s.ResourceMgr.UpdateTool(ctx, name, c)
	case "toolset":
		c, err := UnmarshalYAMLToolsetConfig(ctx, name, req.Config)
		if err != nil {
			updateErr = fmt.Errorf("unable to unmarshal toolset config: %w", err)
			break
		}
		updateErr = s.ResourceMgr.UpdateToolset(ctx, name, c, s.version)
	case "prompt":
		c, err := UnmarshalYAMLPromptConfig(ctx, name, req.Config)
		if err != nil {
			updateErr = fmt.Errorf("unable to unmarshal prompt config: %w", err)
			break
		}
		updateErr = s.ResourceMgr.UpdatePrompt(ctx, name, c)
	default:
		err = fmt.Errorf("invalid primitive kind provided")
		s.logger.DebugContext(ctx, err.Error())
		_ = render.Render(w, r, newErrResponse(err, http.StatusBadRequest))
		return
	}
	if updateErr != nil {
		s.logger.DebugContext(ctx, updateErr.Error())
		_ = render.Render(w, r, newErrResponse(updateErr, http.StatusInternalServerError))
		return
	}
}
