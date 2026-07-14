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

package odata

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/mcp-toolbox/internal/sources"
	"github.com/googleapis/mcp-toolbox/internal/sources/odata"
	"github.com/googleapis/mcp-toolbox/internal/tools"
	"github.com/googleapis/mcp-toolbox/internal/util"
	"github.com/googleapis/mcp-toolbox/internal/util/parameters"
)

const resourceType string = "odata"

func init() {
	if !tools.Register(resourceType, newConfig) {
		panic(fmt.Sprintf("tool type %q already registered", resourceType))
	}
}

func newConfig(ctx context.Context, name string, decoder *yaml.Decoder) (tools.ToolConfig, error) {
	actual := Config{ConfigBase: tools.ConfigBase{Name: name}}
	if err := decoder.DecodeContext(ctx, &actual); err != nil {
		return nil, err
	}
	return actual, nil
}

type compatibleSource interface {
	HttpBaseURL() string
	RunSAPRequest(*http.Request, tools.AccessToken) (any, error)
	Metadata() *odata.ODataMetadata
	Compatibility() odata.CompatibilityConfig
	UseClientAuthorization() bool
	GetAuthTokenHeaderName() string
}

type Config struct {
	tools.ConfigBase `yaml:",inline"`
	Type             string                 `yaml:"type" validate:"required"`
	Source           string                 `yaml:"source" validate:"required"`
	EntitySet        string                 `yaml:"entitySet" validate:"required"`
	Operation        string                 `yaml:"operation" validate:"required"` // READ, CREATE, UPDATE, DELETE, FUNCTION_IMPORT
	ContentType      string                 `yaml:"contentType"`                   // Override default application/json
	QueryParams      parameters.Parameters  `yaml:"queryParams"`
	BodyParams       parameters.Parameters  `yaml:"bodyParams"`
	Annotations      *tools.ToolAnnotations `yaml:"annotations,omitempty"`
}

var _ tools.ToolConfig = Config{}

func (cfg Config) ToolConfigType() string {
	return resourceType
}

func (cfg Config) Initialize(ctx context.Context) (tools.Tool, error) {
	method, params := buildParams(cfg.Operation, cfg.EntitySet, cfg.BodyParams, cfg.QueryParams, nil)

	var defaultAnnotationsFn func() *tools.ToolAnnotations
	if strings.ToUpper(cfg.Operation) == "READ" {
		defaultAnnotationsFn = tools.NewReadOnlyAnnotations
	} else {
		defaultAnnotationsFn = tools.NewDestructiveAnnotations
	}
	annotations := tools.GetAnnotationsOrDefault(cfg.Annotations, defaultAnnotationsFn)

	return Tool{
		BaseTool: tools.NewBaseTool(
			cfg,
			annotations,
			tools.Manifest{Description: cfg.Description, Parameters: params.Manifest(), AuthRequired: cfg.AuthRequired},
			params,
		),
		Method: method,
	}, nil
}

var _ tools.Tool = Tool{}

type Tool struct {
	tools.BaseTool[Config]
	Method string
}

func (t Tool) ToConfig() tools.ToolConfig {
	return t.Cfg
}

// applyODataFormatting automatically applies OData syntax transformations to values, incorporating SAP-specific compatibility flags.
func applyODataFormatting(value string, paramName string, paramType string, mdVersion string, isUrlParam bool, compat odata.CompatibilityConfig) string {
	if compat.SapUrlQuoting {
		// Intelligent Casing Heuristic for SAP
		upperNames := []string{"ID", "CODE", "CARRID", "CONNID", "KEY", "NUM", "CITY", "COUNTRY", "PLANT", "COMPANY", "CURRENCY"}
		upper := strings.ToUpper(paramName)
		shouldUpper := false
		for _, n := range upperNames {
			if strings.Contains(upper, n) {
				shouldUpper = true
				break
			}
		}
		if shouldUpper && !strings.Contains(upper, "DESCRIPTION") && !strings.Contains(upper, "NOTE") {
			value = strings.ToUpper(value)
		}

		// OData v2 requires string literals in the URL to be single quoted.
		if isUrlParam && mdVersion == "2.0" && paramType == "string" {
			if !strings.HasPrefix(value, "'") && !strings.HasSuffix(value, "'") {
				// Escape single quotes by doubling them for OData v2
				escapedValue := strings.ReplaceAll(value, "'", "''")
				value = fmt.Sprintf("'%s'", escapedValue)
			}
		}
	} else {
		// Standard OData string quoting (single-quoted string literals in URLs)
		if isUrlParam && paramType == "string" {
			if !strings.HasPrefix(value, "'") && !strings.HasSuffix(value, "'") {
				escapedValue := strings.ReplaceAll(value, "'", "''")
				value = fmt.Sprintf("'%s'", escapedValue)
			}
		}
	}

	return value
}

