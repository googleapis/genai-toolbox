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

package server_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/auth"
	"github.com/googleapis/genai-toolbox/internal/log"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/sources/alloydbpg"
	"github.com/googleapis/genai-toolbox/internal/telemetry"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/util"
)

func TestServe(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	addr, port := "127.0.0.1", 5000
	cfg := server.ServerConfig{
		Version: "0.0.0",
		Address: addr,
		Port:    port,
	}

	otelShutdown, err := telemetry.SetupOTel(ctx, "0.0.0", "", false, "toolbox")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	defer func() {
		err := otelShutdown(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
	}()

	testLogger, err := log.NewStdLogger(os.Stdout, os.Stderr, "info")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	ctx = util.WithLogger(ctx, testLogger)

	instrumentation, err := telemetry.CreateTelemetryInstrumentation(cfg.Version)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	ctx = util.WithInstrumentation(ctx, instrumentation)

	s, err := server.NewServer(ctx, cfg)
	if err != nil {
		t.Fatalf("unable to initialize server: %v", err)
	}

	err = s.Listen(ctx)
	if err != nil {
		t.Fatalf("unable to start server: %v", err)
	}

	// start server in background
	errCh := make(chan error)
	go func() {
		defer close(errCh)

		err = s.Serve(ctx)
		if err != nil {
			errCh <- err
		}
	}()

	url := fmt.Sprintf("http://%s:%d/", addr, port)
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("error when sending a request: %s", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("response status code is not 200")
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("error reading from request body: %s", err)
	}
	if got := string(raw); strings.Contains(got, "0.0.0") {
		t.Fatalf("version missing from output: %q", got)
	}
}

func TestUpdateServer(t *testing.T) {
	ctx, err := testutils.ContextWithNewLogger()
	if err != nil {
		t.Fatalf("error setting up logger: %s", err)
	}

	addr, port := "127.0.0.1", 5000
	cfg := server.ServerConfig{
		Version: "0.0.0",
		Address: addr,
		Port:    port,
	}

	instrumentation, err := telemetry.CreateTelemetryInstrumentation(cfg.Version)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	ctx = util.WithInstrumentation(ctx, instrumentation)

	s, err := server.NewServer(ctx, cfg)
	if err != nil {
		t.Fatalf("error setting up server: %s", err)
	}

	newSources := map[string]sources.Source{
		"example-source": &alloydbpg.Source{
			Name: "example-alloydb-source",
			Kind: "alloydb-postgres",
		},
	}
	newAuth := map[string]auth.AuthService{"example-auth": nil}
	newTools := map[string]tools.Tool{"example-tool": nil}
	newToolsets := map[string]tools.Toolset{
		"example-toolset": {
			Name: "example-toolset", Tools: []*tools.Tool{},
		},
	}
	err = server.UpdateServer(ctx, s, newSources, newAuth, newTools, newToolsets)
	if err != nil {
		t.Errorf("error updating server: %s", err)
	}

	gotSource, _ := s.ConfigManager.GetSource("example-source")
	if gotSource != newSources["example-source"] {
		t.Errorf("error updating server, got: %s, want: %s", gotSource, newSources["example-source"])
	}

	gotAuthService, _ := s.ConfigManager.GetAuthService("example-auth")
	if gotAuthService != newAuth["example-auth"] {
		t.Errorf("error updating server, got: %s, want: %s", gotSource, newSources["example-auth"])
	}

	gotTool, _ := s.ConfigManager.GetTool("example-tool")
	if gotTool != newTools["example-tool"] {
		t.Errorf("error updating server, got: %s, want: %s", gotSource, newSources["example-tool"])
	}

	gotToolset, _ := s.ConfigManager.GetToolset("example-toolset")
	if diff := cmp.Diff(gotToolset, newToolsets["example-toolset"]); diff != "" {
		t.Errorf("error updating server, toolset (-want +got):\n%s", diff)
	}
}
