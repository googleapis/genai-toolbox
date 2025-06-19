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

package mongodb

import (
	"context"
	"fmt"
	"github.com/googleapis/genai-toolbox/tests"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"os"
	"regexp"
	"testing"
	"time"
)

var (
	MONGODB_SOURCE_KIND = "mongodb"
	MONGODB_TOOL_KIND   = "mongodb-find"
	MONGODB_URI         = os.Getenv("MONGODB_URI")
	MONGODB_DATABASE    = os.Getenv("MONGODB_DATABASE")
)

func getMongoDBVars(t *testing.T) map[string]any {
	switch "" {
	case MONGODB_URI:
		t.Fatal("'MONGODB_URI' not set")
	case MONGODB_DATABASE:
		t.Fatal("'MONGODB_DATABASE' not set")
	}
	return map[string]any{
		"kind":     MONGODB_SOURCE_KIND,
		"uri":      MONGODB_URI,
		"database": MONGODB_DATABASE,
	}
}

func initMongoDbDatabase(ctx context.Context, uri, database string) (*mongo.Database, error) {
	// Create a new mongodb Database
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("unable to connect to mongodb: %s", err)
	}
	err = client.Ping(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to mongodb: %s", err)
	}
	return client.Database(database), nil
}

func TestMongoDBToolEndpoints(t *testing.T) {
	sourceConfig := getMongoDBVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var args []string

	database, err := initMongoDbDatabase(ctx, MONGODB_URI, MONGODB_DATABASE)
	if err != nil {
		t.Fatalf("unable to create MongoDB connection: %s", err)
	}

	// set up data for param tool
	teardownDB := setupMongoDB(t, ctx, database)
	defer teardownDB(t)

	// Write config into a file and pass it to command
	toolsFile := tests.GetMongoDBToolsConfig(sourceConfig, MONGODB_TOOL_KIND)

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

	select1Want, failInvocationWant, invokeParamWant, mcpInvokeParamWant := tests.GetMongoDBWants()
	tests.RunToolInvokeTest(t, select1Want, invokeParamWant)
	tests.RunMCPToolCallMethod(t, mcpInvokeParamWant, failInvocationWant)
}

func setupMongoDB(t *testing.T, ctx context.Context, database *mongo.Database) func(*testing.T) {
	collectionName := "test_collection"

	documents := []map[string]any{
		{"_id": 1, "id": 1, "name": "Alice"},
		{"_id": 2, "id": 2, "name": "Jane"},
		{"_id": 3, "id": 3, "name": "Sid"},
		{"_id": 4, "id": 4, "name": "Bob"},
		{"_id": 5, "id": 3, "name": "Alice"},
	}
	for _, doc := range documents {
		_, err := database.Collection(collectionName).InsertOne(ctx, doc)
		if err != nil {
			t.Fatalf("unable to insert test data: %s", err)
		}
	}

	return func(t *testing.T) {
		// tear down test
		err := database.Collection(collectionName).Drop(ctx)
		if err != nil {
			t.Errorf("Teardown failed: %s", err)
		}
	}

}
