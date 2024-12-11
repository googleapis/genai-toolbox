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

package spanner

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/spanner"
	"github.com/googleapis/genai-toolbox/internal/sources"
	spannerdb "github.com/googleapis/genai-toolbox/internal/sources/spanner"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"google.golang.org/api/iterator"
)

const ToolKind string = "spanner-sql"

type compatibleSource interface {
	SpannerClient() *spanner.Client
	DatabaseDialect() string
}

// validate compatible sources are still compatible
var _ compatibleSource = &spannerdb.Source{}

var compatibleSources = [...]string{spannerdb.SourceKind}

type Config struct {
	Name        string           `yaml:"name"`
	Kind        string           `yaml:"kind"`
	Source      string           `yaml:"source"`
	Description string           `yaml:"description"`
	Statement   string           `yaml:"statement"`
	Parameters  tools.Parameters `yaml:"parameters"`
}

// validate interface
var _ tools.ToolConfig = Config{}

func (cfg Config) ToolConfigKind() string {
	return ToolKind
}

func (cfg Config) Initialize(srcs map[string]sources.Source) (tools.Tool, error) {
	// verify source exists
	rawS, ok := srcs[cfg.Source]
	if !ok {
		return nil, fmt.Errorf("no source named %q configured", cfg.Source)
	}

	// verify the source is compatible
	s, ok := rawS.(compatibleSource)
	if !ok {
		return nil, fmt.Errorf("invalid source for %q tool: source kind must be one of %q", ToolKind, compatibleSources)
	}

	// finish tool setup
	t := Tool{
		Name:       cfg.Name,
		Kind:       ToolKind,
		Parameters: cfg.Parameters,
		Statement:  cfg.Statement,
		Client:     s.SpannerClient(),
		dialect:    s.DatabaseDialect(),
		manifest:   tools.Manifest{Description: cfg.Description, Parameters: cfg.Parameters.Manifest()},
	}
	return t, nil
}

func NewGenericTool(name, stmt, desc string, client *spanner.Client, dialect string, parameters tools.Parameters) Tool {
	return Tool{
		Name:       name,
		Kind:       ToolKind,
		Statement:  stmt,
		Client:     client,
		dialect:    dialect,
		manifest:   tools.Manifest{Description: desc, Parameters: parameters.Manifest()},
		Parameters: parameters,
	}
}

// validate interface
var _ tools.Tool = Tool{}

type Tool struct {
	Name       string           `yaml:"name"`
	Kind       string           `yaml:"kind"`
	Parameters tools.Parameters `yaml:"parameters"`

	Client    *spanner.Client
	dialect   string
	Statement string
	manifest  tools.Manifest
}

func getMapParams(params tools.ParamValues, dialect string) (map[string]interface{}, error) {
	switch strings.ToLower(dialect) {
	case "googlesql":
		return params.AsMap(), nil
	case "postgresql":
		return params.AsMapByOrderedKeys(), nil
	default:
		return nil, fmt.Errorf("invalid dialect %s", dialect)
	}
}

func (t Tool) Invoke(params tools.ParamValues) (string, error) {
	mapParams, err := getMapParams(params, t.dialect)
	if err != nil {
		return "", fmt.Errorf("fail to get map params: %w", err)
	}

	fmt.Printf("Invoked tool %s\n", t.Name)
	var out strings.Builder

	ctx := context.Background()
	_, err = t.Client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		stmt := spanner.Statement{
			SQL:    t.Statement,
			Params: mapParams,
		}
		iter := txn.Query(ctx, stmt)
		defer iter.Stop()

		for {
			row, err := iter.Next()
			if err == iterator.Done {
				return nil
			}
			if err != nil {
				return fmt.Errorf("unable to parse row: %w", err)
			}
			out.WriteString(row.String())
		}
	})
	if err != nil {
		return "", fmt.Errorf("unable to execute client: %w", err)
	}

	return fmt.Sprintf("Stub tool call for %q! Parameters parsed: %q \n Output: %s", t.Name, params, out.String()), nil
}

func (t Tool) ParseParams(data map[string]any) (tools.ParamValues, error) {
	return tools.ParseParams(t.Parameters, data)
}

func (t Tool) Manifest() tools.Manifest {
	return t.manifest
}
