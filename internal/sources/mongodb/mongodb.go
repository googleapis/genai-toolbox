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
	"encoding/json"
	"errors"
	"fmt"

	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/util"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.opentelemetry.io/otel/trace"
)

const SourceKind string = "mongodb"

// validate interface
var _ sources.SourceConfig = Config{}

func init() {
	if !sources.Register(SourceKind, newConfig) {
		panic(fmt.Sprintf("source kind %q already registered", SourceKind))
	}
}

func newConfig(ctx context.Context, name string, decoder *yaml.Decoder) (sources.SourceConfig, error) {
	actual := Config{Name: name}
	if err := decoder.DecodeContext(ctx, &actual); err != nil {
		return nil, err
	}
	return actual, nil
}

type Config struct {
	Name string `yaml:"name" validate:"required"`
	Kind string `yaml:"kind" validate:"required"`
	Uri  string `yaml:"uri" validate:"required"` // MongoDB Atlas connection URI
}

func (r Config) SourceConfigKind() string {
	return SourceKind
}

func (r Config) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {
	client, err := initMongoDBClient(ctx, tracer, r.Name, r.Uri)
	if err != nil {
		return nil, fmt.Errorf("unable to create MongoDB client: %w", err)
	}

	// Verify the connection
	err = client.Ping(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("unable to connect successfully: %w", err)
	}

	s := &Source{
		Config: r,
		Client: client,
	}
	return s, nil
}

var _ sources.Source = &Source{}

type Source struct {
	Config
	Client *mongo.Client
}

func (s *Source) SourceKind() string {
	return SourceKind
}

func (s *Source) ToConfig() sources.SourceConfig {
	return s.Config
}

func (s *Source) MongoClient() *mongo.Client {
	return s.Client
}

func parseData(ctx context.Context, cur *mongo.Cursor) ([]any, error) {
	var data = []any{}
	err := cur.All(ctx, &data)
	if err != nil {
		return nil, err
	}
	var final []any
	for _, item := range data {
		tmp, _ := bson.MarshalExtJSON(item, false, false)
		var tmp2 any
		err = json.Unmarshal(tmp, &tmp2)
		if err != nil {
			return nil, err
		}
		final = append(final, tmp2)
	}
	return final, err
}

func (s *Source) Aggregate(ctx context.Context, pipelineString string, canonical, readOnly bool, database, collection string) ([]any, error) {
	var pipeline = []bson.M{}
	err := bson.UnmarshalExtJSON([]byte(pipelineString), canonical, &pipeline)
	if err != nil {
		return nil, err
	}

	if readOnly {
		//fail if we do a merge or an out
		for _, stage := range pipeline {
			for key := range stage {
				if key == "$merge" || key == "$out" {
					return nil, fmt.Errorf("this is not a read-only pipeline: %+v", stage)
				}
			}
		}
	}

	cur, err := s.MongoClient().Database(database).Collection(collection).Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	res, err := parseData(ctx, cur)
	if err != nil {
		return nil, err
	}
	if res == nil {
		return []any{}, nil
	}
	return res, err
}

func (s *Source) Find(ctx context.Context, filterString, database, collection string, opts *options.FindOptions) ([]any, error) {
	var filter = bson.D{}
	err := bson.UnmarshalExtJSON([]byte(filterString), false, &filter)
	if err != nil {
		return nil, err
	}

	cur, err := s.MongoClient().Database(database).Collection(collection).Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	return parseData(ctx, cur)
}

func (s *Source) FindOne(ctx context.Context, filterString, database, collection string, opts *options.FindOneOptions) ([]any, error) {
	var filter = bson.D{}
	err := bson.UnmarshalExtJSON([]byte(filterString), false, &filter)
	if err != nil {
		return nil, err
	}

	res := s.MongoClient().Database(database).Collection(collection).FindOne(ctx, filter, opts)
	if res.Err() != nil {
		return nil, res.Err()
	}

	var data any
	err = res.Decode(&data)
	if err != nil {
		return nil, err
	}

	var final []any
	tmp, _ := bson.MarshalExtJSON(data, false, false)
	var tmp2 any
	err = json.Unmarshal(tmp, &tmp2)
	if err != nil {
		return nil, err
	}
	final = append(final, tmp2)

	return final, err
}

