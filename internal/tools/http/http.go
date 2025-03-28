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
// limitations under the License.package http
package http

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"maps"
	"text/template"

	"github.com/googleapis/genai-toolbox/internal/sources"
	httpsrc "github.com/googleapis/genai-toolbox/internal/sources/http"
	"github.com/googleapis/genai-toolbox/internal/tools"
)

const ToolKind string = "http"

type Config struct {
	Name         string            `yaml:"name" validate:"required"`
	Kind         string            `yaml:"kind" validate:"required"`
	Source       string            `yaml:"source" validate:"required"`
	Description  string            `yaml:"description" validate:"required"`
	AuthRequired []string          `yaml:"authRequired"`
	Path         string            `yaml:"path" validate:"required"`
	Method       tools.HTTPMethod  `yaml:"method" validate:"required"`
	Headers      map[string]string `yaml:"headers"`
	RequestBody  string            `yaml:"requestBody"`
	QueryParams  tools.Parameters  `yaml:"queryParams"`
	BodyParams   tools.Parameters  `yaml:"bodyParams"`
	HeaderParams tools.Parameters  `yaml:"headerParams"`
}

// validate interface
var _ tools.ToolConfig = Config{}

func (cfg Config) ToolConfigKind() string {
	return ToolKind
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
		return nil, fmt.Errorf("invalid source for %q tool: source kind must be `http`", ToolKind)
	}

	// Create URL based on BaseURL and Path
	// Attach query parameters
	u, err := url.Parse(s.BaseURL + cfg.Path)
	if err != nil {
		return nil, fmt.Errorf("error parsing URL: %s", err)
	}

	// Get existing query parameters from the URL
	queryParameters := u.Query()
	for key, value := range s.QueryParams {
		queryParameters.Add(key, value)
	}
	u.RawQuery = queryParameters.Encode()

	// Combine Source and Tool headers.
	// In case of conflict, Tool header overrides Source header
	combinedHeaders := make(map[string]string)
	maps.Copy(combinedHeaders, s.DefaultHeaders)
	maps.Copy(combinedHeaders, cfg.Headers)

	// Create parameter manifest
	paramManifest := []tools.ParameterManifest{}
	paramManifest = append(paramManifest, cfg.QueryParams.Manifest()...)
	paramManifest = append(paramManifest, cfg.BodyParams.Manifest()...)
	paramManifest = append(paramManifest, cfg.HeaderParams.Manifest()...)

	// Verify there are no duplicate parameter names
	seenNames := make(map[string]bool)
	for _, param := range paramManifest {
		if _, exists := seenNames[param.Name]; exists {
			return nil, fmt.Errorf("parameter name must be unique across queryParams, bodyParams, and headerParams. Duplicate parameter: %s", param.Name)
		}
		seenNames[param.Name] = true
	}

	// finish tool setup
	return Tool{
		Name:         cfg.Name,
		Kind:         ToolKind,
		URL:          u.String(),
		Method:       cfg.Method,
		AuthRequired: cfg.AuthRequired,
		RequestBody:  cfg.RequestBody,
		QueryParams:  cfg.QueryParams,
		BodyParams:   cfg.BodyParams,
		HeaderParams: cfg.HeaderParams,
		Headers:      combinedHeaders,
		Client:       s.Client,
		manifest:     tools.Manifest{Description: cfg.Description, Parameters: paramManifest},
	}, nil
}

// validate interface
var _ tools.Tool = Tool{}

type Tool struct {
	Name         string   `yaml:"name"`
	Kind         string   `yaml:"kind"`
	Description  string   `yaml:"description"`
	AuthRequired []string `yaml:"authRequired"`

	URL          string            `yaml:"url"`
	Method       tools.HTTPMethod  `yaml:"method"`
	Headers      map[string]string `yaml:"headers"`
	RequestBody  string            `yaml:"requestBody"`
	QueryParams  tools.Parameters  `yaml:"queryParams"`
	BodyParams   tools.Parameters  `yaml:"bodyParams"`
	HeaderParams tools.Parameters  `yaml:"headerParams"`

	Client   *http.Client
	manifest tools.Manifest
}

// helper function to convert an array to JSON formatted string
func convertArrayToJSON(a []any) string {
	res := make([]string, len(a))
	for i, v := range a {
		switch v.(type) {
		case string:
			res[i] = fmt.Sprintf("%q", v)
		default:
			res[i] = fmt.Sprint(v)
		}
	}
	return "[" + strings.Join(res, ",") + "]"
}

func getRequestBody(bodyParams tools.Parameters, requestBodyPayload string, paramsMap map[string]any) (string, error) {
	// Create a map for request body parameters
	bodyParamsMap := make(map[string]any)
	for _, p := range bodyParams {
		k := p.GetName()
		v, ok := paramsMap[k]
		if !ok {
			return "", fmt.Errorf("missing request body parameter %s", k)
		}
		bodyParamsMap[k] = v
	}

	// Create a FuncMap to format array parameters
	funcMap := template.FuncMap{
		"json": convertArrayToJSON,
	}
	templ, err := template.New("body").Funcs(funcMap).Parse(requestBodyPayload)
	if err != nil {
		return "", fmt.Errorf("error parsing request body: %s", err)
	}
	var result bytes.Buffer
	err = templ.Execute(&result, bodyParamsMap)
	if err != nil {
		return "", fmt.Errorf("error replacing body payload: %s", err)
	}
	return result.String(), nil
}

func getURL(defaultURL string, queryParams tools.Parameters, paramsMap map[string]any) (string, error) {
	// Set Query Parameters
	u, err := url.Parse(defaultURL)
	if err != nil {
		return "", fmt.Errorf("error parsing URL: %s", err)
	}

	query := u.Query()
	for _, p := range queryParams {
		query.Add(p.GetName(), fmt.Sprintf("%v", paramsMap[p.GetName()]))
	}
	u.RawQuery = query.Encode()
	return u.String(), nil
}

func getHeaders(headerParams tools.Parameters, defaultHeaders map[string]string, paramsMap map[string]any) (map[string]string, error) {
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

func (t Tool) Invoke(params tools.ParamValues) ([]any, error) {
	paramsMap := params.AsMap()

	// Calculate request body
	requestBody, err := getRequestBody(t.BodyParams, t.RequestBody, paramsMap)
	if err != nil {
		return nil, fmt.Errorf("error populating request body: %s", err)
	}

	// Calculate URL
	urlString, err := getURL(t.URL, t.QueryParams, paramsMap)
	if err != nil {
		return nil, fmt.Errorf("error populating query parameters: %s", err)
	}

	req, _ := http.NewRequest(http.MethodPost, urlString, strings.NewReader(requestBody))

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
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d, response body: %s", resp.StatusCode, string(body))
	}

	return []any{string(body)}, nil
}

func (t Tool) ParseParams(data map[string]any, claims map[string]map[string]any) (tools.ParamValues, error) {
	parameters := []tools.Parameter{}
	parameters = append(parameters, t.BodyParams...)
	parameters = append(parameters, t.HeaderParams...)
	parameters = append(parameters, t.QueryParams...)
	return tools.ParseParams(parameters, data, claims)
}

func (t Tool) Manifest() tools.Manifest {
	return t.manifest
}

func (t Tool) Authorized(verifiedAuthServices []string) bool {
	return tools.IsAuthorized(t.AuthRequired, verifiedAuthServices)
}
