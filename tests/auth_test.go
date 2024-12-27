//go:build integration && auth

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
	"os/exec"
	"regexp"
	"time"

	"testing"

	"github.com/googleapis/genai-toolbox/internal/auth"
	"github.com/googleapis/genai-toolbox/internal/auth/google"
)

var clientId string = "32555940559.apps.googleusercontent.com"

var (
	CLOUD_SQL_POSTGRES_PROJECT  = os.Getenv("CLOUD_SQL_POSTGRES_PROJECT")
	CLOUD_SQL_POSTGRES_REGION   = os.Getenv("CLOUD_SQL_POSTGRES_REGION")
	CLOUD_SQL_POSTGRES_INSTANCE = os.Getenv("CLOUD_SQL_POSTGRES_INSTANCE")
	CLOUD_SQL_POSTGRES_DATABASE = os.Getenv("CLOUD_SQL_POSTGRES_DATABASE")
	CLOUD_SQL_POSTGRES_USER     = os.Getenv("CLOUD_SQL_POSTGRES_USER")
	CLOUD_SQL_POSTGRES_PASS     = os.Getenv("CLOUD_SQL_POSTGRES_PASS")
	SERVICE_ACCOUNT_EMAIL 		= os.Getenv("SERVICE_ACCOUNT_EMAIL")
)

// Get a Google ID token
func getGoogleIdToken(audience string) (string, error) {
	// For local testing
	cmd := exec.Command("gcloud", "auth", "print-identity-token")
	output, err := cmd.Output()
	if err == nil {
		return string(output), nil
	} else {
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
}

func TestGoogleAuthVerification(t *testing.T) {

	tcs := []struct {
		authSource auth.AuthSource
		isErr      bool
	}{
		{
			authSource: google.AuthSource{
				Name:     "my-google-auth",
				Kind:     google.AuthSourceKind,
				ClientID: clientId,
			},
			isErr: false,
		},
		{
			authSource: google.AuthSource{
				Name:     "err-google-auth",
				Kind:     google.AuthSourceKind,
				ClientID: "random-client-id",
			},
			isErr: true,
		},
	}
	for _, tc := range tcs {

		token, err := getGoogleIdToken(clientId)

		if err != nil {
			t.Fatalf("ID token generation error: %s", err)
		}
		headers := http.Header{}
		headers.Add("my-google-auth_token", token)
		claims, err := tc.authSource.GetClaimsFromHeader(headers)

		if err != nil {
			if tc.isErr {
				return
			} else {
				t.Fatalf("Error getting claims from token: %s", err)
			}
		}

		_, ok := claims["sub"]
		if !ok {
			if tc.isErr {
				return
			} else {
				t.Fatalf("Invalid claims.")
			}
		}
	}
}

func TestAuthenticatedParameter(t *testing.T) {
	requireCloudSQLPgVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var args []string

	// Set up test table
	pool, err := initCloudSQLPgConnectionPool(CLOUD_SQL_POSTGRES_PROJECT, CLOUD_SQL_POSTGRES_REGION, CLOUD_SQL_POSTGRES_INSTANCE, "public", CLOUD_SQL_POSTGRES_USER, CLOUD_SQL_POSTGRES_PASS, CLOUD_SQL_POSTGRES_DATABASE)
	if err != nil {
		return nil, fmt.Errorf("unable to create pool: %w", err)
	}

	err = pool.Ping(context.Background())
	if err != nil {
		return nil, fmt.Errorf("unable to connect successfully: %w", err)
	}
	
	results, err := pool.Query(context.Background(), `
		CREATE TABLE auth_table (
			id INT AUTO_INCREMENT PRIMARY KEY,
			name TEXT,
			email TEXT
		);
	`)
	stmt, err := pool.Prepare(context.Background(), `
		INSERT INTO auth_table (name, email) 
		VALUES ($1, $2), ($3, $4)
	`)
	if err != nil {
		return nil, fmt.Errorf("unable to prepare test statement: %w", err)
	}
	results, err := stmt.Exec(context.Background(),
		"Alice", SERVICE_ACCOUNT_EMAIL, 
		"Jane", "janedoe@gmail.com",
	)
	if err != nil {
		return nil, fmt.Errorf("unable to insert rows into test database: %w", err)
	}
	defer pool.Exec(context.Background(), `DROP TABLE auth_table;`)

	// Write config into a file and pass it to command
	toolsFile := map[string]any{
		"sources": map[string]any{
			"my-pg-instance": map[string]any{
				"kind":     "cloud-sql-postgres",
				"project":  CLOUD_SQL_POSTGRES_PROJECT,
				"instance": CLOUD_SQL_POSTGRES_INSTANCE,
				"region":   CLOUD_SQL_POSTGRES_REGION,
				"database": CLOUD_SQL_POSTGRES_DATABASE,
				"user":     CLOUD_SQL_POSTGRES_USER,
				"password": CLOUD_SQL_POSTGRES_PASS,
			},
		},
		"authSources": map[string]any{
			"my-google-auth-service": map[string]any{
				"kind":     "google",
				"clientId": clientId,
			},
		},
		"tools": map[string]any{
			"my-auth-tool": map[string]any{
				"kind":        "postgres-sql",
				"source":      "my-pg-instance",
				"description": "Tool to test authenticated parameters.",
				"statement":   "SELECT * WHERE email = $1;",
				"parameters": []map[string]any{
					{
						"name":        "my_string",
						"type":        "string",
						"description": "this param is a string",
						"authSources": []map[string]string{
							{
								"name":  "my-google-auth-service",
								"field": "email",
							},
						},
					},
				},
			},
		},
	}
	cmd, cleanup, err := StartCmd(ctx, toolsFile, args...)
	if err != nil {
		t.Fatalf("command initialization returned an error: %s", err)
	}
	defer cleanup()

	waitCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	out, err := cmd.WaitForString(waitCtx, regexp.MustCompile(`INFO "Server ready to serve"`))
	if err != nil {
		t.Logf("toolbox command logs: \n%s", out)
		t.Fatalf("toolbox didn't start successfully: %s", err)
	}

	// Test tool invocation with authenticated parameters
	invokeTcs := []struct {
		name        string
		api         string
		requestBody io.Reader
		want        string
	}{
		{
			name:        "invoke my-auth-tool",
			api:         "http://127.0.0.1:5000/api/tool/my-auth-tool/invoke",
			requestHeader: map[string]string{"my-google-auth_token":getGoogleIdToken(clientId)}
			requestBody: bytes.NewBuffer([]byte(`{}`)),
			want:        "Stub tool call for \"my-simple-tool\"! Parameters parsed: [] \n Output: [%!s(int32=1)]",
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(method, tc.api, tc.requestBody)
			if err != nil {
				return nil, nil, fmt.Errorf("unable to create request: %w", err)
			}
			req.Header.Add("Content-type", "application/json")
			for k, v := range tc.requestHeader{
				req.Header.Add(k, v)
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return nil, nil, fmt.Errorf("unable to send request: %w", err)
			}

			if resp.StatusCode != 200 {
				t.Fatalf("response status code is not 200")
			}

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
