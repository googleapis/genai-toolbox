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
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	yaml "github.com/goccy/go-yaml"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httplog/v2"
	"github.com/googleapis/genai-toolbox/internal/auth"
	"github.com/googleapis/genai-toolbox/internal/log"
	"github.com/googleapis/genai-toolbox/internal/prompts"
	"github.com/googleapis/genai-toolbox/internal/server/resources"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/telemetry"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/util"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// Server contains info for running an instance of Toolbox. Should be instantiated with NewServer().
type Server struct {
	version         string
	srv             *http.Server
	listener        net.Listener
	root            chi.Router
	logger          log.Logger
	instrumentation *telemetry.Instrumentation
	sseManager      *sseManager
	ResourceMgr     *resources.ResourceManager
}

func InitializeConfigs(ctx context.Context, cfg ServerConfig) (
	map[string]sources.Source,
	map[string]auth.AuthService,
	map[string]tools.Tool,
	map[string]tools.Toolset,
	map[string]prompts.Prompt,
	map[string]prompts.Promptset,
	error,
) {
	ctx = util.WithUserAgent(ctx, cfg.Version)
	instrumentation, err := util.InstrumentationFromContext(ctx)
	if err != nil {
		panic(err)
	}

	l, err := util.LoggerFromContext(ctx)
	if err != nil {
		panic(err)
	}

	// initialize and validate the sources from configs
	sourcesMap := make(map[string]sources.Source)
	for name, sc := range cfg.SourceConfigs {
		s, err := func() (sources.Source, error) {
			childCtx, span := instrumentation.Tracer.Start(
				ctx,
				"toolbox/server/source/init",
				trace.WithAttributes(attribute.String("source_kind", sc.SourceConfigKind())),
				trace.WithAttributes(attribute.String("source_name", name)),
			)
			defer span.End()
			s, err := sc.Initialize(childCtx, instrumentation.Tracer)
			if err != nil {
				return nil, fmt.Errorf("unable to initialize source %q: %w", name, err)
			}
			return s, nil
		}()
		if err != nil {
			return nil, nil, nil, nil, nil, nil, err
		}
		sourcesMap[name] = s
	}
	sourceNames := make([]string, 0, len(sourcesMap))
	for name := range sourcesMap {
		sourceNames = append(sourceNames, name)
	}
	l.InfoContext(ctx, fmt.Sprintf("Initialized %d sources: %s", len(sourcesMap), strings.Join(sourceNames, ", ")))

	// initialize and validate the auth services from configs
	authServicesMap := make(map[string]auth.AuthService)
	for name, sc := range cfg.AuthServiceConfigs {
		a, err := func() (auth.AuthService, error) {
			_, span := instrumentation.Tracer.Start(
				ctx,
				"toolbox/server/auth/init",
				trace.WithAttributes(attribute.String("auth_kind", sc.AuthServiceConfigKind())),
				trace.WithAttributes(attribute.String("auth_name", name)),
			)
			defer span.End()
			a, err := sc.Initialize()
			if err != nil {
				return nil, fmt.Errorf("unable to initialize auth service %q: %w", name, err)
			}
			return a, nil
		}()
		if err != nil {
			return nil, nil, nil, nil, nil, nil, err
		}
		authServicesMap[name] = a
	}
	authServiceNames := make([]string, 0, len(authServicesMap))
	for name := range authServicesMap {
		authServiceNames = append(authServiceNames, name)
	}
	l.InfoContext(ctx, fmt.Sprintf("Initialized %d authServices: %s", len(authServicesMap), strings.Join(authServiceNames, ", ")))

	// initialize and validate the tools from configs
	toolsMap := make(map[string]tools.Tool)
	for name, tc := range cfg.ToolConfigs {
		t, err := func() (tools.Tool, error) {
			_, span := instrumentation.Tracer.Start(
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
			return nil, nil, nil, nil, nil, nil, err
		}
		toolsMap[name] = t
	}
	toolNames := make([]string, 0, len(toolsMap))
	for name := range toolsMap {
		toolNames = append(toolNames, name)
	}
	l.InfoContext(ctx, fmt.Sprintf("Initialized %d tools: %s", len(toolsMap), strings.Join(toolNames, ", ")))

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
		t, err := func() (tools.Toolset, error) {
			_, span := instrumentation.Tracer.Start(
				ctx,
				"toolbox/server/toolset/init",
				trace.WithAttributes(attribute.String("toolset_name", name)),
			)
			defer span.End()
			t, err := tc.Initialize(cfg.Version, toolsMap)
			if err != nil {
				return tools.Toolset{}, fmt.Errorf("unable to initialize toolset %q: %w", name, err)
			}
			return t, err
		}()
		if err != nil {
			return nil, nil, nil, nil, nil, nil, err
		}
		toolsetsMap[name] = t
	}
	toolsetNames := make([]string, 0, len(toolsetsMap))
	for name := range toolsetsMap {
		if name == "" {
			toolsetNames = append(toolsetNames, "default")
		} else {
			toolsetNames = append(toolsetNames, name)
		}
	}
	l.InfoContext(ctx, fmt.Sprintf("Initialized %d toolsets: %s", len(toolsetsMap), strings.Join(toolsetNames, ", ")))

	// initialize and validate the prompts from configs
	promptsMap := make(map[string]prompts.Prompt)
	for name, pc := range cfg.PromptConfigs {
		p, err := func() (prompts.Prompt, error) {
			_, span := instrumentation.Tracer.Start(
				ctx,
				"toolbox/server/prompt/init",
				trace.WithAttributes(attribute.String("prompt_kind", pc.PromptConfigKind())),
				trace.WithAttributes(attribute.String("prompt_name", name)),
			)
			defer span.End()
			p, err := pc.Initialize()
			if err != nil {
				return nil, fmt.Errorf("unable to initialize prompt %q: %w", name, err)
			}
			return p, nil
		}()
		if err != nil {
			return nil, nil, nil, nil, nil, nil, err
		}
		promptsMap[name] = p
	}
	promptNames := make([]string, 0, len(promptsMap))
	for name := range promptsMap {
		promptNames = append(promptNames, name)
	}
	l.InfoContext(ctx, fmt.Sprintf("Initialized %d prompts: %s", len(promptsMap), strings.Join(promptNames, ", ")))

	// create a default promptset that contains all prompts
	allPromptNames := make([]string, 0, len(promptsMap))
	for name := range promptsMap {
		allPromptNames = append(allPromptNames, name)
	}
	if cfg.PromptsetConfigs == nil {
		cfg.PromptsetConfigs = make(PromptsetConfigs)
	}
	cfg.PromptsetConfigs[""] = prompts.PromptsetConfig{Name: "", PromptNames: allPromptNames}

	// initialize and validate the promptsets from configs
	promptsetsMap := make(map[string]prompts.Promptset)
	for name, pc := range cfg.PromptsetConfigs {
		p, err := func() (prompts.Promptset, error) {
			_, span := instrumentation.Tracer.Start(
				ctx,
				"toolbox/server/prompset/init",
				trace.WithAttributes(attribute.String("prompset_name", name)),
			)
			defer span.End()
			p, err := pc.Initialize(cfg.Version, promptsMap)
			if err != nil {
				return prompts.Promptset{}, fmt.Errorf("unable to initialize promptset %q: %w", name, err)
			}
			return p, err
		}()
		if err != nil {
			return nil, nil, nil, nil, nil, nil, err
		}
		promptsetsMap[name] = p
	}
	promptsetNames := make([]string, 0, len(promptsetsMap))
	for name := range promptsetsMap {
		if name == "" {
			promptsetNames = append(promptsetNames, "default")
		} else {
			promptsetNames = append(promptsetNames, name)
		}
	}
	l.InfoContext(ctx, fmt.Sprintf("Initialized %d promptsets: %s", len(promptsetsMap), strings.Join(promptsetNames, ", ")))

	return sourcesMap, authServicesMap, toolsMap, toolsetsMap, promptsMap, promptsetsMap, nil
}

