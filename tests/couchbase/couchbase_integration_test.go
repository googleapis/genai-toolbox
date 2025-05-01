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
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/couchbase/gocb/v2"
	"github.com/google/uuid"
	"github.com/googleapis/genai-toolbox/tests"
)

const (
	couchbaseSourceKind = "couchbase"
	couchbaseToolKind   = "couchbase-sql"
)

var (
	couchbaseConnection   = os.Getenv("COUCHBASE_CONNECTION")
	couchbaseBucket       = os.Getenv("COUCHBASE_BUCKET")
	couchbaseScope        = os.Getenv("COUCHBASE_SCOPE")
	couchbaseUser         = os.Getenv("COUCHBASE_USER")
	couchbasePass         = os.Getenv("COUCHBASE_PASS")
	SERVICE_ACCOUNT_EMAIL = os.Getenv("SERVICE_ACCOUNT_EMAIL")
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
		"kind":                 couchbaseSourceKind,
		"connectionString":     couchbaseConnection,
		"bucket":               couchbaseBucket,
		"scope":                couchbaseScope,
		"username":             couchbaseUser,
		"password":             couchbasePass,
		"queryScanConsistency": 2,
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
	teardownCollection1 := setupCouchbaseCollection(t, ctx, cluster, couchbaseBucket, couchbaseScope, collectionNameParam, params1)
	defer teardownCollection1(t)

	// Set up data for auth tool
	authToolStatement, params2 := getCouchbaseAuthToolInfo(collectionNameAuth)
	teardownCollection2 := setupCouchbaseCollection(t, ctx, cluster, couchbaseBucket, couchbaseScope, collectionNameAuth, params2)
	defer teardownCollection2(t)

	// Write config into a file and pass it to command
	toolsFile := tests.GetToolsConfig(sourceConfig, couchbaseToolKind, paramToolStatement, authToolStatement)

	cmd, cleanup, err := tests.StartCmd(ctx, toolsFile, args...)
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

	tests.RunToolGetTest(t)

	select1Want := "[{\"$1\":1}]"
	failMcpInvocationWant := "{\"jsonrpc\":\"2.0\",\"id\":\"invoke-fail-tool\",\"result\":{\"content\":[{\"type\":\"text\",\"text\":\"unable to execute query: parsing failure | {\\\"statement\\\":\\\"SELEC 1;\\\""

	invokeParamWant, mcpInvokeParamWant := tests.GetNonSpannerInvokeParamWant()
	tests.RunToolInvokeTest(t, select1Want, invokeParamWant)
	tests.RunMCPToolCallMethod(t, mcpInvokeParamWant, failMcpInvocationWant)
}

// setupCouchbaseCollection creates a scope and collection and inserts test data
func setupCouchbaseCollection(t *testing.T, ctx context.Context, cluster *gocb.Cluster,
	bucketName, scopeName, collectionName string, params []map[string]any) func(t *testing.T) {

	// Get bucket reference
	bucket := cluster.Bucket(bucketName)

	// Wait for bucket to be ready
	err := bucket.WaitUntilReady(5*time.Second, nil)
	if err != nil {
		t.Fatalf("failed to connect to bucket: %v", err)
	}

	// Create scope if it doesn't exist
	bucketMgr := bucket.CollectionsV2()
	err = bucketMgr.CreateScope(scopeName, nil)
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		t.Logf("failed to create scope (might already exist): %v", err)
	}

	// Create a collection if it doesn't exist
	err = bucketMgr.CreateCollection(scopeName, collectionName, nil, nil)
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("failed to create collection: %v", err)
	}
	time.Sleep(1 * time.Second)

	// Get a reference to the collection
	collection := bucket.Scope(scopeName).Collection(collectionName)

	// Create primary index if it doesn't exist
	// Create primary index with retry logic
	maxRetries := 5
	retryDelay := 1 * time.Second
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		err = collection.QueryIndexes().CreatePrimaryIndex(
			&gocb.CreatePrimaryQueryIndexOptions{
				IgnoreIfExists: true,
			})
		if err == nil {
			break
		}

		lastErr = err
		t.Logf("Attempt %d: failed to create primary index: %v, retrying in %v", attempt+1, err, retryDelay)
		time.Sleep(retryDelay)
		// Exponential backoff
		retryDelay *= 2
	}

	if lastErr != nil {
		t.Fatalf("failed to create primary index collection after %d attempts: %v", maxRetries, lastErr)
	}

	// Insert test documents
	for i, param := range params {
		_, err = collection.Upsert(fmt.Sprintf("%d", i+1), param, &gocb.UpsertOptions{
			DurabilityLevel: gocb.DurabilityLevelMajority,
		})
		if err != nil {
			t.Fatalf("failed to insert test data: %v", err)
		}
	}

	// Return a cleanup function
	return func(t *testing.T) {
		// Drop the collection
		err := bucketMgr.DropCollection(scopeName, collectionName, nil)
		if err != nil {
			t.Logf("failed to drop collection: %v", err)
		}
	}
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

	params := []map[string]any{
		{"name": "Alice", "email": SERVICE_ACCOUNT_EMAIL},
		{"name": "Jane", "email": "janedoe@gmail.com"},
	}
	return toolStatement, params
}
