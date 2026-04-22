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

package sapodataaction

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/googleapis/mcp-toolbox/internal/embeddingmodels"
	"github.com/googleapis/mcp-toolbox/internal/sources"
	"github.com/googleapis/mcp-toolbox/internal/sources/sapodata"
	"github.com/googleapis/mcp-toolbox/internal/tools"
	"github.com/googleapis/mcp-toolbox/internal/util"
	"github.com/googleapis/mcp-toolbox/internal/util/parameters"
)

const resourceType string = "sap-odata"

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

type sapSource interface {
	HttpBaseURL() string
	RunSAPRequest(*http.Request, tools.AccessToken) (any, error)
	Metadata() *sapodata.ODataMetadata
}

type sapSourceOauth interface {
	sapSource
	IsClientOauthEnabled() bool
	GetAuthTokenHeaderName() string
}

type Config struct {
	Name         string                `yaml:"name" validate:"required"`
	Type         string                `yaml:"type" validate:"required"`
	Source       string                `yaml:"source" validate:"required"`
	Description  string                `yaml:"description" validate:"required"`
	AuthRequired []string              `yaml:"authRequired"`
	EntitySet    string                `yaml:"entitySet" validate:"required"`
	Operation    string                `yaml:"operation" validate:"required"` // READ, CREATE, UPDATE, DELETE, FUNCTION_IMPORT
	ContentType  string                `yaml:"contentType"`                   // Override default application/json
	QueryParams  parameters.Parameters `yaml:"queryParams"`
	BodyParams   parameters.Parameters `yaml:"bodyParams"`
}

var _ tools.ToolConfig = Config{}

func (cfg Config) ToolConfigType() string {
	return resourceType
}

func (cfg Config) Initialize(srcs map[string]sources.Source) (tools.Tool, error) {
	rawS, ok := srcs[cfg.Source]
	if !ok {
		return nil, fmt.Errorf("no source named %q configured", cfg.Source)
	}

	s, ok := rawS.(sapSource)
	if !ok {
		return nil, fmt.Errorf("invalid source for %q tool: source type must be sap-odata", resourceType)
	}

	metadata := s.Metadata()

	var dynamicParams parameters.Parameters
	var method string

	switch strings.ToUpper(cfg.Operation) {
	case "READ":
		method = "GET"
		// Dynmaically build OData parameters for the EntitySet using parsed metadata
		filterDesc := "OData $filter string."
		selectDesc := "OData $select string."

		if metadata != nil {
			if et, err := metadata.GetEntityTypeForSet(cfg.EntitySet); err == nil {
				var props []string
				for _, p := range et.Properties {
					props = append(props, fmt.Sprintf("%s (%s)", p.Name, p.Type))
				}
				filterDesc = fmt.Sprintf("OData $filter string. Available properties: %s", strings.Join(props, ", "))
				selectDesc = fmt.Sprintf("OData $select string. Available properties: %s", strings.Join(props, ", "))
			}
		}

		filterParam := parameters.NewStringParameterWithRequired("filter", filterDesc, false)
		selectParam := parameters.NewStringParameterWithRequired("select", selectDesc, false)
		topParam := parameters.NewIntParameterWithRequired("top", "OData $top integer limit.", false)
		skipParam := parameters.NewIntParameterWithRequired("skip", "OData $skip integer offset.", false)
		skiptokenParam := parameters.NewStringParameterWithRequired("skiptoken", "OData $skiptoken string for server-side pagination.", false)

		dynamicParams = append(dynamicParams, filterParam, selectParam, topParam, skipParam, skiptokenParam)

	case "CREATE":
		method = "POST"
		dynamicParams = append(dynamicParams, cfg.BodyParams...)

	case "UPDATE":
		method = "PUT" // Default fallback
		if metadata != nil {
			if metadata.Version == "4.0" {
				method = "PATCH"
			} else if metadata.Version == "2.0" {
				method = "MERGE"
			}
		}
		dynamicParams = append(dynamicParams, cfg.BodyParams...)

	case "DELETE":
		method = "DELETE"
		// Typically requires passing keys in the URL path, handled via QueryParams currently

	case "FUNCTION_IMPORT":
		method = "POST" // Default for function imports, but should probably read from metadata
		if metadata != nil {
			if fi, ok := metadata.FunctionImps[cfg.EntitySet]; ok {
				if fi.HttpMethod != "" {
					method = fi.HttpMethod
				}
			}
		}
		// Uses QueryParams mostly for args
		dynamicParams = append(dynamicParams, cfg.QueryParams...)
	}

	// Always allow explicitly defined QueryParams (e.g., custom params or keys)
	allParameters := append(parameters.Parameters(nil), dynamicParams...)
	if strings.ToUpper(cfg.Operation) != "FUNCTION_IMPORT" {
		allParameters = append(allParameters, cfg.QueryParams...)
	}

	// Remove duplicates
	err := parameters.CheckDuplicateParameters(allParameters)
	if err != nil {
		return nil, err
	}

	paramManifest := allParameters.Manifest()
	if paramManifest == nil {
		paramManifest = make([]parameters.ParameterManifest, 0)
	}

	mcpManifest := tools.GetMcpManifest(cfg.Name, cfg.Description, cfg.AuthRequired, allParameters, nil)

	return Tool{
		Config:      cfg,
		Method:      method,
		AllParams:   allParameters,
		manifest:    tools.Manifest{Description: cfg.Description, Parameters: paramManifest, AuthRequired: cfg.AuthRequired},
		mcpManifest: mcpManifest,
	}, nil
}

