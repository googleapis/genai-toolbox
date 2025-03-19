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
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"maps"

	"github.com/googleapis/genai-toolbox/internal/sources"
	httpsrc "github.com/googleapis/genai-toolbox/internal/sources/http"
	"github.com/googleapis/genai-toolbox/internal/tools"
)

const ToolKind string = "http"

type compatibleSource interface {
	HTTPClient() *http.Client
	GetBaseURL() string
	GetHeaders() map[string]string
	GetQueryParams() map[string]string
}

// validate compatible sources are still compatible
var _ compatibleSource = &httpsrc.Source{}

var compatibleSources = [...]string{httpsrc.SourceKind}

type Config struct {
	Name         string            `yaml:"name" validate:"required"`
	Kind         string            `yaml:"kind" validate:"required"`
	Source       string            `yaml:"source" validate:"required"`
	Description  string            `yaml:"description" validate:"required"`
	AuthRequired []string          `yaml:"authRequired"`
	Path         string            `yaml:"path" validate:"required"`
	Method       HTTPMethod        `yaml:"method" validate:"required"`
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
	s, ok := rawS.(compatibleSource)
	if !ok {
		return nil, fmt.Errorf("invalid source for %q tool: source kind must be one of %q", ToolKind, compatibleSources)
	}

	// Create URL based on BaseURL and Path
	// Attach query parameters
	u, err := url.Parse(s.GetBaseURL() + cfg.Path)
	if err != nil {
		return nil, fmt.Errorf("error parsing URL: %s", err)
	}

	queryParameters := u.Query()
	for key, value := range s.GetQueryParams() {
		queryParameters.Add(key, value)
	}
	u.RawQuery = queryParameters.Encode()

	// Combine Source and Tool headers.
	// In case of conflict, Tool header overrides Source header
	combinedHeaders := make(map[string]string)
	maps.Copy(combinedHeaders, s.GetHeaders())
	maps.Copy(combinedHeaders, cfg.Headers)

	// finish tool setup
	t := NewGenericTool(cfg.Name, u.String(), cfg.RequestBody, cfg.Method, cfg.AuthRequired, cfg.Description, combinedHeaders, s.HTTPClient(), cfg.QueryParams, cfg.BodyParams, cfg.HeaderParams)
	return t, nil
}

func NewGenericTool(name, url, requestBody string, method HTTPMethod, authRequired []string, desc string, headers map[string]string, client *http.Client, queryParams, bodyParams, headerParams tools.Parameters) Tool {
	// create Tool manifest
	paramManifest := append(append(queryParams.Manifest(), bodyParams.Manifest()...), headerParams.Manifest()...)

	return Tool{
		Name:         name,
		Kind:         ToolKind,
		URL:          url,
		Method:       method,
		AuthRequired: authRequired,
		RequestBody:  requestBody,
		QueryParams:  queryParams,
		BodyParams:   bodyParams,
		HeaderParams: headerParams,
		Headers:      headers,
		Client:       client,
		manifest:     tools.Manifest{Description: desc, Parameters: paramManifest},
	}
}

// validate interface
var _ tools.Tool = Tool{}

type Tool struct {
	Name         string   `yaml:"name"`
	Kind         string   `yaml:"kind"`
	Description  string   `yaml:"description"`
	AuthRequired []string `yaml:"authRequired"`

	URL          string            `yaml:"url" validate:"required"`
	Method       HTTPMethod        `yaml:"method" validate:"required"`
	Headers      map[string]string `yaml:"headers"`
	RequestBody  string            `yaml:"requestBody"`
	QueryParams  tools.Parameters  `yaml:"queryParams"`
	BodyParams   tools.Parameters  `yaml:"bodyParams"`
	HeaderParams tools.Parameters  `yaml:"headerParams"`

	Client   *http.Client
	manifest tools.Manifest
}

func (t Tool) Invoke(params tools.ParamValues) ([]any, error) {
	paramsMap := params.AsMap()

	// Populate reqeust body params
	requestBody := t.RequestBody
	for _, p := range t.BodyParams {
		// parameter placeholder symbol is `$`
		subName := "$" + p.GetName()
		if !strings.Contains(requestBody, subName) {
			return nil, fmt.Errorf("request body parameter placeholder %s is not found in the `Tool.requestBody` string", subName)
		}
		v := paramsMap[p.GetName()]
		valueString := tools.ValueString(v, p.GetType())
		requestBody = strings.ReplaceAll(requestBody, subName, valueString)
	}

	// Set Query Parameters
	u, err := url.Parse(t.URL)
	if err != nil {
		return nil, fmt.Errorf("error parsing URL: %s", err)
	}

	query := u.Query()
	for _, p := range t.QueryParams {
		query.Add(p.GetName(), fmt.Sprintf("%s", paramsMap[p.GetName()]))
	}
	u.RawQuery = query.Encode()

	req, _ := http.NewRequest(http.MethodPost, u.String(), strings.NewReader(requestBody))

	// Populate header params
	for _, p := range t.HeaderParams {
		headerValue, ok := paramsMap[p.GetName()]
		if ok {
			if strValue, ok := headerValue.(string); ok {
				t.Headers[p.GetName()] = strValue
			} else {
				return nil, fmt.Errorf("header param %s got value of type %t, not string", p.GetName(), headerValue)
			}
		}
	}

	// Set request headers
	for k, v := range t.Headers {
		req.Header.Set(k, v)
	}

	// Make request and fetch response
	resp, err := t.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making HTTP request: %s", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d, response body: %s", resp.StatusCode, string(body))
	}

	// JSON response could be either an array or an object
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