func (s *Source) InsertMany(ctx context.Context, jsonData string, canonical bool, database, collection string) ([]any, error) {
	var data = []any{}
	err := bson.UnmarshalExtJSON([]byte(jsonData), canonical, &data)
	if err != nil {
		return nil, err
	}

	res, err := s.MongoClient().Database(database).Collection(collection).InsertMany(ctx, data, options.InsertMany())
	if err != nil {
		return nil, err
	}
	return res.InsertedIDs, nil
}

func (s *Source) InsertOne(ctx context.Context, jsonData string, canonical bool, database, collection string) (any, error) {
	var data any
	err := bson.UnmarshalExtJSON([]byte(jsonData), canonical, &data)
	if err != nil {
		return nil, err
	}

	res, err := s.MongoClient().Database(database).Collection(collection).InsertOne(ctx, data, options.InsertOne())
	if err != nil {
		return nil, err
	}
	return res.InsertedID, nil
}

func (s *Source) UpdateMany(ctx context.Context, filterString string, canonical bool, updateString, database, collection string, upsert bool) ([]any, error) {
	var filter = bson.D{}
	err := bson.UnmarshalExtJSON([]byte(filterString), canonical, &filter)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal filter string: %w", err)
	}
	var update = bson.D{}
	err = bson.UnmarshalExtJSON([]byte(updateString), false, &update)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal update string: %w", err)
	}

	res, err := s.MongoClient().Database(database).Collection(collection).UpdateMany(ctx, filter, update, options.Update().SetUpsert(upsert))
	if err != nil {
		return nil, fmt.Errorf("error updating collection: %w", err)
	}
	return []any{res.ModifiedCount, res.UpsertedCount, res.MatchedCount}, nil
}

func (s *Source) UpdateOne(ctx context.Context, filterString string, canonical bool, updateString, database, collection string, upsert bool) (any, error) {
	var filter = bson.D{}
	err := bson.UnmarshalExtJSON([]byte(filterString), false, &filter)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal filter string: %w", err)
	}
	var update = bson.D{}
	err = bson.UnmarshalExtJSON([]byte(updateString), canonical, &update)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal update string: %w", err)
	}

	res, err := s.MongoClient().Database(database).Collection(collection).UpdateOne(ctx, filter, update, options.Update().SetUpsert(upsert))
	if err != nil {
		return nil, fmt.Errorf("error updating collection: %w", err)
	}
	return res.ModifiedCount, nil
}

func (s *Source) DeleteMany(ctx context.Context, filterString, database, collection string) (any, error) {
	var filter = bson.D{}
	err := bson.UnmarshalExtJSON([]byte(filterString), false, &filter)
	if err != nil {
		return nil, err
	}

	res, err := s.MongoClient().Database(database).Collection(collection).DeleteMany(ctx, filter, options.Delete())
	if err != nil {
		return nil, err
	}

	if res.DeletedCount == 0 {
		return nil, errors.New("no document found")
	}
	return res.DeletedCount, nil
}

func (s *Source) DeleteOne(ctx context.Context, filterString, database, collection string) (any, error) {
	var filter = bson.D{}
	err := bson.UnmarshalExtJSON([]byte(filterString), false, &filter)
	if err != nil {
		return nil, err
	}

	res, err := s.MongoClient().Database(database).Collection(collection).DeleteOne(ctx, filter, options.Delete())
	if err != nil {
		return nil, err
	}
	return res.DeletedCount, nil
}

func initMongoDBClient(ctx context.Context, tracer trace.Tracer, name, uri string) (*mongo.Client, error) {
	// Start a tracing span
	ctx, span := sources.InitConnectionSpan(ctx, tracer, SourceKind, name)
	defer span.End()

	userAgent, err := util.UserAgentFromContext(ctx)
	if err != nil {
		return nil, err
	}

	// Create a new MongoDB client
	clientOpts := options.Client().ApplyURI(uri).SetAppName(userAgent)
	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return nil, fmt.Errorf("unable to create MongoDB client: %w", err)
	}

	return client, nil
}