var _ tools.Tool = Tool{}

type Tool struct {
	Config
	Method      string
	AllParams   parameters.Parameters
	manifest    tools.Manifest
	mcpManifest tools.McpManifest
}

func (t Tool) ToConfig() tools.ToolConfig {
	return t.Config
}

// applySAPFormatting automatically applies v2/v4 syntax transformations to values based on SAP heuristics
func applySAPFormatting(value string, paramName string, paramType string, mdVersion string, isUrlParam bool) string {
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

	// OData v2 vs v4 formatting logic can be expanded here based on EDm Type heuristics,
	// but mostly depends on how the LLM maps those strings.
	// OData v2 requires string literals in the URL to be single quoted.
	if isUrlParam && mdVersion == "2.0" && paramType == "string" {
		if !strings.HasPrefix(value, "'") && !strings.HasSuffix(value, "'") {
			// Escape single quotes by doubling them for OData v2
			escapedValue := strings.ReplaceAll(value, "'", "''")
			value = fmt.Sprintf("'%s'", escapedValue)
		}
	}

	return value
}

func (t Tool) Invoke(ctx context.Context, resourceMgr tools.SourceProvider, params parameters.ParamValues, accessToken tools.AccessToken) (any, util.ToolboxError) {
	source, err := tools.GetCompatibleSource[sapSource](resourceMgr, t.Source, t.Name, resourceType)
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
	var reqURL string

	if strings.ToUpper(t.Operation) == "FUNCTION_IMPORT" {
		reqURL = fmt.Sprintf("%s/%s", baseURL, t.EntitySet)
	} else {
		reqURL = fmt.Sprintf("%s/%s", baseURL, t.EntitySet)
	}

	parsedURL, err := url.Parse(reqURL)
	if err != nil {
		return nil, util.NewClientServerError("invalid url format", http.StatusInternalServerError, err)
	}

	query := parsedURL.Query()

	// 2. Map Query Parameters
	if strings.ToUpper(t.Operation) == "READ" {
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
	} else if strings.ToUpper(t.Operation) == "FUNCTION_IMPORT" {
		for _, p := range t.QueryParams {
			if v, ok := paramsMap[p.GetName()]; ok && v != nil {
				formattedVal := applySAPFormatting(fmt.Sprintf("%v", v), p.GetName(), p.GetType(), version, true)
				query.Set(p.GetName(), formattedVal)
			}
		}
	}

	// Map generic explicitly defined query params
	for _, p := range t.QueryParams {
		if v, ok := paramsMap[p.GetName()]; ok && v != nil {
			formattedVal := applySAPFormatting(fmt.Sprintf("%v", v), p.GetName(), p.GetType(), version, true)
			query.Set(p.GetName(), formattedVal)
		}
	}

	parsedURL.RawQuery = query.Encode()

	// 3. Map Body Parameters for CREATE/UPDATE
	var bodyBytes []byte
	if (strings.ToUpper(t.Operation) == "CREATE" || strings.ToUpper(t.Operation) == "UPDATE") && len(t.BodyParams) > 0 {
		payloadMap := make(map[string]interface{})
		for _, p := range t.BodyParams {
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
		if t.ContentType != "" {
			cType = t.ContentType
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

func (t Tool) EmbedParams(ctx context.Context, paramValues parameters.ParamValues, embeddingModelsMap map[string]embeddingmodels.EmbeddingModel) (parameters.ParamValues, error) {
	return paramValues, nil
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

func (t Tool) RequiresClientAuthorization(resourceMgr tools.SourceProvider) (bool, error) {
	s, ok := resourceMgr.GetSource(t.Config.Source)
	if !ok {
		return false, fmt.Errorf("unable to retrieve source %q", t.Config.Source)
	}

	if oauthSource, ok := s.(sapSourceOauth); ok {
		return oauthSource.IsClientOauthEnabled(), nil
	}
	return false, nil
}

func (t Tool) GetAuthTokenHeaderName(resourceMgr tools.SourceProvider) (string, error) {
	s, ok := resourceMgr.GetSource(t.Config.Source)
	if !ok {
		return "Authorization", fmt.Errorf("unable to retrieve source %q", t.Config.Source)
	}

	if oauthSource, ok := s.(sapSourceOauth); ok {
		if oauthSource.IsClientOauthEnabled() {
			return oauthSource.GetAuthTokenHeaderName(), nil
		}
	}
	return "Authorization", nil
}

func (t Tool) GetParameters() parameters.Parameters {
	return t.AllParams
}
