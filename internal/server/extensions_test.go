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
	"context"
	"os"
	"reflect"
	"testing"

	"github.com/googleapis/mcp-toolbox/internal/log"
	"github.com/googleapis/mcp-toolbox/internal/telemetry"
	"github.com/googleapis/mcp-toolbox/internal/util"
)

func TestNewServer_Extensions(t *testing.T) {
	ctx := context.Background()
	testLogger, err := log.NewStdLogger(os.Stdout, os.Stderr, "info")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	ctx = util.WithLogger(ctx, testLogger)

	instrumentation, err := telemetry.CreateTelemetryInstrumentation("0.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	ctx = util.WithInstrumentation(ctx, instrumentation)

	tests := []struct {
		name       string
		enableExt  []string
		disableExt []string
		want       []string
	}{
		{
			name:       "default enables toolbox extension",
			enableExt:  nil,
			disableExt: nil,
			want:       []string{"com.google.cloud/toolbox.v1"},
		},
		{
			name:       "custom enable and disable",
			enableExt:  []string{"com.google.cloud/toolbox.v1", "io.modelcontextprotocol/tasks"},
			disableExt: []string{"io.modelcontextprotocol/tasks"},
			want:       []string{"com.google.cloud/toolbox.v1"},
		},
		{
			name:       "disable default extension",
			enableExt:  nil,
			disableExt: []string{"com.google.cloud/toolbox.v1"},
			want:       nil,
		},
		{
			name:       "empty strings ignored and deduplicated",
			enableExt:  []string{"com.google.cloud/toolbox.v1", "", "com.google.cloud/toolbox.v1"},
			disableExt: nil,
			want:       []string{"com.google.cloud/toolbox.v1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := ServerConfig{
				EnableExt:  tt.enableExt,
				DisableExt: tt.disableExt,
			}
			s, err := NewServer(ctx, cfg)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(s.extensions, tt.want) {
				t.Errorf("extensions = %v, want %v", s.extensions, tt.want)
			}
		})
	}
}
