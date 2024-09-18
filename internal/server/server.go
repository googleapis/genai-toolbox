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
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/tools"
)

// Server contains info for running an instance of Toolbox. Should be instantiated with NewServer().
type Server struct {
	conf Config
	root chi.Router

	sources map[string]sources.Source
	tools   map[string]tools.Tool
}

// NewServer returns a Server object based on provided Config.
func NewServer(cfg Config) (*Server, error) {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("🧰 Hello world! 🧰"))
	})

	// initalize and validate the sources
	sources := make(map[string]sources.Source)
	for name, sc := range cfg.SourceConfigs {
		s, err := sc.Initialize()
		if err != nil {
			return nil, fmt.Errorf("Unable to initialize tool %s: %w", name, err)
		}
		sources[name] = s
	}
	fmt.Printf("Initalized %d sources.\n", len(sources))

	// initalize and validate the tools
	tools := make(map[string]tools.Tool)
	for name, tc := range cfg.ToolConfigs {
		t, err := tc.Initialize(sources)
		if err != nil {
			return nil, fmt.Errorf("Unable to initialize tool %s: %w", name, err)
		}
		tools[name] = t
	}
	fmt.Printf("Initalized %d tools.\n", len(tools))

	s := &Server{
		conf:    cfg,
		root:    r,
		sources: sources,
		tools:   tools,
	}
	r.Mount("/api", apiRouter(s))

	return s, nil
}

// ListenAndServe starts an HTTP server for the given Server instance.
func (s *Server) ListenAndServe(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	addr := net.JoinHostPort(s.conf.Address, strconv.Itoa(s.conf.Port))
	lc := net.ListenConfig{KeepAlive: 30 * time.Second}
	l, err := lc.Listen(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to open listener for %q: %w", addr, err)
	}

	return http.Serve(l, s.root)
}