// NewServer returns a Server object based on provided Config.
func NewServer(ctx context.Context, cfg ServerConfig) (*Server, error) {
	instrumentation, err := util.InstrumentationFromContext(ctx)
	if err != nil {
		return nil, err
	}

	ctx, span := instrumentation.Tracer.Start(ctx, "toolbox/server/init")
	defer span.End()

	l, err := util.LoggerFromContext(ctx)
	if err != nil {
		return nil, err
	}

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
			RequestHeaders:   false,
			MessageFieldName: "message",
			SourceFieldName:  "logging.googleapis.com/sourceLocation",
			TimeFieldName:    "timestamp",
			LevelFieldName:   "severity",
		}
	case "standard":
		httpOpts = httplog.Options{
			LogLevel:         logLevel,
			Concise:          true,
			RequestHeaders:   false,
			MessageFieldName: "message",
		}
	default:
		return nil, fmt.Errorf("invalid Logging format: %q", cfg.LoggingFormat.String())
	}
	httpLogger := httplog.NewLogger("httplog", httpOpts)
	r.Use(httplog.RequestLogger(httpLogger))

	sourcesMap, authServicesMap, toolsMap, toolsetsMap, promptsMap, promptsetsMap, err := InitializeConfigs(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("unable to initialize configs: %w", err)
	}

	addr := net.JoinHostPort(cfg.Address, strconv.Itoa(cfg.Port))
	srv := &http.Server{Addr: addr, Handler: r}

	sseManager := newSseManager(ctx)

	resourceManager := resources.NewResourceManager(sourcesMap, authServicesMap, toolsMap, toolsetsMap, promptsMap, promptsetsMap)

	s := &Server{
		version:         cfg.Version,
		srv:             srv,
		root:            r,
		logger:          l,
		instrumentation: instrumentation,
		sseManager:      sseManager,
		ResourceMgr:     resourceManager,
	}

	// cors
	if slices.Contains(cfg.AllowedOrigins, "*") {
		s.logger.WarnContext(ctx, "wildcard (`*`) allows all origin to access the resource and is not secure. Use it with cautious for public, non-sensitive data, or during local development. Recommended to use `--allowed-origins` flag to prevent DNS rebinding attacks")
	}
	corsOpts := cors.Options{
		AllowedOrigins:   cfg.AllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "DELETE", "OPTIONS"},
		AllowCredentials: true, // required since Toolbox uses auth headers
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "Mcp-Session-Id", "MCP-Protocol-Version"},
		ExposedHeaders:   []string{"Mcp-Session-Id"}, // headers that are sent to clients
		MaxAge:           300,                        // cache preflight results for 5 minutes
	}
	r.Use(cors.Handler(corsOpts))

	// control plane
	apiR, err := apiRouter(s)
	if err != nil {
		return nil, err
	}
	r.Mount("/api", apiR)
	mcpR, err := mcpRouter(s)
	if err != nil {
		return nil, err
	}
	r.Mount("/mcp", mcpR)
	if cfg.UI {
		webR, err := webRouter()
		if err != nil {
			return nil, err
		}
		r.Mount("/ui", webR)
	}
	// default endpoint for validating server is running
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("🧰 Hello, World! 🧰"))
	})

	if err := serveJsonFromYamlFile(r, cfg.OAuthProtectedResource, "/.well-known/oauth-protected-resource"); err != nil {
		return nil, err
	}

	if err := serveJsonFromYamlFile(r, cfg.OAuthAuthorizationServer, "/.well-known/oauth-authorization-server"); err != nil {
		return nil, err
	}

	return s, nil
}