func (t Tool) Invoke(ctx context.Context, resourceMgr tools.SourceProvider, params parameters.ParamValues, accessToken tools.AccessToken) (any, util.ToolboxError) {
	source, err := tools.GetCompatibleSource[compatibleSource](resourceMgr, t.Cfg.Source, t.Cfg.Name, resourceType)
	if err != nil {
		return nil, util.NewClientServerError("source used is not compatible with the tool", http.StatusInternalServerError, err)
	}

	paramsMap := params.AsMap()
	metadata := source.Metadata()
	version := "2.0"
	if metadata != nil && metadata.Version != "" {
		version = metadata.Version
	}

	// 1. Build URL
	baseURL := strings.TrimRight(source.HttpBaseURL(), "/")
	reqURL := fmt.Sprintf("%s/%s", baseURL, t.Cfg.EntitySet)

	parsedURL, err := url.Parse(reqURL)
	if err != nil {
		return nil, util.NewClientServerError("invalid url format", http.StatusInternalServerError, err)
	}

	query := parsedURL.Query()

	// 2. Map Query Parameters
	if strings.ToUpper(t.Cfg.Operation) == "READ" {
		query.Set("$format", "json")
		if v, ok := paramsMap["filter"]; ok && v != nil {
			query.Set("$filter", fmt.Sprintf("%v", v))
		}
		if v, ok := paramsMap["select"]; ok && v != nil {
			query.Set("$select", fmt.Sprintf("%v", v))
		}
		if v, ok := paramsMap["top"]; ok && v != nil {
			query.Set("$top", fmt.Sprintf("%v", v))
		}
		if v, ok := paramsMap["skip"]; ok && v != nil {
			query.Set("$skip", fmt.Sprintf("%v", v))
		}
		if v, ok := paramsMap["skiptoken"]; ok && v != nil {
			query.Set("$skiptoken", fmt.Sprintf("%v", v))
		}
	}

	// Map generic explicitly defined query params
	for _, p := range t.Cfg.QueryParams {
		if v, ok := paramsMap[p.GetName()]; ok && v != nil {
			formattedVal := applyODataFormatting(fmt.Sprintf("%v", v), p.GetName(), p.GetType(), version, true, source.Compatibility())
			query.Set(p.GetName(), formattedVal)
		}
	}

	parsedURL.RawQuery = query.Encode()

	// 3. Map Body Parameters for CREATE/UPDATE
	var bodyBytes []byte
	if (strings.ToUpper(t.Cfg.Operation) == "CREATE" || strings.ToUpper(t.Cfg.Operation) == "UPDATE") && len(t.Cfg.BodyParams) > 0 {
		payloadMap := make(map[string]interface{})
		for _, p := range t.Cfg.BodyParams {
			if v, ok := paramsMap[p.GetName()]; ok && v != nil {
				// We don't do complex URL string formatting here, just map directly.
				// Go's JSON parser will serialize nested arrays seamlessly for Deep Inserts.
				payloadMap[p.GetName()] = v
			}
		}
		var marshalErr error
		bodyBytes, marshalErr = json.Marshal(payloadMap)
		if marshalErr != nil {
			return nil, util.NewAgentError("failed to serialize SAP request payload", marshalErr)
		}
	}

	// 4. Execute standard SAP OData execution
	req, err := http.NewRequestWithContext(ctx, t.Method, parsedURL.String(), bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, util.NewClientServerError("error creating http request", http.StatusInternalServerError, err)
	}

	req.Header.Set("Accept", "application/json")
	if len(bodyBytes) > 0 {
		cType := "application/json"
		if t.Cfg.ContentType != "" {
			cType = t.Cfg.ContentType
		}
		req.Header.Set("Content-Type", cType)
	}

	resp, err := source.RunSAPRequest(req, accessToken)
	if err != nil {
		return nil, util.ProcessGeneralError(err)
	}

	// Handle SAP Server Pagination warning
	if respMap, ok := resp.(map[string]interface{}); ok {
		// OData v2
		if d, ok := respMap["d"].(map[string]interface{}); ok {
			if nextLink, hasNext := d["__next"]; hasNext {
				respMap["_NOTICE"] = fmt.Sprintf("Results truncated by SAP server pagination. To get the next set, call again with $skiptoken extracted from this URL: %v", nextLink)
			}
		}
		// OData v4
		if nextLink, hasNext := respMap["@odata.nextLink"]; hasNext {
			respMap["_NOTICE"] = fmt.Sprintf("Results truncated by SAP server pagination. To get the next set, call again with $skiptoken extracted from this URL: %v", nextLink)
		}
	}

	return resp, nil
}

