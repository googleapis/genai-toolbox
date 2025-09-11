// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package lookerhealthanalyze

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/testutils"
)

func TestNewLookerAnalyzeConfig(t *testing.T) {
	ctx := context.Background()
	sourceName := "test-looker-source"
	mockSource := testutils.NewMockLookerSource(t, sourceName)
	sources := map[string]sources.Source{sourceName: mockSource}

	testCases := []struct {
		name        string
		configYAML  string
		expectError bool
	}{
		{
			name: "valid config",
			configYAML: `
name: looker-analyze-tool
kind: looker-analyze
source: test-looker-source
description: A tool to analyze Looker projects, models, and explores.
`,
			expectError: false,
		},
		{
			name: "missing kind",
			configYAML: `
name: looker-analyze-tool
source: test-looker-source
description: A tool to analyze Looker projects, models, and explores.
`,
			expectError: true,
		},
		{
			name: "missing source",
			configYAML: `
name: looker-analyze-tool
kind: looker-analyze
description: A tool to analyze Looker projects, models, and explores.
`,
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			decoder := yaml.NewDecoder(strings.NewReader(tc.configYAML))
			cfg, err := newConfig(ctx, "test-tool", decoder)

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected an error but got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if _, err := cfg.Initialize(sources); err != nil {
				t.Errorf("Initialization failed: %v", err)
			}
		})
	}
}

func TestLookerAnalyzeInvoke(t *testing.T) {
	t.Skip("Skipping Invoke test as it requires a real Looker SDK client mock and complex data setup.")
}