// helper function to serve yaml files as json from fixed paths.
func serveJsonFromYamlFile(r *chi.Mux, filePath, endpointPath string) error {
	if filePath == "" {
		return nil
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("error reading yaml file %s: %w", filePath, err)
	}
	var metadata map[string]interface{}
	if err := yaml.Unmarshal(data, &metadata); err != nil {
		return fmt.Errorf("error unmarshalling yaml file %s: %w", filePath, err)
	}
	jsonData, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshalling yaml file %s as json: %w", filePath, err)
	}
	r.Get(endpointPath, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(jsonData)
	})
	return nil
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
	s.logger.DebugContext(ctx, fmt.Sprintf("server listening on %s", s.srv.Addr))
	return nil
}

// Serve starts an HTTP server for the given Server instance.
func (s *Server) Serve(ctx context.Context) error {
	s.logger.DebugContext(ctx, "Starting a HTTP server.")
	return s.srv.Serve(s.listener)
}

// ServeStdio starts a new stdio session for mcp.
func (s *Server) ServeStdio(ctx context.Context, stdin io.Reader, stdout io.Writer) error {
	stdioServer := NewStdioSession(s, stdin, stdout)
	return stdioServer.Start(ctx)
}

// Shutdown gracefully shuts down the server without interrupting any active
// connections. It uses http.Server.Shutdown() and has the same functionality.
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.DebugContext(ctx, "shutting down the server.")
	return s.srv.Shutdown(ctx)
}
