//go:build integration && couchbase

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

package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/couchbase/gocb/v2"
	"github.com/google/uuid"
)

var (
	COUCHBASE_SOURCE_KIND = "couchbase"
	COUCHBASE_TOOL_KIND   = "couchbase-sql"
	COUCHBASE_CONNECTION  = os.Getenv("COUCHBASE_CONNECTION")
	COUCHBASE_BUCKET      = os.Getenv("COUCHBASE_BUCKET")
	COUCHBASE_SCOPE       = os.Getenv("COUCHBASE_SCOPE")
	COUCHBASE_USERNAME    = os.Getenv("COUCHBASE_USERNAME")
	COUCHBASE_PASSWORD    = os.Getenv("COUCHBASE_PASSWORD")
)

func getCouchbaseVars(t *testing.T) map[string]any {
	switch "" {
	case COUCHBASE_CONNECTION:
		t.Fatal("'COUCHBASE_CONNECTION' not set")
	case COUCHBASE_BUCKET:
		t.Fatal("'COUCHBASE_BUCKET' not set")
	case COUCHBASE_SCOPE:
		t.Fatal("'COUCHBASE_SCOPE' not set")
	case COUCHBASE_USERNAME:
		t.Fatal("'COUCHBASE_USERNAME' not set")
	case COUCHBASE_PASSWORD:
		t.Fatal("'COUCHBASE_PASSWORD' not set")
	}

	return map[string]any{
		"kind":              COUCHBASE_SOURCE_KIND,
		"connection_string": COUCHBASE_CONNECTION,
		"bucket":            COUCHBASE_BUCKET,
		"scope":             COUCHBASE_SCOPE,
		"username":          COUCHBASE_USERNAME,
		"password":          COUCHBASE_PASSWORD,
	}
}

// initCouchbaseCluster initializes a connection to the Couchbase cluster
func initCouchbaseCluster(connectionString, username, password string) (*gocb.Cluster, error) {
	opts := gocb.ClusterOptions{
		Authenticator: gocb.PasswordAuthenticator{
			Username: username,
			Password: password,
		},
	}

	cluster, err := gocb.Connect(connectionString, opts)
	if err != nil {
		return nil, fmt.Errorf("gocb.Connect: %w", err)
	}
	return cluster, nil
}

// GetCouchbaseParamToolInfo returns statements and params for my-param-tool couchbase-sql kind
func GetCouchbaseParamToolInfo(collectionName string) (string, []map[string]any) {
	// N1QL uses positional or named parameters with $ prefix
	tool_statement := fmt.Sprintf("SELECT TONUMBER(meta().id) as id, "+collectionName+".* FROM %s WHERE meta().id = TOSTRING($id) OR name = $name order by meta().id", collectionName)

	params := []map[string]any{
		map[string]any{"name": "Alice"},
		map[string]any{"name": "Jane"},
		map[string]any{"name": "Sid"},
	}
	return tool_statement, params
}

// GetCouchbaseAuthToolInfo returns statements and param of my-auth-tool for couchbase-sql kind
func GetCouchbaseAuthToolInfo(collectionName string) (string, []map[string]any) {
	tool_statement := fmt.Sprintf("SELECT name FROM %s WHERE email = $email", collectionName)

	// Use a placeholder email for testing
	testEmail := os.Getenv("SERVICE_ACCOUNT_EMAIL")
	if testEmail == "" {
		testEmail = "test@example.com"
	}

	params := []map[string]any{
		map[string]any{"name": "Alice", "email": testEmail},
		map[string]any{"name": "Jane", "email": "janedoe@gmail.com"},
	}
	return tool_statement, params
}

// SetupCouchbaseCollection creates a scope and collection and inserts test data
func SetupCouchbaseCollection(t *testing.T, ctx context.Context, cluster *gocb.Cluster,
	collectionName string, params []map[string]any) func(t *testing.T) {

	// Get bucket reference
	bucket := cluster.Bucket(COUCHBASE_BUCKET)

	// Wait for bucket to be ready
	err := bucket.WaitUntilReady(5*time.Second, nil)
	if err != nil {
		t.Fatalf("failed to connect to bucket: %v", err)
	}

	// Create scope if it doesn't exist
	// Note: This might fail if scope already exists, which is fine
	bucketMgr := bucket.Collections()
	err = bucketMgr.CreateScope(COUCHBASE_SCOPE, nil)
	if err != nil {
		// Ignore error if scope already exists
		if !strings.Contains(err.Error(), "already exists") {
			t.Logf("failed to create scope (might already exist): %v", err)
		}
	}

	// Create collection if it doesn't exist
	err = bucketMgr.CreateCollection(gocb.CollectionSpec{
		Name:      collectionName,
		ScopeName: COUCHBASE_SCOPE,
	}, nil)
	if err != nil {
		if !strings.Contains(err.Error(), "already exists") {
			t.Fatalf("failed to create collection: %v", err)
		}
	}

	// Get a reference to the collection
	collection := bucket.Scope(COUCHBASE_SCOPE).Collection(collectionName)

	// Insert test documents
	// For param tool test
	for i, param := range params {
		_, err = collection.Upsert(fmt.Sprintf("%d", i+1), param, &gocb.UpsertOptions{})
		if err != nil {
			t.Fatalf("failed to insert test data: %v", err)
		}
	}

	// Return a cleanup function
	return func(t *testing.T) {
		// Drop the collection
		err := bucketMgr.DropCollection(gocb.CollectionSpec{
			Name:      collectionName,
			ScopeName: COUCHBASE_SCOPE,
		}, nil)
		if err != nil {
			t.Logf("failed to drop collection: %v", err)
		}
	}
}