// buildParams builds the tool's parameters from the source's metadata configuration.
func buildParams(operation, entitySet string, bodyParams, queryParams parameters.Parameters, metadata *odata.ODataMetadata) (string, parameters.Parameters) {
	var dynamicParams parameters.Parameters
	var method string

	switch strings.ToUpper(operation) {
	case "READ":
		method = "GET"
		filterDesc := "OData $filter string."
		selectDesc := "OData $select string."

		if metadata != nil {
			if et, err := metadata.GetEntityTypeForSet(entitySet); err == nil {
				var props []string
				for _, p := range et.Properties {
					props = append(props, fmt.Sprintf("%s (%s)", p.Name, p.Type))
				}
				filterDesc = fmt.Sprintf("OData $filter string. Available properties: %s", strings.Join(props, ", "))
				selectDesc = fmt.Sprintf("OData $select string. Available properties: %s", strings.Join(props, ", "))
			}
		}
		filterParam := parameters.NewStringParameter("filter", filterDesc, parameters.WithStringRequired(false))
		selectParam := parameters.NewStringParameter("select", selectDesc, parameters.WithStringRequired(false))
		topParam := parameters.NewIntParameter("top", "OData $top integer limit.", parameters.WithIntRequired(false))
		skipParam := parameters.NewIntParameter("skip", "OData $skip integer offset.", parameters.WithIntRequired(false))
		skiptokenParam := parameters.NewStringParameter("skiptoken", "OData $skiptoken string for server-side pagination.", parameters.WithStringRequired(false))
		dynamicParams = append(dynamicParams, filterParam, selectParam, topParam, skipParam, skiptokenParam)

	case "CREATE":
		method = "POST"
		dynamicParams = append(dynamicParams, bodyParams...)

	case "UPDATE":
		method = "PUT" // Default fallback
		dynamicParams = append(dynamicParams, bodyParams...)

	case "DELETE":
		method = "DELETE"
		// Typically requires passing keys in the URL path, handled via QueryParams currently

	case "FUNCTION_IMPORT":
		method = "POST" // Default for function imports, but should probably read from metadata
	}

	// Always allow explicitly defined QueryParams (e.g., custom params or keys)
	allParameters := append(dynamicParams, queryParams...)
	return method, allParameters
}

// resolveParams builds the tool's parameters using the source's metadata configuration.
func (t Tool) resolveParams(srcs map[string]sources.Source) (parameters.Parameters, error) {
	s, err := tools.GetCompatibleSourceFromMap[compatibleSource](srcs, t.Cfg.Source, t.Cfg.Name, t.Cfg.Type)
	if err != nil {
		return nil, err
	}
	_, params := buildParams(t.Cfg.Operation, t.Cfg.EntitySet, t.Cfg.BodyParams, t.Cfg.QueryParams, s.Metadata())
	return params, nil
}

func (t Tool) GetParameters(srcs map[string]sources.Source) (parameters.Parameters, error) {
	return t.resolveParams(srcs)
}

func (t Tool) Manifest(srcs map[string]sources.Source) (tools.Manifest, error) {
	allParameters, err := t.resolveParams(srcs)
	if err != nil {
		return tools.Manifest{}, err
	}
	return tools.Manifest{
		Description:  t.GetDescription(),
		Parameters:   allParameters.Manifest(),
		AuthRequired: t.GetAuthRequired(),
	}, nil
}

func (t Tool) RequiresClientAuthorization(resourceMgr tools.SourceProvider) (bool, error) {
	source, err := tools.GetCompatibleSource[compatibleSource](resourceMgr, t.Cfg.Source, t.Cfg.Name, t.Cfg.Type)
	if err != nil {
		return false, err
	}
	return source.UseClientAuthorization(), nil
}

func (t Tool) GetAuthTokenHeaderName(resourceMgr tools.SourceProvider) (string, error) {
	source, err := tools.GetCompatibleSource[compatibleSource](resourceMgr, t.Cfg.Source, t.Cfg.Name, t.Cfg.Type)
	if err != nil {
		return "", err
	}

	return source.GetAuthTokenHeaderName(), nil
}
