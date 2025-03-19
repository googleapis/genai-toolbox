//go:build integration && http

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

package tests

import (
	"context"
	"os"
	"regexp"
	"testing"
	"time"
)

var (
	HTTP_SOURCE_KIND = "http"
	HTTP_TOOL_KIND   = "http-json"
	HTTP_BASE_URL    = os.Getenv("HTTP_BASE_URL")
)

func getHTTPVars(t *testing.T) map[string]any {
	idToken, err := GetGoogleIdToken(ClientId)
	if err != nil {
		t.Fatalf("error getting ID token: %s", err)
	}
	idToken = "Bearer " + idToken
	switch "" {
	case HTTP_BASE_URL:
	}
	return map[string]any{
		"kind":    HTTP_SOURCE_KIND,
		"baseUrl": HTTP_BASE_URL,
		"headers": map[string]string{"Authorization": idToken},
	}
}

func TestJSONToolEndpoints(t *testing.T) {
	sourceConfig := getHTTPVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var args []string

	toolsFile := GetHTTPToolsConfig(sourceConfig, HTTP_TOOL_KIND)
	cmd, cleanup, err := StartCmd(ctx, toolsFile, args...)
	if err != nil {
		t.Fatalf("command initialization returned an error: %s", err)
	}
	defer cleanup()

	waitCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	out, err := cmd.WaitForString(waitCtx, regexp.MustCompile(`Server ready to serve`))
	if err != nil {
		t.Logf("toolbox command logs: \n%s", out)
		t.Fatalf("toolbox didn't start successfully: %s", err)
	}
	RunToolGetTest(t)
	RunToolInvokeTest(t, "[\"Hello\",\"World\"]")
}
