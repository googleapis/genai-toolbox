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

const (
	couchbaseSourceKind = "couchbase"
	couchbaseToolKind   = "couchbase-sql"
)

var (
	couchbaseConnection = os.Getenv("COUCHBASE_CONNECTION")
	couchbaseBucket     = os.Getenv("COUCHBASE_BUCKET")
	couchbaseScope      = os.Getenv("COUCHBASE_SCOPE")
	couchbaseUser       = os.Getenv("COUCHBASE_USER")
	couchbasePass       = os.Getenv("COUCHBASE_PASS")
)

// getCouchbaseVars validates and returns Couchbase configuration variables
func getCouchbaseVars(t *testing.T) map[string]any {
	switch "" {
	case couchbaseConnection:
		t.Fatal("'COUCHBASE_CONNECTION' not set")
	case couchbaseBucket:
		t.Fatal("'COUCHBASE_BUCKET' not set")
	case couchbaseScope:
		t.Fatal("'COUCHBASE_SCOPE' not set")
	case couchbaseUser:
		t.Fatal("'COUCHBASE_USER' not set")
	case couchbasePass:
		t.Fatal("'COUCHBASE_PASS' not set")
	}

	return map[string]any{
		"kind":              couchbaseSourceKind,
		"connection_string": couchbaseConnection,
		"bucket":            couchbaseBucket,
		"scope":             couchbaseScope,
		"username":          couchbaseUser,
		"password":          couchbasePass,
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

// getCouchbaseParamToolInfo returns statements and params for my-param-tool couchbase-sql kind
func getCouchbaseParamToolInfo(collectionName string) (string, []map[string]any) {
	// N1QL uses positional or named parameters with $ prefix
	toolStatement := fmt.Sprintf("SELECT TONUMBER(meta().id) as id, "+
		"%s.* FROM %s WHERE meta().id = TOSTRING($id) OR name = $name order by meta().id",
		collectionName, collectionName)

	params := []map[string]any{
		{"name": "Alice"},
		{"name": "Jane"},
		{"name": "Sid"},
	}
	return toolStatement, params
}

// getCouchbaseAuthToolInfo returns statements and param of my-auth-tool for couchbase-sql kind
func getCouchbaseAuthToolInfo(collectionName string) (string, []map[string]any) {
	toolStatement := fmt.Sprintf("SELECT name FROM %s WHERE email = $email", collectionName)

	// Use a placeholder email for testing
	testEmail := os.Getenv("SERVICE_ACCOUNT_EMAIL")
	if testEmail == "" {
		testEmail = "test@example.com"
	}

	params := []map[string]any{
		{"name": "Alice", "email": testEmail},
		{"name": "Jane", "email": "janedoe@gmail.com"},
	}
	return toolStatement, params
}

// setupCouchbaseCollection creates a scope and collection and inserts test data
func setupCouchbaseCollection(t *testing.T, ctx context.Context, cluster *gocb.Cluster,
	collectionName string, params []map[string]any) func(t *testing.T) {

	// Get bucket reference
	bucket := cluster.Bucket(couchbaseBucket)

	// Wait for bucket to be ready
	err := bucket.WaitUntilReady(5*time.Second, nil)
	if err != nil {
		t.Fatalf("failed to connect to bucket: %v", err)
	}

	// Create scope if it doesn't exist
	bucketMgr := bucket.Collections()
	err = bucketMgr.CreateScope(couchbaseScope, nil)
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		t.Logf("failed to create scope (might already exist): %v", err)
	}

	// Create collection if it doesn't exist
	err = bucketMgr.CreateCollection(gocb.CollectionSpec{
		Name:      collectionName,
		ScopeName: couchbaseScope,
	}, nil)
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("failed to create collection: %v", err)
	}

	// Get a reference to the collection
	collection := bucket.Scope(couchbaseScope).Collection(collectionName)

	// Insert test documents
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
			ScopeName: couchbaseScope,
		}, nil)
		if err != nil {
			t.Logf("failed to drop collection: %v", err)
		}
	}
}

// getCouchbaseToolsConfig returns a mock tools config file
func getCouchbaseToolsConfig(sourceConfig map[string]any, toolKind, paramToolStatement, authToolStatement string) map[string]any {
	// Get client ID with a default value to avoid validation errors
	clientID := os.Getenv("CLIENT_ID")
	if clientID == "" {
		clientID = "test-client-id" // Default value for testing
	}

	// Create tools configuration
	return map[string]any{
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
				"statement":   paramToolStatement,
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
				"statement":   authToolStatement,
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
}

// runCouchbaseToolGetTest tests the tool get endpoint
func runCouchbaseToolGetTest(t *testing.T) {
	// Test tool get endpoint
	testCases := []struct {
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

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := http.Get(tc.api)
			if err != nil {
				t.Fatalf("error when sending a request: %s", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Fatalf("response status code is %d, expected %d", resp.StatusCode, http.StatusOK)
			}

			var body map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
				t.Fatalf("error parsing response body: %v", err)
			}

			got, ok := body["tools"]
			if !ok {
				t.Fatalf("unable to find tools in response body")
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestCouchbaseToolEndpoints(t *testing.T) {
	sourceConfig := getCouchbaseVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var args []string

	cluster, err := initCouchbaseCluster(couchbaseConnection, couchbaseUser, couchbasePass)
	if err != nil {
		t.Fatalf("unable to create Couchbase connection: %s", err)
	}
	defer cluster.Close(nil)

	// Create collection names with UUID
	collectionNameParam := "param_" + strings.Replace(uuid.New().String(), "-", "", -1)
	collectionNameAuth := "auth_" + strings.Replace(uuid.New().String(), "-", "", -1)

	// Set up data for param tool
	paramToolStatement, params1 := getCouchbaseParamToolInfo(collectionNameParam)
	teardownCollection1 := setupCouchbaseCollection(t, ctx, cluster, collectionNameParam, params1)
	defer teardownCollection1(t)

	// Set up data for auth tool
	authToolStatement, params2 := getCouchbaseAuthToolInfo(collectionNameAuth)
	teardownCollection2 := setupCouchbaseCollection(t, ctx, cluster, collectionNameAuth, params2)
	defer teardownCollection2(t)

	// Write config into a file and pass it to command
	toolsFile := getCouchbaseToolsConfig(sourceConfig, couchbaseToolKind, paramToolStatement, authToolStatement)

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

	runCouchbaseToolGetTest(t)

	select1Want := "[{\"$1\":1}]"
	RunToolInvokeTest(t, select1Want)
}
