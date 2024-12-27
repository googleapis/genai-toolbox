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
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httplog/v2"
	"github.com/googleapis/genai-toolbox/internal/auth"
	logLib "github.com/googleapis/genai-toolbox/internal/log"
	"github.com/googleapis/genai-toolbox/internal/sources"
	telemetrytrace "github.com/googleapis/genai-toolbox/internal/telemetry/trace"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// Server contains info for running an instance of Toolbox. Should be instantiated with NewServer().
type Server struct {
	conf   ServerConfig
	root   chi.Router
	logger logLib.Logger
	metric *telemetry.Metric

	sources     map[string]sources.Source
	authSources map[string]auth.AuthSource
	tools       map[string]tools.Tool
	toolsets    map[string]tools.Toolset
}

// NewServer returns a Server object based on provided Config.
func NewServer(cfg ServerConfig, log logLib.Logger) (*Server, error) {
	ctx, span := telemetrytrace.Tracer().Start(context.Background(), "toolbox/server/init")
	defer span.End()

	logLevel, err := logLib.SeverityToLevel(cfg.LogLevel.String())
	if err != nil {
		return nil, fmt.Errorf("unable to initialize http log: %w", err)
	}
	var httpOpts httplog.Options
	switch cfg.LoggingFormat.String() {
	case "json":
		httpOpts = httplog.Options{
			JSON:             true,
			LogLevel:         logLevel,
			Concise:          true,
			RequestHeaders:   true,
			MessageFieldName: "message",
			SourceFieldName:  "logging.googleapis.com/sourceLocation",
			TimeFieldName:    "timestamp",
			LevelFieldName:   "severity",
		}
	default:
		httpOpts = httplog.Options{
			LogLevel:         logLevel,
			Concise:          true,
			RequestHeaders:   true,
			MessageFieldName: "message",
		}
	}

	logger := httplog.NewLogger("httplog", httpOpts)
	r := chi.NewRouter()
	r.Use(httplog.RequestLogger(logger))
	r.Use(middleware.Recoverer)

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("🧰 Hello world! 🧰"))
	})

	// initialize and validate the sources
	sourcesMap := make(map[string]sources.Source)
	for name, sc := range cfg.SourceConfigs {
		s, err := func() (sources.Source, error) {
			ctx, span := telemetrytrace.Tracer().Start(
				ctx,
				"toolbox/server/source/init",
				trace.WithAttributes(attribute.String("source_kind", sc.SourceConfigKind())),
				trace.WithAttributes(attribute.String("source_name", name)),
			)
			defer span.End()
			s, err := sc.Initialize(ctx)
			if err != nil {
				return nil, fmt.Errorf("unable to initialize source %q: %w", name, err)
			}
			return s, nil
		}()
		if err != nil {
			return nil, err
		}
		sourcesMap[name] = s
	}
	log.Info(fmt.Sprintf("Initialized %d sources.", len(sourcesMap)))

	// initialize and validate the auth sources
	authSourcesMap := make(map[string]auth.AuthSource)
	for name, sc := range cfg.AuthSourceConfigs {
		a, err := sc.Initialize()
		if err != nil {
			return nil, fmt.Errorf("unable to initialize auth source %q: %w", name, err)
		}
		authSourcesMap[name] = a
	}
	log.Info(fmt.Sprintf("Initialized %d authSources.", len(authSourcesMap)))

	// initialize and validate the tools
	toolsMap := make(map[string]tools.Tool)
	for name, tc := range cfg.ToolConfigs {
		t, err := func() (tools.Tool, error) {
			var span trace.Span
			ctx, span = telemetrytrace.Tracer().Start(
				ctx,
				"toolbox/server/tool/init",
				trace.WithAttributes(attribute.String("tool_kind", tc.ToolConfigKind())),
				trace.WithAttributes(attribute.String("tool_name", name)),
			)
			defer span.End()
			t, err := tc.Initialize(sourcesMap)
			if err != nil {
				return nil, fmt.Errorf("unable to initialize tool %q: %w", name, err)
			}
			return t, nil
		}()
		if err != nil {
			return nil, err
		}
		toolsMap[name] = t
	}
	log.Info(fmt.Sprintf("Initialized %d tools.", len(toolsMap)))

	// create a default toolset that contains all tools
	allToolNames := make([]string, 0, len(toolsMap))
	for name := range toolsMap {
		allToolNames = append(allToolNames, name)
	}
	if cfg.ToolsetConfigs == nil {
		cfg.ToolsetConfigs = make(ToolsetConfigs)
	}
	cfg.ToolsetConfigs[""] = tools.ToolsetConfig{Name: "", ToolNames: allToolNames}
	// initialize and validate the toolsets
	toolsetsMap := make(map[string]tools.Toolset)
	for name, tc := range cfg.ToolsetConfigs {
		t, err := tc.Initialize(cfg.Version, toolsMap)
		if err != nil {
			return nil, fmt.Errorf("unable to initialize toolset %q: %w", name, err)
		}
		toolsetsMap[name] = t
	}
	log.Info(fmt.Sprintf("Initialized %d toolsets.", len(toolsetsMap)))

	s := &Server{
		conf:        cfg,
		root:        r,
		logger:      log,
		metric:      metric,
		sources:     sourcesMap,
		authSources: authSourcesMap,
		tools:       toolsMap,
		toolsets:    toolsetsMap,
	}

	if router, err := apiRouter(s); err != nil {
		return nil, err
	} else {
		r.Mount("/api", router)
	}

	return s, nil
}

// Listen starts a listener for the given Server instance.
func (s *Server) Listen(ctx context.Context) (net.Listener, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	addr := net.JoinHostPort(s.conf.Address, strconv.Itoa(s.conf.Port))
	lc := net.ListenConfig{KeepAlive: 30 * time.Second}
	l, err := lc.Listen(ctx, "tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to open listener for %q: %w", addr, err)
	}
	return l, nil
}

// Serve starts an HTTP server for the given Server instance.
func (s *Server) Serve(l net.Listener) error {
	return http.Serve(l, s.root)
}
