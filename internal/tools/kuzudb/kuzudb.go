package kuzudb

import (
	"context"
	"fmt"

	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/sources/kuzudb"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/kuzudb/go-kuzu"
)

var kind string = "kuzudb-cypher"

func init() {
	if !tools.Register(kind, newConfig) {
		panic(fmt.Sprintf("tool kind %q already registered", kind))
	}
}

func newConfig(ctx context.Context, name string, decoder *yaml.Decoder) (tools.ToolConfig, error) {
	actual := KuzuDBToolConfig{Name: name}
	if err := decoder.DecodeContext(ctx, &actual); err != nil {
		return nil, err
	}
	return actual, nil
}

type KuzuDBToolConfig struct {
	Name         string           `yaml:"name" validate:"required"`
	Kind         string           `yaml:"kind" validate:"required"`
	Source       string           `yaml:"source" validate:"required"`
	Description  string           `yaml:"description" validate:"required"`
	Statement    string           `yaml:"statement" validate:"required"`
	AuthRequired []string         `yaml:"authRequired"`
	Parameters   tools.Parameters `yaml:"parameters"`
}

type compatibleSource interface {
	KuzuDB() *kuzu.Connection
}

// validate compatible sources are still compatible
var _ compatibleSource = &kuzudb.KuzuDbSource{}
var compatibleSources = [...]string{kuzudb.KuzuDbKind}

// Initialize implements tools.ToolConfig.
func (k KuzuDBToolConfig) Initialize(srcs map[string]sources.Source) (tools.Tool, error) {
	rawS, ok := srcs[k.Source]
	if !ok {
		return nil, fmt.Errorf("no source named %q configured", k.Source)
	}

	// verify the source is compatible
	s, ok := rawS.(compatibleSource)
	if !ok {
		return nil, fmt.Errorf("invalid source for %q tool: source kind must be one of %q", kind, compatibleSources)
	}

	mcpManifest := tools.McpManifest{
		Name:        k.Name,
		Description: k.Description,
		InputSchema: k.Parameters.McpManifest(),
	}

	// finish tool setup
	t := KuzuDBTool{
		Name:         k.Name,
		Kind:         kind,
		Parameters:   k.Parameters,
		Statement:    k.Statement,
		AuthRequired: k.AuthRequired,
		Connection:   s.KuzuDB(),
		manifest:     tools.Manifest{Description: k.Description, Parameters: k.Parameters.Manifest(), AuthRequired: k.AuthRequired},
		mcpManifest:  mcpManifest,
	}
	return t, nil
}

// ToolConfigKind implements tools.ToolConfig.
func (k KuzuDBToolConfig) ToolConfigKind() string {
	return kind
}

var _ tools.ToolConfig = KuzuDBToolConfig{}

type KuzuDBTool struct {
	Name string `yaml:"name" validate:"required"`

	Kind         string           `yaml:"kind"`
	AuthRequired []string         `yaml:"authRequired"`
	Parameters   tools.Parameters `yaml:"parameters"`

	Connection  *kuzu.Connection
	Statement   string `yaml:"statement"`
	manifest    tools.Manifest
	mcpManifest tools.McpManifest
}

// Authorized implements tools.Tool.
func (k KuzuDBTool) Authorized(verifiedAuthServices []string) bool {
	return tools.IsAuthorized(k.AuthRequired, verifiedAuthServices)
}

// Invoke implements tools.Tool.
func (k KuzuDBTool) Invoke(ctx context.Context, params tools.ParamValues) ([]any, error) {
	conn := k.Connection
	paramsMap := params.AsMap()

	preparedStatement, err := conn.Prepare(k.Statement)
	if err != nil {
		return nil, fmt.Errorf("unable to generate prepared statement %w", err)
	}

	result, err := conn.Execute(preparedStatement, paramsMap)
	if err != nil {
		return nil, fmt.Errorf("unable to execute query: %w", err)
	}
	defer result.Close()
	cols := result.GetColumnNames()
	var out []any
	for result.HasNext() {
		tuple, err := result.Next()
		if err != nil {
			return nil, fmt.Errorf("unable to parse row: %w", err)
		}
		defer tuple.Close()

		// The result is a tuple, which can be converted to a slice.
		slice, err := tuple.GetAsSlice()
		if err != nil {
			return nil, fmt.Errorf("unable to slice row: %w", err)
		}
		rowMap := make(map[string]interface{})
		for i, col := range cols {
			val := slice[i]
			// Store the value in the map
			rowMap[col] = val
		}
		out = append(out, rowMap)
	}
	return out, nil
}

// Manifest implements tools.Tool.
func (k KuzuDBTool) Manifest() tools.Manifest {
	return k.manifest
}

// McpManifest implements tools.Tool.
func (k KuzuDBTool) McpManifest() tools.McpManifest {
	return k.mcpManifest
}

// ParseParams implements tools.Tool.
func (k KuzuDBTool) ParseParams(data map[string]any, claimsMap map[string]map[string]any) (tools.ParamValues, error) {
	return tools.ParseParams(k.Parameters, data, claimsMap)
}

var _ tools.Tool = KuzuDBTool{}
