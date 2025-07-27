package cassandracql

import (
	"context"
	"fmt"

	yaml "github.com/goccy/go-yaml"
	"github.com/gocql/gocql"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/sources/cassandra"
	"github.com/googleapis/genai-toolbox/internal/tools"
)

const kind string = "cassandra-cql"

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
	CassandraSession() *gocql.Session
}

var _ compatibleSource = &cassandra.Source{}

var compatibleSources = [...]string{cassandra.SourceKind}

type Config struct {
	Name               string           `yaml:"name" validate:"required"`
	Kind               string           `yaml:"kind" validate:"required"`
	Source             string           `yaml:"source" validate:"required"`
	Description        string           `yaml:"description" validate:"required"`
	Statement          string           `yaml:"statement" validate:"required"`
	AuthRequired       []string         `yaml:"authRequired"`
	Parameters         tools.Parameters `yaml:"parameters"`
	TemplateParameters tools.Parameters `yaml:"templateParameters"`
}

// Initialize implements tools.ToolConfig.
func (c Config) Initialize(srcs map[string]sources.Source) (tools.Tool, error) {
	// verify source exists
	rawS, ok := srcs[c.Source]
	if !ok {
		return nil, fmt.Errorf("no source named %q configured", c.Source)
	}

	// verify the source is compatible
	s, ok := rawS.(compatibleSource)
	if !ok {
		return nil, fmt.Errorf("invalid source for %q tool: source kind must be one of %q", kind, compatibleSources)
	}

	allParameters, paramManifest, paramMcpManifest := tools.ProcessParameters(c.TemplateParameters, c.Parameters)

	mcpManifest := tools.McpManifest{
		Name:        c.Name,
		Description: c.Description,
		InputSchema: paramMcpManifest,
	}

	t := Tool{
		Name:               c.Name,
		Kind:               kind,
		Parameters:         c.Parameters,
		TemplateParameters: c.TemplateParameters,
		AllParams:          allParameters,
		Statement:          c.Statement,
		AuthRequired:       c.AuthRequired,
		Session:            s.CassandraSession(),
		manifest:           tools.Manifest{Description: c.Description, Parameters: paramManifest, AuthRequired: c.AuthRequired},
		mcpManifest:        mcpManifest,
	}
	return t, nil
}

// ToolConfigKind implements tools.ToolConfig.
func (c Config) ToolConfigKind() string {
	return kind
}

var _ tools.ToolConfig = Config{}

type Tool struct {
	Name               string           `yaml:"name"`
	Kind               string           `yaml:"kind"`
	AuthRequired       []string         `yaml:"authRequired"`
	Parameters         tools.Parameters `yaml:"parameters"`
	TemplateParameters tools.Parameters `yaml:"templateParameters"`
	AllParams          tools.Parameters `yaml:"allParams"`

	Session     *gocql.Session
	Statement   string
	manifest    tools.Manifest
	mcpManifest tools.McpManifest
}

// Authorized implements tools.Tool.
func (t Tool) Authorized(verifiedAuthServices []string) bool {
	return tools.IsAuthorized(t.AuthRequired, verifiedAuthServices)
}

// Invoke implements tools.Tool.
func (t Tool) Invoke(ctx context.Context, params tools.ParamValues) (any, error) {
	paramsMap := params.AsMap()
	newStatement, err := tools.ResolveTemplateParams(t.TemplateParameters, t.Statement, paramsMap)
	if err != nil {
		return nil, fmt.Errorf("unable to extract template params %w", err)
	}

	newParams, err := tools.GetParams(t.Parameters, paramsMap)
	if err != nil {
		return nil, fmt.Errorf("unable to extract standard params %w", err)
	}
	sliceParams := newParams.AsSlice()
	iter := t.Session.Query(newStatement, sliceParams...).WithContext(ctx).Iter()

	// Create a slice to store the out
	var out []map[string]interface{}

	// Scan results into a map and append to the slice
	for {
		row := make(map[string]interface{}) // Create a new map for each row
		if !iter.MapScan(row) {
			break // No more rows
		}
		out = append(out, row)
	}

	if err := iter.Close(); err != nil {
		return nil, fmt.Errorf("unable to parse rows: %w", err)
	}
	return out, nil
}

// Manifest implements tools.Tool.
func (t Tool) Manifest() tools.Manifest {
	return t.manifest
}

// McpManifest implements tools.Tool.
func (t Tool) McpManifest() tools.McpManifest {
	return t.mcpManifest
}

// ParseParams implements tools.Tool.
func (t Tool) ParseParams(data map[string]any, claims map[string]map[string]any) (tools.ParamValues, error) {
	return tools.ParseParams(t.AllParams, data, claims)
}

var _ tools.Tool = Tool{}
