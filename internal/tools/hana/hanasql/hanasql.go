package hanasql

import (
    "context"
    "database/sql"
    "fmt"

    yaml "github.com/goccy/go-yaml"
    "github.com/googleapis/genai-toolbox/internal/sources"
    "github.com/googleapis/genai-toolbox/internal/sources/hana"
    "github.com/googleapis/genai-toolbox/internal/tools"
)

const kind string = "hana-sql"

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
    HanaDB() *sql.DB
}

// Validate compatible sources compile-time.
var _ compatibleSource = &hana.Source{}

var compatibleSources = [...]string{hana.SourceKind}

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

var _ tools.ToolConfig = Config{}

func (cfg Config) ToolConfigKind() string { return kind }

func (cfg Config) Initialize(srcs map[string]sources.Source) (tools.Tool, error) {
    rawS, ok := srcs[cfg.Source]
    if !ok {
        return nil, fmt.Errorf("no source named %q configured", cfg.Source)
    }

    s, ok := rawS.(compatibleSource)
    if !ok {
        return nil, fmt.Errorf("invalid source for %q tool: source kind must be one of %q", kind, compatibleSources)
    }

    allParameters, paramManifest, paramMcpManifest := tools.ProcessParameters(cfg.TemplateParameters, cfg.Parameters)

    mcpManifest := tools.McpManifest{
        Name:        cfg.Name,
        Description: cfg.Description,
        InputSchema: paramMcpManifest,
    }

    t := Tool{
        Name:               cfg.Name,
        Kind:               kind,
        Parameters:         cfg.Parameters,
        TemplateParameters: cfg.TemplateParameters,
        AllParams:          allParameters,
        Statement:          cfg.Statement,
        AuthRequired:       cfg.AuthRequired,
        DB:                 s.HanaDB(),
        manifest:           tools.Manifest{Description: cfg.Description, Parameters: paramManifest, AuthRequired: cfg.AuthRequired},
        mcpManifest:        mcpManifest,
    }
    return t, nil
}

// Tool implementation
var _ tools.Tool = Tool{}

type Tool struct {
    Name               string           `yaml:"name"`
    Kind               string           `yaml:"kind"`
    AuthRequired       []string         `yaml:"authRequired"`
    Parameters         tools.Parameters `yaml:"parameters"`
    TemplateParameters tools.Parameters `yaml:"templateParameters"`
    AllParams          tools.Parameters `yaml:"allParams"`

    DB          *sql.DB
    Statement   string
    manifest    tools.Manifest
    mcpManifest tools.McpManifest
}

func (t Tool) Invoke(ctx context.Context, params tools.ParamValues) ([]any, error) {
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
    rows, err := t.DB.QueryContext(ctx, newStatement, sliceParams...)
    if err != nil {
        return nil, fmt.Errorf("unable to execute query: %w", err)
    }
    defer rows.Close()

    cols, err := rows.Columns()
    if err != nil {
        return nil, fmt.Errorf("unable to retrieve column names: %w", err)
    }

    rawVals := make([]any, len(cols))
    scanArgs := make([]any, len(cols))
    for i := range rawVals {
        scanArgs[i] = &rawVals[i]
    }

    // Retrieve column types for potential future enhancements (e.g. better type mapping).
    // We intentionally ignore the returned slice for now to avoid unused variable linting errors.
    if _, err := rows.ColumnTypes(); err != nil {
        return nil, fmt.Errorf("unable to get column types: %w", err)
    }

    var out []any
    for rows.Next() {
        if err := rows.Scan(scanArgs...); err != nil {
            return nil, fmt.Errorf("unable to parse row: %w", err)
        }
        vMap := make(map[string]any)
        for i, name := range cols {
            val := rawVals[i]
            if val == nil {
                vMap[name] = nil
                continue
            }
            // The go-hdb driver returns []byte for string-like types.
            // Convert to string for JSON friendliness.
            switch v := val.(type) {
            case []byte:
                vMap[name] = string(v)
            default:
                vMap[name] = v
            }
        }
        out = append(out, vMap)
    }

    if err := rows.Err(); err != nil {
        return nil, fmt.Errorf("errors encountered during row iteration: %w", err)
    }

    return out, nil
}

func (t Tool) ParseParams(data map[string]any, claims map[string]map[string]any) (tools.ParamValues, error) {
    return tools.ParseParams(t.AllParams, data, claims)
}

func (t Tool) Manifest() tools.Manifest { return t.manifest }

func (t Tool) McpManifest() tools.McpManifest { return t.mcpManifest }

func (t Tool) Authorized(verifiedAuthServices []string) bool {
    return tools.IsAuthorized(t.AuthRequired, verifiedAuthServices)
} 