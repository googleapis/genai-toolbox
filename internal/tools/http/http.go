// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package http

import (
	"bytes"
	"context"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"text/template"

	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/embeddingmodels"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/util"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
)

const resourceType string = "http"

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
	HttpDefaultHeaders() map[string]string
	HttpBaseURL() string
	HttpQueryParams() map[string]string
	RunRequest(*http.Request) (any, error)
}

type Config struct {
	Name         string                `yaml:"name" validate:"required"`
	Type         string                `yaml:"type" validate:"required"`
	Source       string                `yaml:"source" validate:"required"`
	Description  string                `yaml:"description" validate:"required"`
	AuthRequired []string              `yaml:"authRequired"`
	Path         string                `yaml:"path" validate:"required"`
	Method       tools.HTTPMethod      `yaml:"method" validate:"required"`
	Headers      map[string]string     `yaml:"headers"`
	RequestBody  string                `yaml:"requestBody"`
	PathParams   parameters.Parameters `yaml:"pathParams"`
	QueryParams  parameters.Parameters `yaml:"queryParams"`
	BodyParams   parameters.Parameters `yaml:"bodyParams"`
	HeaderParams parameters.Parameters `yaml:"headerParams"`
}

// validate interface
var _ tools.ToolConfig = Config{}

func (cfg Config) ToolConfigType() string {
	return resourceType
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
		return nil, fmt.Errorf("invalid source for %q tool: source type must be `http`", resourceType)
	}

	// Combine Source and Tool headers.
	// In case of conflict, Tool header overrides Source header
	combinedHeaders := make(map[string]string)
	maps.Copy(combinedHeaders, s.HttpDefaultHeaders())
	maps.Copy(combinedHeaders, cfg.Headers)

	// Create a slice for all parameters
	allParameters := slices.Concat(cfg.PathParams, cfg.QueryParams, cfg.BodyParams, cfg.HeaderParams)

	// Verify no duplicate parameter names
	err := parameters.CheckDuplicateParameters(allParameters)
	if err != nil {
		return nil, err
	}

	// Create Toolbox manifest
	paramManifest := allParameters.Manifest()

	if paramManifest == nil {
		paramManifest = make([]parameters.ParameterManifest, 0)
	}

	// Create MCP manifest
	mcpManifest := tools.GetMcpManifest(cfg.Name, cfg.Description, cfg.AuthRequired, allParameters, nil)

	// finish tool setup
	return Tool{
		Config:      cfg,
		Headers:     combinedHeaders,
		AllParams:   allParameters,
		manifest:    tools.Manifest{Description: cfg.Description, Parameters: paramManifest, AuthRequired: cfg.AuthRequired},
		mcpManifest: mcpManifest,
	}, nil
}

// validate interface
var _ tools.Tool = Tool{}

type Tool struct {
	Config
	Headers     map[string]string     `yaml:"headers"`
	AllParams   parameters.Parameters `yaml:"allParams"`
	manifest    tools.Manifest
	mcpManifest tools.McpManifest
}

func (t Tool) ToConfig() tools.ToolConfig {
	return t.Config
}

// Helper function to generate the HTTP request body upon Tool invocation.
func getRequestBody(bodyParams parameters.Parameters, requestBodyPayload string, paramsMap map[string]any) (string, error) {
	bodyParamValues, err := parameters.GetParams(bodyParams, paramsMap)
	if err != nil {
		return "", err
	}
	bodyParamsMap := bodyParamValues.AsMap()

	requestBodyStr, err := parameters.PopulateTemplateWithJSON("HTTPToolRequestBody", requestBodyPayload, bodyParamsMap)
	if err != nil {
		return "", err
	}
	return requestBodyStr, nil
}

