// Copyright 2025 Google LLC
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

package dataplex

import (
	"context"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/tests"
)

var (
	DataplexSourceKind            = "dataplex"
	DataplexSearchEntriesToolKind = "dataplex-search-entries"
	DataplexProject               = os.Getenv("DATAPLEX_PROJECT")
)

func getDataplexVars(t *testing.T) map[string]any {
	switch "" {
	case DataplexProject:
		t.Fatal("'DATAPLEX_PROJECT' not set")
	}
	return map[string]any{
		"kind":    DataplexSourceKind,
		"project": DataplexProject,
	}
}

func TestDataplexToolEndpoints(t *testing.T) {
	sourceConfig := getDataplexVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var args []string

	toolsFile := getDataplexToolsConfig(sourceConfig)

	cmd, cleanup, err := tests.StartCmd(ctx, toolsFile, args...)
	if err != nil {
		t.Fatalf("command initialization returned an error: %s", err)
	}
	defer cleanup()

	waitCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	out, err := testutils.WaitForString(waitCtx, regexp.MustCompile(`Server ready to serve`), cmd.Out)
	if err != nil {
		t.Logf("toolbox command logs: \n%s", out)
		t.Fatalf("toolbox didn't start successfully: %s", err)
	}

	tests.RunToolGetTest(t)
}

func getDataplexToolsConfig(sourceConfig map[string]any) map[string]any {
	// Write config into a file and pass it to command
	toolsFile := map[string]any{
		"sources": map[string]any{
			"my-dataplex-instance": sourceConfig,
		},
		"tools": map[string]any{
			"my-search-entries-tool": map[string]any{
				"kind":        DataplexSearchEntriesToolKind,
				"source":      "my-dataplex-instance",
				"description": "Simple tool to test end to end functionality.",
			},
		},
	}

	return toolsFile
}
