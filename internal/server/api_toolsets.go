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

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
)

// toolsetHandler handles the request for information about a Toolset.
func toolsetHandler(s *Server, w http.ResponseWriter, r *http.Request) {
	ctx, span := s.instrumentation.Tracer.Start(r.Context(), "toolbox/server/toolset/get")
	r = r.WithContext(ctx)

	toolsetName := chi.URLParam(r, "toolsetName")
	s.logger.DebugContext(ctx, fmt.Sprintf("toolset name: %s", toolsetName))
	span.SetAttributes(attribute.String("toolset_name", toolsetName))
	var err error
	defer func() {
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()

		status := "success"
		if err != nil {
			status = "error"
		}
		s.instrumentation.ToolsetGet.Add(
			r.Context(),
			1,
			metric.WithAttributes(attribute.String("toolbox.name", toolsetName)),
			metric.WithAttributes(attribute.String("toolbox.operation.status", status)),
		)
	}()

	toolset, ok := s.ResourceMgr.GetToolset(toolsetName)
	if !ok {
		err = fmt.Errorf("toolset %q does not exist", toolsetName)
		s.logger.DebugContext(ctx, err.Error())
		_ = render.Render(w, r, newErrResponse(err, http.StatusNotFound))
		return
	}
	render.JSON(w, r, toolset.Manifest)
}
