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

package bigtable

import (
	"context"
	"fmt"

	"cloud.google.com/go/bigtable"
	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/util"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/api/option"
)

const SourceKind string = "bigtable"

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
	Name     string `yaml:"name" validate:"required"`
	Kind     string `yaml:"kind" validate:"required"`
	Project  string `yaml:"project" validate:"required"`
	Instance string `yaml:"instance" validate:"required"`
}

func (r Config) SourceConfigKind() string {
	return SourceKind
}

func (r Config) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {
	client, err := initBigtableClient(ctx, tracer, r.Name, r.Project, r.Instance)
	if err != nil {
		return nil, fmt.Errorf("unable to create client: %w", err)
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
	Client *bigtable.Client
}

func (s *Source) SourceKind() string {
	return SourceKind
}

func (s *Source) ToConfig() sources.SourceConfig {
	return s.Config
}

func (s *Source) BigtableClient() *bigtable.Client {
	return s.Client
}

func getBigtableType(paramType string) (bigtable.SQLType, error) {
	switch paramType {
	case "boolean":
		return bigtable.BoolSQLType{}, nil
	case "string":
		return bigtable.StringSQLType{}, nil
	case "integer":
		return bigtable.Int64SQLType{}, nil
	case "float":
		return bigtable.Float64SQLType{}, nil
	case "array":
		return bigtable.ArraySQLType{}, nil
	default:
		return nil, fmt.Errorf("unknow param type %s", paramType)
	}
}

func getMapParamsType(tparams parameters.Parameters) (map[string]bigtable.SQLType, error) {
	btParamTypes := make(map[string]bigtable.SQLType)
	for _, p := range tparams {
		if p.GetType() == "array" {
			itemType, err := getBigtableType(p.Manifest().Items.Type)
			if err != nil {
				return nil, err
			}
			btParamTypes[p.GetName()] = bigtable.ArraySQLType{
				ElemType: itemType,
			}
			continue
		}
		paramType, err := getBigtableType(p.GetType())
		if err != nil {
			return nil, err
		}
		btParamTypes[p.GetName()] = paramType
	}
	return btParamTypes, nil
}

func (s *Source) RunSQL(ctx context.Context, statement string, configParam parameters.Parameters, params parameters.ParamValues) (any, error) {
	mapParamsType, err := getMapParamsType(configParam)
	if err != nil {
		return nil, fmt.Errorf("fail to get map params: %w", err)
	}

	ps, err := s.BigtableClient().PrepareStatement(
		ctx,
		statement,
		mapParamsType,
	)
	if err != nil {
		return nil, fmt.Errorf("unable to prepare statement: %w", err)
	}

	bs, err := ps.Bind(params.AsMap())
	if err != nil {
		return nil, fmt.Errorf("unable to bind: %w", err)
	}

	var out []any
	var rowErr error
	err = bs.Execute(ctx, func(resultRow bigtable.ResultRow) bool {
		vMap := make(map[string]any)
		cols := resultRow.Metadata.Columns

		for _, c := range cols {
			var columValue any
			if err = resultRow.GetByName(c.Name, &columValue); err != nil {
				rowErr = err
				return false
			}
			vMap[c.Name] = columValue
		}

		out = append(out, vMap)

		return true
	})
	if err != nil {
		return nil, fmt.Errorf("unable to execute client: %w", err)
	}
	if rowErr != nil {
		return nil, fmt.Errorf("error processing row: %w", rowErr)
	}

	return out, nil
}

func initBigtableClient(ctx context.Context, tracer trace.Tracer, name, project, instance string) (*bigtable.Client, error) {
	//nolint:all // Reassigned ctx
	ctx, span := sources.InitConnectionSpan(ctx, tracer, SourceKind, name)
	defer span.End()

	// Set up Bigtable data operations client.
	poolSize := 10
	userAgent, err := util.UserAgentFromContext(ctx)
	if err != nil {
		return nil, err
	}

	client, err := bigtable.NewClient(ctx, project, instance, option.WithUserAgent(userAgent), option.WithGRPCConnectionPool(poolSize))

	if err != nil {
		return nil, fmt.Errorf("unable to create bigtable.NewClient: %w", err)
	}

	return client, nil
}
