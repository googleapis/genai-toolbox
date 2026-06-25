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

package server_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/googleapis/mcp-toolbox/internal/log"
	"github.com/googleapis/mcp-toolbox/internal/server"
	"github.com/googleapis/mcp-toolbox/internal/telemetry"
	"github.com/googleapis/mcp-toolbox/internal/util"
)

func testContext() context.Context {
	ctx := context.Background()
	instrumentation, _ := telemetry.CreateTelemetryInstrumentation("1.0.0")
	ctx = util.WithInstrumentation(ctx, instrumentation)
	logger, _ := log.NewLogger("standard", "info", os.Stdout, os.Stderr)
	ctx = util.WithLogger(ctx, logger)
	return ctx
}

func TestInitializeConfigs_Resources(t *testing.T) {
	ctx := testContext()

	t.Run("valid configuration with resources and templates", func(t *testing.T) {
		yamlStr := `
---
kind: resource
name: static-doc
type: text
text: "This is a static text document."
uri: "info://static-doc"
description: "A valid text resource"
---
kind: resourceTemplate
name: logs-template
type: file
uri: "file:///var/log/{path}"
allowedPaths:
  - "/var/log"
`
		unmarshaled, err := server.UnmarshalConfigs(ctx, []byte(yamlStr))
		if err != nil {
			t.Fatalf("UnmarshalConfigs failed: %v", err)
		}

		cfg := server.ServerConfig{
			Version:                 "1.0.0",
			ResourceConfigs:         unmarshaled.Resources,
			ResourceTemplateConfigs: unmarshaled.ResourceTemplates,
			ConfigDir:               "/var/log",
			UsingConfigFolder:       false,
		}

		_, _, _, _, _, _, _, resMap, tempMap, err := server.InitializeConfigs(ctx, cfg)
		if err != nil {
			t.Fatalf("InitializeConfigs failed: %v", err)
		}

		if len(resMap) != 1 || resMap["static-doc"] == nil {
			t.Errorf("expected 1 static resource, got: %v", resMap)
		}
		if len(tempMap) != 1 || tempMap["logs-template"] == nil {
			t.Errorf("expected 1 resource template, got: %v", tempMap)
		}
	})

	t.Run("name collision prevention", func(t *testing.T) {
		yamlStr := `
---
kind: resource
name: duplicate-name
type: text
text: "First doc."
---
kind: resourceTemplate
name: duplicate-name
type: file
uri: "file:///var/log/{path}"
`
		unmarshaled, err := server.UnmarshalConfigs(ctx, []byte(yamlStr))
		if err != nil {
			t.Fatalf("UnmarshalConfigs failed: %v", err)
		}

		cfg := server.ServerConfig{
			ResourceConfigs:         unmarshaled.Resources,
			ResourceTemplateConfigs: unmarshaled.ResourceTemplates,
		}

		_, _, _, _, _, _, _, _, _, err = server.InitializeConfigs(ctx, cfg)
		if err == nil || !strings.Contains(err.Error(), "resource name collision") {
			t.Errorf("expected error for duplicate resource name, got: %v", err)
		}
	})

	t.Run("URI collision prevention", func(t *testing.T) {
		yamlStr := `
---
kind: resource
name: resource-one
type: text
text: "Doc one."
uri: "info://shared-uri"
---
kind: resource
name: resource-two
type: text
text: "Doc two."
uri: "info://shared-uri"
`
		unmarshaled, err := server.UnmarshalConfigs(ctx, []byte(yamlStr))
		if err != nil {
			t.Fatalf("UnmarshalConfigs failed: %v", err)
		}

		cfg := server.ServerConfig{
			ResourceConfigs: unmarshaled.Resources,
		}

		_, _, _, _, _, _, _, _, _, err = server.InitializeConfigs(ctx, cfg)
		if err == nil || !strings.Contains(err.Error(), "resource URI collision") {
			t.Errorf("expected error for duplicate resource URI, got: %v", err)
		}
	})

	t.Run("scheme whitelisting checks", func(t *testing.T) {
		yamlStr := `
---
kind: resource
name: invalid-scheme-file
type: file
uri: "http://var/log/app.log"
`
		unmarshaled, err := server.UnmarshalConfigs(ctx, []byte(yamlStr))
		if err != nil {
			t.Fatalf("UnmarshalConfigs failed: %v", err)
		}

		cfg := server.ServerConfig{
			ResourceConfigs: unmarshaled.Resources,
		}

		_, _, _, _, _, _, _, _, _, err = server.InitializeConfigs(ctx, cfg)
		if err == nil || !strings.Contains(err.Error(), "URI must start with file://") {
			t.Errorf("expected initialization error for invalid file URI scheme, got: %v", err)
		}
	})

	t.Run("local-only template scoping checks", func(t *testing.T) {
		yamlStr := `
---
kind: resourceTemplate
name: bad-template
type: file
uri: "file:///var/log/{invalid_var}"
`
		unmarshaled, err := server.UnmarshalConfigs(ctx, []byte(yamlStr))
		if err != nil {
			t.Fatalf("UnmarshalConfigs failed: %v", err)
		}

		cfg := server.ServerConfig{
			ResourceTemplateConfigs: unmarshaled.ResourceTemplates,
		}

		_, _, _, _, _, _, _, _, _, err = server.InitializeConfigs(ctx, cfg)
		if err == nil || !strings.Contains(err.Error(), "only '{path}' is permitted") {
			t.Errorf("expected error for invalid template variable, got: %v", err)
		}
	})

	t.Run("allowed paths validation in config-folder mode at boot", func(t *testing.T) {
		yamlStr := `
---
kind: resource
name: no-allowed-paths
type: file
uri: "file:///var/log/app.log"
`
		unmarshaled, err := server.UnmarshalConfigs(ctx, []byte(yamlStr))
		if err != nil {
			t.Fatalf("UnmarshalConfigs failed: %v", err)
		}

		cfg := server.ServerConfig{
			ResourceConfigs:   unmarshaled.Resources,
			ConfigDir:         "/var/log",
			UsingConfigFolder: true, // Config folder mode is active
		}

		_, _, _, _, _, _, _, _, _, err = server.InitializeConfigs(ctx, cfg)
		if err == nil || !strings.Contains(err.Error(), "must explicitly define 'allowedPaths'") {
			t.Errorf("expected error for missing allowedPaths in config-folder mode, got: %v", err)
		}
	})
}
