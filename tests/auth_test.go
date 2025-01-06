//go:build integration

// Copyright 2024 Google LLC
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

package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"testing"
)

var clientId string = "32555940559.apps.googleusercontent.com"

var (
	CLOUD_SQL_POSTGRES_PROJECT  = os.Getenv("CLOUD_SQL_POSTGRES_PROJECT")
	CLOUD_SQL_POSTGRES_REGION   = os.Getenv("CLOUD_SQL_POSTGRES_REGION")
	CLOUD_SQL_POSTGRES_INSTANCE = os.Getenv("CLOUD_SQL_POSTGRES_INSTANCE")
	CLOUD_SQL_POSTGRES_DATABASE = os.Getenv("CLOUD_SQL_POSTGRES_DATABASE")
	CLOUD_SQL_POSTGRES_USER     = os.Getenv("CLOUD_SQL_POSTGRES_USER")
	CLOUD_SQL_POSTGRES_PASS     = os.Getenv("CLOUD_SQL_POSTGRES_PASS")
	SERVICE_ACCOUNT_EMAIL       = os.Getenv("SERVICE_ACCOUNT_EMAIL")
)

// Get a Google ID token
func getGoogleIdToken(audience string) (string, error) {
	// For local testing
	cmd := exec.Command("gcloud", "auth", "print-identity-token")
	output, err := cmd.Output()
	if err == nil {
		return strings.TrimSpace(string(output)), nil
	}
	// Cloud Build testing
	url := "http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/identity?audience=" + audience
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Metadata-Flavor", "Google")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func GoogleAuthenticatedParameterTestHelper(t *testing.T, sourceConfig map[string]any, toolKind string) {
	// create query statement
	var statement string
	switch {
	case strings.EqualFold(toolKind, "postgres-sql"):
		statement = "SELECT * FROM auth_table WHERE email = $1;"
	default:
		t.Fatalf("invalid tool kind: %s", toolKind)
	}

	// Write config into a file and pass it to command
	toolsFile := map[string]any{
		"sources": map[string]any{
			"my-pg-instance": sourceConfig,
		},
		"authSources": map[string]any{
			"my-google-auth": map[string]any{
				"kind":     "google",
				"clientId": clientId,
			},
		},
		"tools": map[string]any{
			"my-auth-tool": map[string]any{
				"kind":        toolKind,
				"source":      "my-pg-instance",
				"description": "Tool to test authenticated parameters.",
				// statement to auto-fill authenticated parameter
				"statement": statement,
				"parameters": []map[string]any{
					{
						"name":        "email",
						"type":        "string",
						"description": "user email",
						"authSources": []map[string]string{
							{
								"name":  "my-google-auth",
								"field": "email",
							},
						},
					},
				},
			},
		},
	}

	// Initialize a test command
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var args []string

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

	// Get ID token
	idToken, err := getGoogleIdToken(clientId)
	if err != nil {
		t.Fatalf("error getting Google ID token: %s", err)
	}

	// Create wanted string
	wantResult := fmt.Sprintf("Stub tool call for \"my-auth-tool\"! Parameters parsed: [{\"email\" \"%s\"}] \n Output: [%%!s(int32=1) Alice %s]", SERVICE_ACCOUNT_EMAIL, SERVICE_ACCOUNT_EMAIL)

	// Test tool invocation with authenticated parameters
	invokeTcs := []struct {
		name          string
		api           string
		requestHeader map[string]string
		requestBody   io.Reader
		want          string
		isErr         bool
	}{
		{
			name:          "Invoke my-auth-tool with auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-tool/invoke",
			requestHeader: map[string]string{"my-google-auth_token": idToken},
			requestBody:   bytes.NewBuffer([]byte(`{}`)),
			isErr:         false,
			want:          wantResult,
		},
		{
			name:          "Invoke my-auth-tool with invalid auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-tool/invoke",
			requestHeader: map[string]string{"my-google-auth_token": "INVALID_TOKEN"},
			requestBody:   bytes.NewBuffer([]byte(`{}`)),
			isErr:         true,
		},
		{
			name:          "Invoke my-auth-tool without auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{}`)),
			isErr:         true,
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			// Send Tool invocation request with ID token
			req, err := http.NewRequest(http.MethodPost, tc.api, tc.requestBody)
			if err != nil {
				t.Fatalf("unable to create request: %s", err)
			}
			req.Header.Add("Content-type", "application/json")
			for k, v := range tc.requestHeader {
				req.Header.Add(k, v)
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("unable to send request: %s", err)
			}

			if resp.StatusCode != http.StatusOK {
				if tc.isErr == true {
					return
				}
				bodyBytes, _ := io.ReadAll(resp.Body)
				t.Fatalf("response status code is not 200, got %d: %s", resp.StatusCode, string(bodyBytes))
			}

			// Check response body
			var body map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&body)
			if err != nil {
				t.Fatalf("error parsing response body")
			}
			got, ok := body["result"].(string)
			if !ok {
				t.Fatalf("unable to find result in response body")
			}

			if got != tc.want {
				t.Fatalf("unexpected value: got %q, want %q", got, tc.want)
			}
		})
	}
}

