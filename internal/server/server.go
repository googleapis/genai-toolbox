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
	"io"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httplog/v2"
	"github.com/googleapis/genai-toolbox/internal/auth"
	"github.com/googleapis/genai-toolbox/internal/log"
	"github.com/googleapis/genai-toolbox/internal/prompts"
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
	ResourceMgr     *ResourceManager
}

// ResourceManager contains available resources for the server. Should be initialized with NewResourceManager().
type ResourceManager struct {
	mu           sync.RWMutex
	sources      map[string]sources.Source
	authServices map[string]auth.AuthService
	tools        map[string]tools.Tool
	toolsets     map[string]tools.Toolset
	prompts      map[string]prompts.Prompt
	promptsets   map[string]prompts.Promptset
}

func NewResourceManager(
	sourcesMap map[string]sources.Source,
	authServicesMap map[string]auth.AuthService,
	toolsMap map[string]tools.Tool, toolsetsMap map[string]tools.Toolset,
	promptsMap map[string]prompts.Prompt, promptsetsMap map[string]prompts.Promptset,

) *ResourceManager {
	resourceMgr := &ResourceManager{
		mu:           sync.RWMutex{},
		sources:      sourcesMap,
		authServices: authServicesMap,
		tools:        toolsMap,
		toolsets:     toolsetsMap,
		prompts:      promptsMap,
		promptsets:   promptsetsMap,
	}

	return resourceMgr
}

func (r *ResourceManager) GetSource(sourceName string) (sources.Source, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	source, ok := r.sources[sourceName]
	return source, ok
}

func (r *ResourceManager) GetAuthService(authServiceName string) (auth.AuthService, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	authService, ok := r.authServices[authServiceName]
	return authService, ok
}

func (r *ResourceManager) GetTool(toolName string) (tools.Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	tool, ok := r.tools[toolName]
	return tool, ok
}

func (r *ResourceManager) GetToolset(toolsetName string) (tools.Toolset, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	toolset, ok := r.toolsets[toolsetName]
	return toolset, ok
}

func (r *ResourceManager) GetPrompt(promptName string) (prompts.Prompt, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	prompt, ok := r.prompts[promptName]
	return prompt, ok
}

func (r *ResourceManager) GetPromptSet(promptsetName string) (prompts.Promptset, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	promptset, ok := r.promptsets[promptsetName]
	return promptset, ok
}

func (r *ResourceManager) SetResources(sourcesMap map[string]sources.Source, authServicesMap map[string]auth.AuthService, toolsMap map[string]tools.Tool, toolsetsMap map[string]tools.Toolset, promptsMap map[string]prompts.Prompt, promptsetsMap map[string]prompts.Promptset) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sources = sourcesMap
	r.authServices = authServicesMap
	r.tools = toolsMap
	r.toolsets = toolsetsMap
	r.prompts = promptsMap
	r.promptsets = promptsetsMap
}

func (r *ResourceManager) GetAuthServiceMap() map[string]auth.AuthService {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.authServices
}

func (r *ResourceManager) GetToolsMap() map[string]tools.Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.tools
}

func (r *ResourceManager) GetPromptsMap() map[string]prompts.Prompt {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.prompts
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
	l.InfoContext(ctx, fmt.Sprintf("Initialized %d sources.", len(sourcesMap)))

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
	l.InfoContext(ctx, fmt.Sprintf("Initialized %d authServices.", len(authServicesMap)))

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
	l.InfoContext(ctx, fmt.Sprintf("Initialized %d toolsets.", len(toolsetsMap)))

	// initialize and validate the prompts from configs
	promptsMap := make(map[string]prompts.Prompt)
	for name, tc := range cfg.PromptConfigs {
		t, err := func() (prompts.Prompt, error) {
			_, span := instrumentation.Tracer.Start(
				ctx,
				"toolbox/server/prompt/init",
				trace.WithAttributes(attribute.String("prompt_name", name)),
			)
			defer span.End()
			t, err := tc.Initialize(promptsMap)
			if err != nil {
				return nil, fmt.Errorf("unable to initialize prompt %q: %w", name, err)
			}
			return t, nil
		}()
		if err != nil {
			return nil, nil, nil, nil, nil, nil, err
		}
		promptsMap[name] = t
	}
	l.InfoContext(ctx, fmt.Sprintf("Initialized %d prompts.", len(promptsMap)))

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
	for name, tc := range cfg.PromptsetConfigs {
		t, err := func() (prompts.Promptset, error) {
			_, span := instrumentation.Tracer.Start(
				ctx,
				"toolbox/server/prompset/init",
				trace.WithAttributes(attribute.String("prompset_name", name)),
			)
			defer span.End()
			t, err := tc.Initialize(cfg.Version, promptsMap)
			if err != nil {
				return prompts.Promptset{}, fmt.Errorf("unable to initialize promptset %q: %w", name, err)
			}
			return t, err
		}()
		if err != nil {
			return nil, nil, nil, nil, nil, nil, err
		}
		promptsetsMap[name] = t
	}
	l.InfoContext(ctx, fmt.Sprintf("Initialized %d promptsets.", len(promptsetsMap)))

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

	resourceManager := NewResourceManager(sourcesMap, authServicesMap, toolsMap, toolsetsMap, promptsMap, promptsetsMap)

	s := &Server{
		version:         cfg.Version,
		srv:             srv,
		root:            r,
		logger:          l,
		instrumentation: instrumentation,
		sseManager:      sseManager,
		ResourceMgr:     resourceManager,
	}
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
