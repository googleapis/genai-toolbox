// Copyright © 2025, Oracle and/or its affiliates.

package oraclesql

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"

	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/mcp-toolbox/internal/sources"
	"github.com/googleapis/mcp-toolbox/internal/tools"
	"github.com/googleapis/mcp-toolbox/internal/util"
	"github.com/googleapis/mcp-toolbox/internal/util/parameters"
)

const resourceType string = "oracle-sql"

func init() {
	if !tools.Register(resourceType, newConfig) {
		panic(fmt.Sprintf("tool type %q already registered", resourceType))
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
	RunSQL(context.Context, string, []any, bool) (any, error)
}

type Config struct {
	Name               string                 `yaml:"name" validate:"required"`
	Type               string                 `yaml:"type" validate:"required"`
	Source             string                 `yaml:"source" validate:"required"`
	Description        string                 `yaml:"description" validate:"required"`
	Statement          string                 `yaml:"statement" validate:"required"`
	ReadOnly           *bool                  `yaml:"readOnly"`
	AuthRequired       []string               `yaml:"authRequired"`
	Parameters         parameters.Parameters  `yaml:"parameters"`
	TemplateParameters parameters.Parameters  `yaml:"templateParameters"`
	Annotations        *tools.ToolAnnotations `yaml:"annotations,omitempty"`

	ScopesRequired []string `yaml:"scopesRequired"`
}

// validate interface
var _ tools.ToolConfig = Config{}

func (cfg Config) ToolConfigType() string {
	return resourceType
}

func (cfg Config) Initialize(srcs map[string]sources.Source) (tools.Tool, error) {
	allParameters, paramManifest, err := parameters.ProcessParameters(cfg.TemplateParameters, cfg.Parameters)
	if err != nil {
		return nil, fmt.Errorf("error processing parameters: %w", err)
	}

	return Tool{
		BaseTool: tools.BaseTool{
			Name:             cfg.Name,
			Description:      cfg.Description,
			Metadata:         tools.Manifest{Description: cfg.Description, Parameters: paramManifest, AuthRequired: cfg.AuthRequired},
			StaticParameters: allParameters,
			ScopesRequired:   cfg.ScopesRequired,
			Annotations:      tools.GetAnnotationsOrDefault(cfg.Annotations, tools.NewDestructiveAnnotations),
		},
		cfg: cfg,
	}, nil
}

// validate interface
var _ tools.Tool = Tool{}

type Tool struct {
	tools.BaseTool
	cfg Config
}

func (t Tool) ToConfig() tools.ToolConfig {
	return t.cfg
}

func (t Tool) Invoke(ctx context.Context, resourceMgr tools.SourceProvider, params parameters.ParamValues, accessToken tools.AccessToken) (any, util.ToolboxError) {
	source, err := tools.GetCompatibleSource[compatibleSource](resourceMgr, t.cfg.Source, t.cfg.Name, t.cfg.Type)
	if err != nil {
		return nil, util.NewClientServerError("source used is not compatible with the tool", http.StatusInternalServerError, err)
	}

	paramsMap := params.AsMap()
	newStatement, err := parameters.ResolveTemplateParams(t.cfg.TemplateParameters, t.cfg.Statement, paramsMap)
	if err != nil {
		return nil, util.NewAgentError("unable to extract template params", err)
	}

	newParams, err := parameters.GetParams(t.cfg.Parameters, paramsMap)
	if err != nil {
		return nil, util.NewAgentError("unable to extract standard params", err)
	}
	sliceParams := newParams.AsSlice()

	for i, p := range sliceParams {
		fmt.Printf("[%d]=%T ", i, p)
	}
	fmt.Printf("\n")

	isReadOnly := true
	if t.cfg.ReadOnly != nil {
		isReadOnly = *t.cfg.ReadOnly
	}

	resp, err := source.RunSQL(ctx, newStatement, sliceParams, isReadOnly)

	if err != nil {
		return nil, util.ProcessGeneralError(err)
	}
	return resp, nil
}