func AuthRequiredToolInvocationTestHelper(t *testing.T, sourceConfig map[string]any, toolKind string) {
	// Write config into a file and pass it to command
	toolsFile := map[string]any{
		"sources": map[string]any{
			"my-pg-instance": sourceConfig,
		},
		"authSources": map[string]any{
			"my-google-auth": map[string]any{
				"kind":     "google",
				"clientId": clientId,
			},
		},
		"tools": map[string]any{
			"my-auth-tool": map[string]any{
				"kind":        toolKind,
				"source":      "my-pg-instance",
				"description": "Tool to test authenticated parameters.",
				"authRequired": []string{
					"my-google-auth",
				},
			},
		},
	}

	// Initialize a test command
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var args []string

	cmd, cleanup, err := StartCmd(ctx, toolsFile, args...)
	if err != nil {
		t.Fatalf("command initialization returned an error: %s", err)
	}
	defer cleanup()

	waitCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	out, err := cmd.WaitForString(waitCtx, regexp.MustCompile(`Server ready to serve`))
	if err != nil {
		t.Logf("toolbox command logs: \n%s", out)
		t.Fatalf("toolbox didn't start successfully: %s", err)
	}

	// Get ID token
	idToken, err := getGoogleIdToken(clientId)
	if err != nil {
		t.Fatalf("error getting Google ID token: %s", err)
	}

	// Test auth-required Tool invocation
	invokeTcs := []struct {
		name          string
		api           string
		requestHeader map[string]string
		requestBody   io.Reader
		want          string
		isErr         bool
	}{
		{
			name:          "Invoke my-auth-tool with auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-tool/invoke",
			requestHeader: map[string]string{"my-google-auth_token": idToken},
			requestBody:   bytes.NewBuffer([]byte(`{}`)),
			isErr:         false,
			want:          "Stub tool call for \"my-auth-tool\"! Parameters parsed: [] \n Output: [%!s(int32=1)]",
		},
		{
			name:          "Invoke my-auth-tool with invalid auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-tool/invoke",
			requestHeader: map[string]string{"my-google-auth_token": "INVALID_TOKEN"},
			requestBody:   bytes.NewBuffer([]byte(`{}`)),
			isErr:         true,
		},
		{
			name:          "Invoke my-auth-tool without auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{}`)),
			isErr:         true,
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			// Send Tool invocation request with ID token
			req, err := http.NewRequest(http.MethodPost, tc.api, tc.requestBody)
			if err != nil {
				t.Fatalf("unable to create request: %s", err)
			}
			req.Header.Add("Content-type", "application/json")
			for k, v := range tc.requestHeader {
				req.Header.Add(k, v)
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("unable to send request: %s", err)
			}

			if resp.StatusCode != http.StatusOK {
				if tc.isErr == true {
					return
				}
				bodyBytes, _ := io.ReadAll(resp.Body)
				t.Fatalf("response status code is not 200, got %d: %s", resp.StatusCode, string(bodyBytes))
			}

			// Check response body
			var body map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&body)
			if err != nil {
				t.Fatalf("error parsing response body")
			}
			got, ok := body["result"].(string)
			if !ok {
				t.Fatalf("unable to find result in response body")
			}

			if got != tc.want {
				t.Fatalf("unexpected value: got %q, want %q", got, tc.want)
			}
		})
	}
}