// Helper function to generate the HTTP request URL upon Tool invocation.
func getURL(baseURL, path string, pathParams, queryParams parameters.Parameters, defaultQueryParams map[string]string, paramsMap map[string]any) (string, error) {
	// use Go template to replace path params
	pathParamValues, err := parameters.GetParams(pathParams, paramsMap)
	if err != nil {
		return "", err
	}
	pathParamsMap := pathParamValues.AsMap()

	templ, err := template.New("url").Parse(path)
	if err != nil {
		return "", fmt.Errorf("error parsing URL: %s", err)
	}
	var templatedPath bytes.Buffer
	err = templ.Execute(&templatedPath, pathParamsMap)
	if err != nil {
		return "", fmt.Errorf("error replacing pathParams: %s", err)
	}

	// Create URL based on BaseURL and Path
	// Attach query parameters
	parsedURL, err := url.Parse(baseURL + templatedPath.String())
	if err != nil {
		return "", fmt.Errorf("error parsing URL: %s", err)
	}

	// Get existing query parameters from the URL
	queryParameters := parsedURL.Query()
	for key, value := range defaultQueryParams {
		queryParameters.Add(key, value)
	}
	parsedURL.RawQuery = queryParameters.Encode()

	// Set dynamic query parameters
	query := parsedURL.Query()
	for _, p := range queryParams {
		v, ok := paramsMap[p.GetName()]
		if !ok || v == nil {
			if !p.GetRequired() {
				// If the param is not required AND
				// Not provodid OR provided with a nil value
				// Omitted from the URL
				continue
			}
			v = ""
		}
		query.Add(p.GetName(), fmt.Sprintf("%v", v))
	}
	parsedURL.RawQuery = query.Encode()
	return parsedURL.String(), nil
}

// Helper function to generate the HTTP headers upon Tool invocation.
func getHeaders(headerParams parameters.Parameters, defaultHeaders map[string]string, paramsMap map[string]any) (map[string]string, error) {
	// Populate header params
	allHeaders := make(map[string]string)
	maps.Copy(allHeaders, defaultHeaders)
	for _, p := range headerParams {
		headerValue, ok := paramsMap[p.GetName()]
		if ok {
			if strValue, ok := headerValue.(string); ok {
				allHeaders[p.GetName()] = strValue
			} else {
				return nil, fmt.Errorf("header param %s got value of type %t, not string", p.GetName(), headerValue)
			}
		}
	}
	return allHeaders, nil
}

func (t Tool) Invoke(ctx context.Context, resourceMgr tools.SourceProvider, params parameters.ParamValues, accessToken tools.AccessToken) (any, util.ToolboxError) {
	source, err := tools.GetCompatibleSource[compatibleSource](resourceMgr, t.Source, t.Name, t.Type)
	if err != nil {
		return nil, util.NewClientServerError("source used is not compatible with the tool", http.StatusInternalServerError, err)
	}

	paramsMap := params.AsMap()

	// Calculate request body
	requestBody, err := getRequestBody(t.BodyParams, t.RequestBody, paramsMap)
	if err != nil {
		return nil, util.NewAgentError("error populating request body", err)
	}

	// Calculate URL
	urlString, err := getURL(source.HttpBaseURL(), t.Path, t.PathParams, t.QueryParams, source.HttpQueryParams(), paramsMap)
	if err != nil {
		return nil, util.NewAgentError("error populating path parameters", err)
	}

	req, err := http.NewRequestWithContext(ctx, string(t.Method), urlString, strings.NewReader(requestBody))
	if err != nil {
		return nil, util.NewClientServerError("error creating http request", http.StatusInternalServerError, err)
	}

	// Calculate request headers
	allHeaders, err := getHeaders(t.HeaderParams, t.Headers, paramsMap)
	if err != nil {
		return nil, util.NewAgentError("error populating request headers", err)
	}
	// Set request headers
	for k, v := range allHeaders {
		req.Header.Set(k, v)
	}

	resp, err := source.RunRequest(req)
	if err != nil {
		return nil, util.ProcessGeneralError(err)
	}
	return resp, nil
}

func (t Tool) EmbedParams(ctx context.Context, paramValues parameters.ParamValues, embeddingModelsMap map[string]embeddingmodels.EmbeddingModel) (parameters.ParamValues, error) {
	return parameters.EmbedParams(ctx, t.AllParams, paramValues, embeddingModelsMap, nil)
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
	return false, nil
}

func (t Tool) GetAuthTokenHeaderName(resourceMgr tools.SourceProvider) (string, error) {
	return "Authorization", nil
}

func (t Tool) GetParameters() parameters.Parameters {
	return t.AllParams
}