// GetCouchbaseToolsConfig returns a mock tools config file
func GetCouchbaseToolsConfig(sourceConfig map[string]any, toolKind, param_tool_statement, auth_tool_statement string) map[string]any {
	// Get client ID with a default value to avoid validation errors
	clientID := os.Getenv("CLIENT_ID")
	if clientID == "" {
		clientID = "test-client-id" // Default value for testing
	}

	// Write config into a file and pass it to command
	toolsFile := map[string]any{
		"sources": map[string]any{
			"my-instance": sourceConfig,
		},
		"authServices": map[string]any{
			"my-google-auth": map[string]any{
				"kind":     "google",
				"clientId": clientID,
			},
		},
		"tools": map[string]any{
			"my-simple-tool": map[string]any{
				"kind":        toolKind,
				"source":      "my-instance",
				"description": "Simple tool to test end to end functionality.",
				"statement":   "SELECT 1;",
			},
			"my-param-tool": map[string]any{
				"kind":        toolKind,
				"source":      "my-instance",
				"description": "Tool to test invocation with params.",
				"statement":   param_tool_statement,
				"parameters": []any{
					map[string]any{
						"name":        "id",
						"type":        "integer",
						"description": "user ID",
					},
					map[string]any{
						"name":        "name",
						"type":        "string",
						"description": "user name",
					},
				},
			},
			"my-auth-tool": map[string]any{
				"kind":        toolKind,
				"source":      "my-instance",
				"description": "Tool to test authenticated parameters.",
				// statement to auto-fill authenticated parameter
				"statement": auth_tool_statement,
				"parameters": []map[string]any{
					{
						"name":        "email",
						"type":        "string",
						"description": "user email",
						"authServices": []map[string]string{
							{
								"name":  "my-google-auth",
								"field": "email",
							},
						},
					},
				},
			},
			"my-auth-required-tool": map[string]any{
				"kind":        toolKind,
				"source":      "my-instance",
				"description": "Tool to test auth required invocation.",
				"statement":   "SELECT 1;",
				"authRequired": []string{
					"my-google-auth",
				},
			},
		},
	}

	return toolsFile
}

// RunCouchbaseToolGetTest tests the tool get endpoint
func RunCouchbaseToolGetTest(t *testing.T) {
	// Test tool get endpoint
	tcs := []struct {
		name string
		api  string
		want map[string]any
	}{
		{
			name: "get my-simple-tool",
			api:  "http://127.0.0.1:5000/api/tool/my-simple-tool/",
			want: map[string]any{
				"my-simple-tool": map[string]any{
					"description": "Simple tool to test end to end functionality.",
					"parameters":  []any{},
				},
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := http.Get(tc.api)
			if err != nil {
				t.Fatalf("error when sending a request: %s", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != 200 {
				t.Fatalf("response status code is not 200")
			}

			var body map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&body)
			if err != nil {
				t.Fatalf("error parsing response body")
			}

			got, ok := body["tools"]
			if !ok {
				t.Fatalf("unable to find tools in response body")
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestCouchbaseToolEndpoints(t *testing.T) {
	sourceConfig := getCouchbaseVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var args []string

	cluster, err := initCouchbaseCluster(COUCHBASE_CONNECTION, COUCHBASE_USERNAME, COUCHBASE_PASSWORD)
	if err != nil {
		t.Fatalf("unable to create Couchbase connection: %s", err)
	}
	defer cluster.Close(nil)

	// Create collection names with UUID
	collectionNameParam := "param_" + strings.Replace(uuid.New().String(), "-", "", -1)
	collectionNameAuth := "auth_" + strings.Replace(uuid.New().String(), "-", "", -1)

	// Set up data for param tool
	tool_statement1, params1 := GetCouchbaseParamToolInfo(collectionNameParam)
	teardownCollection1 := SetupCouchbaseCollection(t, ctx, cluster, collectionNameParam, params1)
	defer teardownCollection1(t)

	// Set up data for auth tool
	tool_statement2, params2 := GetCouchbaseAuthToolInfo(collectionNameAuth)
	teardownCollection2 := SetupCouchbaseCollection(t, ctx, cluster, collectionNameAuth, params2)
	defer teardownCollection2(t)

	// Write config into a file and pass it to command
	toolsFile := GetCouchbaseToolsConfig(sourceConfig, COUCHBASE_TOOL_KIND, tool_statement1, tool_statement2)

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

	RunCouchbaseToolGetTest(t)

	select_1_want := "[{\"$1\":1}]"
	RunToolInvokeTest(t, select_1_want)
}
