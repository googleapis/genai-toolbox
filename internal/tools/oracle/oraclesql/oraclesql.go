// Copyright © 2025, Oracle and/or its affiliates.

package oraclesql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/sources/oracle"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
)

const kind string = "oracle-sql"

func init() {
	if !tools.Register(kind, newConfig) {
		panic(fmt.Sprintf("tool kind %q already registered", kind))
	}
}

func newConfig(ctx context.Context, name string, decoder *yaml.Decoder) (tools.ToolConfig, error) {
	actual := Config{Name: name}
	if err := decoder.DecodeContext(ctx, &actual); err != nil {
		return nil, err
	}
	return actual, nil
}

type compatibleSource interface {
	OracleDB() *sql.DB
}

// validate compatible sources are still compatible
var _ compatibleSource = &oracle.Source{}

var compatibleSources = [...]string{oracle.SourceKind}

type Config struct {
	Name               string                `yaml:"name" validate:"required"`
	Kind               string                `yaml:"kind" validate:"required"`
	Source             string                `yaml:"source" validate:"required"`
	Description        string                `yaml:"description" validate:"required"`
	Statement          string                `yaml:"statement" validate:"required"`
	AuthRequired       []string              `yaml:"authRequired"`
	Parameters         parameters.Parameters `yaml:"parameters"`
	TemplateParameters parameters.Parameters `yaml:"templateParameters"`
}

// validate interface
var _ tools.ToolConfig = Config{}

func (cfg Config) ToolConfigKind() string {
	return kind
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
		return nil, fmt.Errorf("invalid source for %q tool: source kind must be one of %q", kind, compatibleSources)
	}

	allParameters, paramManifest, err := parameters.ProcessParameters(cfg.TemplateParameters, cfg.Parameters)
	if err != nil {
		return nil, fmt.Errorf("error processing parameters: %w", err)
	}

	mcpManifest := tools.GetMcpManifest(cfg.Name, cfg.Description, cfg.AuthRequired, allParameters, nil)

	// finish tool setup
	t := Tool{
		Config:      cfg,
		AllParams:   allParameters,
		DB:          s.OracleDB(),
		manifest:    tools.Manifest{Description: cfg.Description, Parameters: paramManifest, AuthRequired: cfg.AuthRequired},
		mcpManifest: mcpManifest,
	}
	return t, nil
}

// validate interface
var _ tools.Tool = Tool{}

type Tool struct {
	Config
	AllParams parameters.Parameters `yaml:"allParams"`

	DB          *sql.DB
	manifest    tools.Manifest
	mcpManifest tools.McpManifest
}

func (t Tool) Invoke(ctx context.Context, params parameters.ParamValues, accessToken tools.AccessToken) (any, error) {
	paramsMap := params.AsMap()
	newStatement, err := parameters.ResolveTemplateParams(t.TemplateParameters, t.Statement, paramsMap)
	if err != nil {
		return nil, fmt.Errorf("unable to extract template params %w", err)
	}

	newParams, err := parameters.GetParams(t.Parameters, paramsMap)
	if err != nil {
		return nil, fmt.Errorf("unable to extract standard params %w", err)
	}
	sliceParams := newParams.AsSlice()

	for i, p := range sliceParams {
		fmt.Printf("[%d]=%T ", i, p)
	}
	fmt.Printf("\n")

	rows, err := t.DB.QueryContext(ctx, newStatement, sliceParams...)
	if err != nil {
		return nil, fmt.Errorf("unable to execute query: %w", err)
	}
	defer rows.Close()

	cols, _ := rows.Columns()

	// Get Column types
	colTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, fmt.Errorf("unable to get column types: %w", err)
	}

	var out []any
	for rows.Next() {
		values := make([]any, len(cols))
		for i, colType := range colTypes {
			switch strings.ToUpper(colType.DatabaseTypeName()) {
			case "NUMBER", "FLOAT", "BINARY_FLOAT", "BINARY_DOUBLE":
				if _, scale, ok := colType.DecimalSize(); ok && scale == 0 {
					// Scale is 0, treat it as an integer.
					values[i] = new(sql.NullInt64)
				} else {
					// Scale is non-zero or unknown, treat
					// it as a float.
					values[i] = new(sql.NullFloat64)
				}
			case "DATE", "TIMESTAMP", "TIMESTAMP WITH TIME ZONE", "TIMESTAMP WITH LOCAL TIME ZONE":
				values[i] = new(sql.NullTime)
			case "JSON":
				values[i] = new(sql.RawBytes)
			default:
				values[i] = new(sql.NullString)
			}
		}

		if err := rows.Scan(values...); err != nil {
			return nil, fmt.Errorf("unable to scan row: %w", err)
		}

		vMap := make(map[string]any)
		for i, col := range cols {
			receiver := values[i]

			switch v := receiver.(type) {
			case *sql.NullInt64:
				if v.Valid {
					vMap[col] = v.Int64
				} else {
					vMap[col] = nil
				}
			case *sql.NullFloat64:
				if v.Valid {
					vMap[col] = v.Float64
				} else {
					vMap[col] = nil
				}
			case *sql.NullString:
				if v.Valid {
					vMap[col] = v.String
				} else {
					vMap[col] = nil
				}
			case *sql.NullTime:
				if v.Valid {
					vMap[col] = v.Time
				} else {
					vMap[col] = nil
				}
			case *sql.RawBytes:
				if *v != nil {
					var unmarshaledData any
					if err := json.Unmarshal(*v, &unmarshaledData); err != nil {
						return nil, fmt.Errorf("unable to unmarshal json data for column %s", col)
					}
					vMap[col] = unmarshaledData
				} else {
					vMap[col] = nil
				}
			default:
				return nil, fmt.Errorf("unexpected receiver type: %T", v)
			}
		}
		out = append(out, vMap)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("errors encountered during query execution or row processing: %w", err)
	}

	return out, nil
}

func (t Tool) ParseParams(data map[string]any, claims map[string]map[string]any) (parameters.ParamValues, error) {
	return parameters.ParseParams(t.AllParams, data, claims)
}

func (t Tool) Manifest() tools.Manifest {
	return t.manifest
}

func (t Tool) McpManifest() tools.McpManifest {
	return t.mcpManifest
}

func (t Tool) Authorized(verifiedAuthServices []string) bool {
	return tools.IsAuthorized(t.AuthRequired, verifiedAuthServices)
}

func (t Tool) RequiresClientAuthorization() bool {
	return false
}

func (t Tool) ToConfig() tools.ToolConfig {
	return t.Config
}

func (t Tool) GetAuthTokenHeaderName() string {
	return "Authorization"
}
