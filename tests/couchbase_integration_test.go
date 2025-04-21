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
	"fmt"
	"os"
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
	paramToolStatement, params1 := GetCouchbaseParamToolInfo(collectionNameParam)
	teardownCollection1 := SetupCouchbaseCollection(t, ctx, cluster, couchbaseBucket, couchbaseScope, collectionNameParam, params1)
	defer teardownCollection1(t)

	// Set up data for auth tool
	authToolStatement, params2 := GetCouchbaseAuthToolInfo(collectionNameAuth)
	_ = SetupCouchbaseCollection(t, ctx, cluster, couchbaseBucket, couchbaseScope, collectionNameAuth, params2)
	//defer teardownCollection2(t)

	// Write config into a file and pass it to command
	toolsFile := GetToolsConfig(sourceConfig, couchbaseToolKind, paramToolStatement, authToolStatement)

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

	select1Want := "[{\"$1\":1}]"
	//time.Sleep(3 * time.Second)
	RunToolInvokeTest(t, select1Want)
}
