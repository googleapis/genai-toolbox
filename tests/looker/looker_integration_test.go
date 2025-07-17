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

package looker

import (
	"context"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/googleapis/genai-toolbox/internal/log"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/internal/util"
	"github.com/googleapis/genai-toolbox/tests"
)

var (
	LookerSourceKind   = "looker"
	LookerToolKind     = "looker_get_models"
	LookerBaseUrl      = os.Getenv("LOOKER_BASE_URL")
	LookerVerifySsl    = os.Getenv("LOOKER_VERIFY_SSL")
	LookerClientId     = os.Getenv("LOOKER_CLIENT_ID")
	LookerClientSecret = os.Getenv("LOOKER_CLIENT_SECRET")
)

func getLookerVars(t *testing.T) map[string]any {
	switch "" {
	case LookerBaseUrl:
		t.Fatal("'LOOKER_BASE_URL' not set")
	case LookerVerifySsl:
		t.Fatal("'LOOKER_VERIFY_SSL' not set")
	case LookerClientId:
		t.Fatal("'LOOKER_CLIENT_ID' not set")
	case LookerClientSecret:
		t.Fatal("'LOOKER_CLIENT_SECRET' not set")
	}

	return map[string]any{
		"kind":          LookerSourceKind,
		"base_url":      LookerBaseUrl,
		"verify_ssl":    (LookerVerifySsl == "true"),
		"client_id":     LookerClientId,
		"client_secret": LookerClientSecret,
	}
}

func TestLooker(t *testing.T) {
	sourceConfig := getLookerVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	testLogger, err := log.NewStdLogger(os.Stdout, os.Stderr, "info")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	ctx = util.WithLogger(ctx, testLogger)

	var args []string

	// Write config into a file and pass it to command
	toolsFile := map[string]any{
		"sources": map[string]any{
			"my-instance": sourceConfig,
		},
		"tools": map[string]any{
			"my-simple-tool": map[string]any{
				"kind":        LookerToolKind,
				"source":      "my-instance",
				"description": "Simple tool to test end to end functionality.",
			},
		},
	}

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

	wantResult := "{\"label\":\"System Activity\",\"name\":\"system__activity\",\"project_name\":\"system__activity\"}"
	tests.RunToolInvokeSimpleTest(t, wantResult)
}
