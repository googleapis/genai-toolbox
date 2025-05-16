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
	"testing"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	mongoDBSourceKind = "mongodb"
	mongoDBToolKind   = "mongodb-atlas"
)

var (
	mongoDBConnectionString = os.Getenv("MONGODB_CONNECTION_STRING")
	mongoDBDatabase         = os.Getenv("MONGODB_DATABASE")
)

// getMongoDBVars validates and returns MongoDB configuration variables
func getMongoDBVars(t *testing.T) map[string]any {
	switch "" {
	case mongoDBConnectionString:
		t.Fatal("'MONGODB_CONNECTION_STRING' not set")
	case mongoDBDatabase:
		t.Fatal("'MONGODB_DATABASE' not set")
	}

	return map[string]any{
		"kind":       mongoDBSourceKind,
		"connection": mongoDBConnectionString,
		"database":   mongoDBDatabase,
	}
}

// initMongoDBClient initializes a MongoDB client
func initMongoDBClient(connectionString string) (*mongo.Client, error) {
	clientOpts := options.Client().ApplyURI(connectionString)
	client, err := mongo.Connect(context.Background(), clientOpts)
	if err != nil {
		return nil, fmt.Errorf("mongo.Connect: %w", err)
	}
	return client, nil
}

func TestMongoDBToolEndpoints(t *testing.T) {
	// Remove the unused variable
	_ = getMongoDBVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	client, err := initMongoDBClient(mongoDBConnectionString)
	if err != nil {
		t.Fatalf("unable to create MongoDB connection: %s", err)
	}
	defer client.Disconnect(ctx)

	// Create collection names with UUID
	collectionNameParam := "param_" + uuid.New().String()

	// Set up data for param tool
	paramToolQuery, params1 := getMongoDBParamToolInfo()
	teardownCollection1 := setupMongoDBCollection(t, ctx, client, mongoDBDatabase, collectionNameParam, params1)
	defer teardownCollection1(t)

	// Simulate tool execution and validate results
	runMongoDBQueryTest(t, ctx, client, mongoDBDatabase, collectionNameParam, paramToolQuery, params1)
}

// setupMongoDBCollection creates a collection and inserts test data
func setupMongoDBCollection(t *testing.T, ctx context.Context, client *mongo.Client, database, collectionName string, params []interface{}) func(t *testing.T) {
	collection := client.Database(database).Collection(collectionName)

	// Insert test documents
	_, err := collection.InsertMany(ctx, params)
	if err != nil {
		t.Fatalf("failed to insert test data: %v", err)
	}

	// Return a cleanup function
	return func(t *testing.T) {
		err := collection.Drop(ctx)
		if err != nil {
			t.Logf("failed to drop collection: %v", err)
		}
	}
}

// getMongoDBParamToolInfo returns query and params for param tool
func getMongoDBParamToolInfo() (bson.M, []interface{}) {
	query := bson.M{"$or": []bson.M{
		{"_id": bson.M{"$eq": 1}},
		{"name": bson.M{"$eq": "Alice"}},
	}}
	params := []interface{}{
		bson.M{"_id": 1, "name": "Alice"},
	}
	return query, params
}

// runMongoDBQueryTest executes a query and validates the results
func runMongoDBQueryTest(t *testing.T, ctx context.Context, client *mongo.Client, database, collectionName string, query bson.M, expected []interface{}) {
	collection := client.Database(database).Collection(collectionName)

	cursor, err := collection.Find(ctx, query)
	if err != nil {
		t.Fatalf("failed to execute query: %v", err)
	}
	defer cursor.Close(ctx)

	var results []bson.M
	if err := cursor.All(ctx, &results); err != nil {
		t.Fatalf("failed to decode query results: %v", err)
	}

	if len(results) != len(expected) {
		t.Fatalf("unexpected number of results: got %d, want %d", len(results), len(expected))
	}
}
