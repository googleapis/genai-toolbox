// Copyright 2026 Google LLC
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

package falkordb

import (
	"context"
	"fmt"

	"github.com/FalkorDB/falkordb-go/v2"
	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/tools/neo4j/neo4jexecutecypher/classifier"
	"github.com/googleapis/genai-toolbox/internal/util"
	"go.opentelemetry.io/otel/trace"
)

const SourceType string = "falkordb"

var sourceClassifier *classifier.QueryClassifier = classifier.NewQueryClassifier()

// validate interface
var _ sources.SourceConfig = Config{}

func init() {
	if !sources.Register(SourceType, newConfig) {
		panic(fmt.Sprintf("source type %q already registered", SourceType))
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
	Name     string `yaml:"name" validate:"required"`
	Type     string `yaml:"type" validate:"required"`
	Addr     string `yaml:"addr" validate:"required"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Graph    string `yaml:"graph" validate:"required"`
}

func (r Config) SourceConfigType() string {
	return SourceType
}

func (r Config) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {
	db, graph, err := initFalkorDBConnection(ctx, tracer, r.Addr, r.Username, r.Password, r.Graph, r.Name)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection: %w", err)
	}

	s := &Source{
		Config:   r,
		FalkorDB: db,
		Graph:    graph,
	}
	return s, nil
}

var _ sources.Source = &Source{}

type Source struct {
	Config
	FalkorDB *falkordb.FalkorDB
	Graph    *falkordb.Graph
}

func (s *Source) SourceType() string {
	return SourceType
}

func (s *Source) ToConfig() sources.SourceConfig {
	return s.Config
}

func (s *Source) FalkorDBGraph() *falkordb.Graph {
	return s.Graph
}

func (s *Source) RunQuery(ctx context.Context, cypherStr string, params map[string]any, readOnly, dryRun bool) (any, error) {
	// validate the cypher query before executing
	cf := sourceClassifier.Classify(cypherStr)
	if cf.Error != nil {
		return nil, cf.Error
	}

	if cf.Type == classifier.WriteQuery && readOnly {
		return nil, fmt.Errorf("this tool is read-only and cannot execute write queries")
	}

	if dryRun {
		// FalkorDB's ExecutionPlan returns a string representation of the query plan
		plan, err := s.Graph.ExecutionPlan(cypherStr)
		if err != nil {
			return nil, fmt.Errorf("unable to get execution plan: %w", err)
		}
		return []map[string]any{{"executionPlan": plan}}, nil
	}

	var result *falkordb.QueryResult
	var err error

	if readOnly || cf.Type == classifier.ReadQuery {
		result, err = s.Graph.ROQuery(cypherStr, params, nil)
	} else {
		result, err = s.Graph.Query(cypherStr, params, nil)
	}

	if err != nil {
		return nil, fmt.Errorf("unable to execute query: %w", err)
	}

	var out []map[string]any
	for result.Next() {
		record := result.Record()
		vMap := make(map[string]any)
		for i, key := range record.Keys() {
			vMap[key] = convertValue(record.Values()[i])
		}
		out = append(out, vMap)
	}

	return out, nil
}

// convertValue converts FalkorDB values to JSON-compatible values
func convertValue(value any) any {
	switch v := value.(type) {
	case nil:
		return nil
	case bool, string, int, int8, int16, int32, int64, float32, float64:
		return v
	case *falkordb.Node:
		return map[string]any{
			"id":         v.ID,
			"labels":     v.Labels,
			"properties": convertValue(v.Properties),
		}
	case falkordb.Node:
		return map[string]any{
			"id":         v.ID,
			"labels":     v.Labels,
			"properties": convertValue(v.Properties),
		}
	case *falkordb.Edge:
		return map[string]any{
			"id":         v.ID,
			"type":       v.Relation,
			"srcNode":    v.SourceNodeID(),
			"destNode":   v.DestNodeID(),
			"properties": convertValue(v.Properties),
		}
	case falkordb.Edge:
		return map[string]any{
			"id":         v.ID,
			"type":       v.Relation,
			"srcNode":    v.Source.ID,
			"destNode":   v.Destination.ID,
			"properties": convertValue(v.Properties),
		}
	case falkordb.Path:
		var nodes []any
		var edges []any
		for _, n := range v.Nodes {
			nodes = append(nodes, convertValue(n))
		}
		for _, e := range v.Edges {
			edges = append(edges, convertValue(e))
		}
		return map[string]any{
			"nodes": nodes,
			"edges": edges,
		}
	case []any:
		arr := make([]any, len(v))
		for i, elem := range v {
			arr[i] = convertValue(elem)
		}
		return arr
	case map[string]any:
		m := make(map[string]any)
		for key, val := range v {
			m[key] = convertValue(val)
		}
		return m
	}
	return fmt.Sprintf("%v", value)
}

func initFalkorDBConnection(ctx context.Context, tracer trace.Tracer, addr, username, password, graphName, sourceName string) (*falkordb.FalkorDB, *falkordb.Graph, error) {
	//nolint:all // Reassigned ctx
	ctx, span := sources.InitConnectionSpan(ctx, tracer, SourceType, sourceName)
	defer span.End()

	userAgent, err := util.UserAgentFromContext(ctx)
	if err != nil {
		return nil, nil, err
	}

	// Create FalkorDB connection options
	opts := &falkordb.ConnectionOption{
		Addr:       addr,
		ClientName: userAgent,
	}

	if username != "" {
		opts.Username = username
	}

	if password != "" {
		opts.Password = password
	}

	db, err := falkordb.FalkorDBNew(opts)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create FalkorDB client: %w", err)
	}

	// Select the graph
	graph := db.SelectGraph(graphName)

	// Verify connectivity by running a simple query
	_, err = graph.ROQuery("RETURN 1", nil, nil)
	if err != nil {
		// Close the underlying Redis connection if connectivity verification fails.
		if closeErr := db.Conn.Close(); closeErr != nil {
			return nil, nil, fmt.Errorf("unable to verify connectivity: %w; also failed to close connection: %w", err, closeErr)
		}
		return nil, nil, fmt.Errorf("unable to verify connectivity: %w", err)
	}

	return db, graph, nil
}
