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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"
)

var (
	HTTP_SOURCE_KIND = "http"
	HTTP_TOOL_KIND   = "http-json"
)

func getHTTPVars(t *testing.T) map[string]any {
	idToken, err := GetGoogleIdToken(ClientId)
	if err != nil {
		t.Fatalf("error getting ID token: %s", err)
	}
	idToken = "Bearer " + idToken
	return map[string]any{
		"kind":    HTTP_SOURCE_KIND,
		"headers": map[string]string{"Authorization": idToken},
	}
}

// handler function for the test server
func multiTool(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	path = strings.TrimPrefix(path, "/") // Remove leading slash

	switch path {
	case "tool0":
		handleTool0(w, r)
	case "tool1":
		handleTool1(w, r)
	case "tool2":
		handleTool2(w, r)
	default:
		http.NotFound(w, r) // Return 404 for unknown paths
	}
}

// handler function for the test server
func handleTool0(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	response := []string{
		"Hello",
		"World",
	}
	err := json.NewEncoder(w).Encode(response)
	if err != nil {
		http.Error(w, "Failed to encode JSON", http.StatusInternalServerError)
		return
	}
}

// handler function for the test server
func handleTool1(w http.ResponseWriter, r *http.Request) {
	// Parse request body
	var requestBody map[string]interface{}
	bodyBytes, readErr := io.ReadAll(r.Body)
	if readErr != nil {
		http.Error(w, "Bad Request: Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()
	err := json.Unmarshal(bodyBytes, &requestBody)
	if err != nil {
		errorMessage := fmt.Sprintf("Bad Request: Error unmarshalling request body: %s, Raw body: %s", err, string(bodyBytes))
		http.Error(w, errorMessage, http.StatusBadRequest)
		return
	}

	// Extract name
	name, ok := requestBody["name"].(string)
	if !ok || name == "" {
		http.Error(w, "Bad Request: Missing or invalid name", http.StatusBadRequest)
		return
	}

	if name == "Alice" {
		response := []map[string]interface{}{
			{"id": 1, "name": "Alice"},
			{"id": 3, "name": "Sid"},
		}
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			http.Error(w, "Failed to encode JSON", http.StatusInternalServerError)
			return
		}
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// handler function for the test server
func handleTool2(w http.ResponseWriter, r *http.Request) {
	email := r.URL.Query().Get("email")
	if email != "" {
		response := []map[string]interface{}{
			{"name": "Alice"},
		}
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			http.Error(w, "Failed to encode JSON", http.StatusInternalServerError)
			return
		}
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func TestJSONToolEndpoints(t *testing.T) {
	// start a test server
	server := httptest.NewServer(http.HandlerFunc(multiTool))
	defer server.Close()

	sourceConfig := getHTTPVars(t)
	sourceConfig["baseUrl"] = server.URL
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
