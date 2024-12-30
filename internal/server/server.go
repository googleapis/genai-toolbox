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
	"github.com/googleapis/genai-toolbox/internal/log"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/tools"
)

// Server contains info for running an instance of Toolbox. Should be instantiated with NewServer().
type Server struct {
	version  string
	srv      *http.Server
	listener net.Listener
	root     chi.Router
	logger   log.Logger

	sources     map[string]sources.Source
	authSources map[string]auth.AuthSource
	tools       map[string]tools.Tool
	toolsets    map[string]tools.Toolset
}

// NewServer returns a Server object based on provided Config.
func NewServer(ctx context.Context, cfg ServerConfig, l log.Logger) (*Server, error) {
	var s *Server

	// set up http serving
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	// logging
	logLevel, err := log.SeverityToLevel(cfg.LogLevel.String())
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
	case "standard":
		httpOpts = httplog.Options{
			LogLevel:         logLevel,
			Concise:          true,
			RequestHeaders:   true,
			MessageFieldName: "message",
		}
	default:
		return nil, fmt.Errorf("invalid Logging format: %q", cfg.LoggingFormat.String())
	}
	httpLogger := httplog.NewLogger("httplog", httpOpts)
	r.Use(httplog.RequestLogger(httpLogger))
	// control plane
	apiR, err := apiRouter(s)
	if err != nil {
		return nil, err
	}
	r.Mount("/api", apiR)
	// default endpoint for validating server is running
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("🧰 Hello world! 🧰"))
	})

	// initialize and validate the sources from configs
	sourcesMap := make(map[string]sources.Source)
	for name, sc := range cfg.SourceConfigs {
		s, err := sc.Initialize()
		if err != nil {
			return nil, fmt.Errorf("unable to initialize source %q: %w", name, err)
		}
		sourcesMap[name] = s
	}
	l.InfoContext(ctx, fmt.Sprintf("Initialized %d sources.", len(sourcesMap)))

	// initialize and validate the auth sources from configs
	authSourcesMap := make(map[string]auth.AuthSource)
	for name, sc := range cfg.AuthSourceConfigs {
		a, err := sc.Initialize()
		if err != nil {
			return nil, fmt.Errorf("unable to initialize auth source %q: %w", name, err)
		}
		authSourcesMap[name] = a
	}
	l.InfoContext(ctx, fmt.Sprintf("Initialized %d authSources.", len(authSourcesMap)))

	// initialize and validate the tools from configs
	toolsMap := make(map[string]tools.Tool)
	for name, tc := range cfg.ToolConfigs {
		t, err := tc.Initialize(sourcesMap)
		if err != nil {
			return nil, fmt.Errorf("unable to initialize tool %q: %w", name, err)
		}
		toolsMap[name] = t
	}
	l.InfoContext(ctx, fmt.Sprintf("Initialized %d tools.", len(toolsMap)))

	// create a default toolset that contains all tools
	allToolNames := make([]string, 0, len(toolsMap))
	for name := range toolsMap {
		allToolNames = append(allToolNames, name)
	}
	if cfg.ToolsetConfigs == nil {
		cfg.ToolsetConfigs = make(ToolsetConfigs)
	}
	cfg.ToolsetConfigs[""] = tools.ToolsetConfig{Name: "", ToolNames: allToolNames}

	// initialize and validate the toolsets from configs
	toolsetsMap := make(map[string]tools.Toolset)
	for name, tc := range cfg.ToolsetConfigs {
		t, err := tc.Initialize(cfg.Version, toolsMap)
		if err != nil {
			return nil, fmt.Errorf("unable to initialize toolset %q: %w", name, err)
		}
		toolsetsMap[name] = t
	}
	l.InfoContext(ctx, fmt.Sprintf("Initialized %d toolsets.", len(toolsetsMap)))

	addr := net.JoinHostPort(cfg.Address, strconv.Itoa(cfg.Port))
	srv := &http.Server{Addr: addr, Handler: r}

	s = &Server{
		srv:         srv,
		root:        r,
		logger:      l,
		sources:     sourcesMap,
		authSources: authSourcesMap,
		tools:       toolsMap,
		toolsets:    toolsetsMap,
	}

	return s, nil
}

// Listen starts a listener for the given Server instance.
func (s *Server) Listen(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	if s.listener != nil {
		return fmt.Errorf("server is already listening: %s", s.listener.Addr().String())
	}
	lc := net.ListenConfig{KeepAlive: 30 * time.Second}
	var err error
	if s.listener, err = lc.Listen(ctx, "tcp", s.srv.Addr); err != nil {
		return fmt.Errorf("failed to open listener for %q: %w", s.srv.Addr, err)
	}
	return nil
}

// Serve starts an HTTP server for the given Server instance.
func (s *Server) Serve() error {
	return s.srv.Serve(s.listener)
}

// Shutdown gracefully shuts down the server without interrupting any active
// connections. It uses http.Server.Shutdown() and has the same functionality.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.srv.Shutdown(ctx)
}
