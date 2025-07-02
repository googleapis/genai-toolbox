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
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/googleapis/genai-toolbox/tests"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	MongoDbSourceKind   = "mongodb"
	MongoDbToolKind     = "mongodb-find"
	MongoDbUri          = os.Getenv("MONGODB_URI")
	MongoDbDatabase     = os.Getenv("MONGODB_DATABASE")
	ServiceAccountEmail = os.Getenv("SERVICE_ACCOUNT_EMAIL")
)

func getMongoDBVars(t *testing.T) map[string]any {
	switch "" {
	case MongoDbUri:
		t.Fatal("'MongoDbUri' not set")
	case MongoDbDatabase:
		t.Fatal("'MongoDbDatabase' not set")
	}
	return map[string]any{
		"kind": MongoDbSourceKind,
		"uri":  MongoDbUri,
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

	database, err := initMongoDbDatabase(ctx, MongoDbUri, MongoDbDatabase)
	if err != nil {
		t.Fatalf("unable to create MongoDB connection: %s", err)
	}

	// set up data for param tool
	teardownDB := setupMongoDB(t, ctx, database)
	defer teardownDB(t)

	// Write config into a file and pass it to command
	toolsFile := getMongoDBToolsConfig(sourceConfig, MongoDbToolKind)

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

	select1Want, failInvocationWant, invokeParamWant, mcpInvokeParamWant := getMongoDBWants()
	tests.RunToolInvokeTest(t, select1Want, invokeParamWant)
	tests.RunMCPToolCallMethod(t, mcpInvokeParamWant, failInvocationWant)
}

func setupMongoDB(t *testing.T, ctx context.Context, database *mongo.Database) func(*testing.T) {
	collectionName := "test_collection"

	documents := []map[string]any{
		{"_id": 1, "id": 1, "name": "Alice", "email": ServiceAccountEmail},
		{"_id": 2, "id": 2, "name": "Jane"},
		{"_id": 3, "id": 3, "name": "Sid"},
		{"_id": 4, "id": 4, "name": "Bob"},
		{"_id": 5, "id": 3, "name": "Alice", "email": "alice@gmail.com"},
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

func getMongoDBToolsConfig(sourceConfig map[string]any, toolKind string) map[string]any {
	toolsFile := map[string]any{
		"sources": map[string]any{
			"my-instance": sourceConfig,
		},
		"authServices": map[string]any{
			"my-google-auth": map[string]any{
				"kind":     "google",
				"clientId": tests.ClientId,
			},
		},
		"tools": map[string]any{
			"my-simple-tool": map[string]any{
				"kind":          "mongodb-find-one",
				"source":        "my-instance",
				"description":   "Simple tool to test end to end functionality.",
				"collection":    "test_collection",
				"filterPayload": `{ "_id" : 3 }`,
				"filterParams":  []any{},
			},
			"my-param-tool": map[string]any{
				"kind":          toolKind,
				"source":        "my-instance",
				"description":   "Tool to test invocation with params.",
				"authRequired":  []string{},
				"collection":    "test_collection",
				"filterPayload": `{ "id" : {{ .id }}, "name" : {{json .name }} }`,
				"filterParams": []map[string]any{
					{
						"name":        "id",
						"type":        "integer",
						"description": "user id",
					},
					{
						"name":        "name",
						"type":        "string",
						"description": "user name",
					},
				},
				"projectPayload": `{ "_id": 1, "id": 1, "name" : 1 }`,
			},
			"my-auth-tool": map[string]any{
				"kind":          toolKind,
				"source":        "my-instance",
				"description":   "Tool to test authenticated parameters.",
				"authRequired":  []string{},
				"collection":    "test_collection",
				"filterPayload": `{ "email" : {{json .email }} }`,
				"filterParams": []map[string]any{
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
				"projectPayload": `{ "_id": 0, "name" : 1 }`,
			},
			"my-auth-required-tool": map[string]any{
				"kind":        toolKind,
				"source":      "my-instance",
				"description": "Tool to test auth required invocation.",
				"authRequired": []string{
					"my-google-auth",
				},
				"collection":    "test_collection",
				"filterPayload": `{ "_id": 3, "id": 3 }`,
				"filterParams":  []any{},
			},
			"my-fail-tool": map[string]any{
				"kind":          toolKind,
				"source":        "my-instance",
				"description":   "Tool to test statement with incorrect syntax.",
				"authRequired":  []string{},
				"collection":    "test_collection",
				"filterPayload": `{ "id" ; 1 }"}`,
				"filterParams":  []any{},
			},
		},
	}

	return toolsFile

}

func getMongoDBWants() (string, string, string, string) {
	select1Want := `[{"_id":3,"id":3,"name":"Sid"}]`
	failInvocationWant := `invalid JSON input: missing colon after key `
	invokeParamWant := `[{"_id":5,"id":3,"name":"Alice"}]`
	mcpInvokeParamWant := `{"jsonrpc":"2.0","id":"my-param-tool","result":{"content":[{"type":"text","text":"{\"_id\":5,\"id\":3,\"name\":\"Alice\"}"}]}}`
	return select1Want, failInvocationWant, invokeParamWant, mcpInvokeParamWant
}
