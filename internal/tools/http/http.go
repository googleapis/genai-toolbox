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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"maps"
	"text/template"

	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	httpsrc "github.com/googleapis/genai-toolbox/internal/sources/http"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
)

const kind string = "http"

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

type Config struct {
	Name         string                `yaml:"name" validate:"required"`
	Kind         string                `yaml:"kind" validate:"required"`
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
	s, ok := rawS.(*httpsrc.Source)
	if !ok {
		return nil, fmt.Errorf("invalid source for %q tool: source kind must be `http`", kind)
	}

	// Combine Source and Tool headers.
	// In case of conflict, Tool header overrides Source header
	combinedHeaders := make(map[string]string)
	maps.Copy(combinedHeaders, s.DefaultHeaders)
	maps.Copy(combinedHeaders, cfg.Headers)

	// Create a slice for all parameters
	allParameters := slices.Concat(cfg.PathParams, cfg.BodyParams, cfg.HeaderParams, cfg.QueryParams)

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
		Config:             cfg,
		BaseURL:            s.BaseURL,
		Headers:            combinedHeaders,
		DefaultQueryParams: s.QueryParams,
		Client:             s.Client,
		AllParams:          allParameters,
		manifest:           tools.Manifest{Description: cfg.Description, Parameters: paramManifest, AuthRequired: cfg.AuthRequired},
		mcpManifest:        mcpManifest,
	}, nil
}

// validate interface
var _ tools.Tool = Tool{}

type Tool struct {
	Config
	BaseURL            string                `yaml:"baseURL"`
	Headers            map[string]string     `yaml:"headers"`
	DefaultQueryParams map[string]string     `yaml:"defaultQueryParams"`
	AllParams          parameters.Parameters `yaml:"allParams"`

	Client      *http.Client
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

func (t Tool) Invoke(ctx context.Context, params parameters.ParamValues, accessToken tools.AccessToken) (any, error) {
	paramsMap := params.AsMap()

	// Calculate request body
	requestBody, err := getRequestBody(t.BodyParams, t.RequestBody, paramsMap)
	if err != nil {
		return nil, fmt.Errorf("error populating request body: %s", err)
	}

	// Calculate URL
	urlString, err := getURL(t.BaseURL, t.Path, t.PathParams, t.QueryParams, t.DefaultQueryParams, paramsMap)
	if err != nil {
		return nil, fmt.Errorf("error populating path parameters: %s", err)
	}

	req, _ := http.NewRequest(string(t.Method), urlString, strings.NewReader(requestBody))

	// Calculate request headers
	allHeaders, err := getHeaders(t.HeaderParams, t.Headers, paramsMap)
	if err != nil {
		return nil, fmt.Errorf("error populating request headers: %s", err)
	}
	// Set request headers
	for k, v := range allHeaders {
		req.Header.Set(k, v)
	}

	// Make request and fetch response
	resp, err := t.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making HTTP request: %s", err)
	}
	defer resp.Body.Close()

	var body []byte
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("unexpected status code: %d, response body: %s", resp.StatusCode, string(body))
	}

	var data any
	if err = json.Unmarshal(body, &data); err != nil {
		// if unable to unmarshal data, return result as string.
		return string(body), nil
	}
	return data, nil
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

func (t Tool) GetAuthTokenHeaderName() string {
	return "Authorization"
}
