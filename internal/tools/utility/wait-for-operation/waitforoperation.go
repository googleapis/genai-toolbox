// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package waitforoperation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"text/template"
	"time"

	"maps"
	"slices"

	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	httpsrc "github.com/googleapis/genai-toolbox/internal/sources/http"
	"github.com/googleapis/genai-toolbox/internal/tools"
)

const kind string = "wait-for-operation"

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

// Config defines the configuration for the wait-for-operation tool.
type Config struct {
	Name         string            `yaml:"name" validate:"required"`
	Kind         string            `yaml:"kind" validate:"required"`
	Source       string            `yaml:"source" validate:"required"`
	Description  string            `yaml:"description" validate:"required"`
	AuthRequired []string          `yaml:"authRequired"`
	Path         string            `yaml:"path" validate:"required"`
	Method       tools.HTTPMethod  `yaml:"method" validate:"required"`
	Headers      map[string]string `yaml:"headers"`
	PathParams   tools.Parameters  `yaml:"pathParams"`
	HeaderParams tools.Parameters  `yaml:"headerParams"`
}

// validate interface
var _ tools.ToolConfig = Config{}

// ToolConfigKind returns the kind of the tool.
func (cfg Config) ToolConfigKind() string {
	return kind
}

// Initialize initializes the tool from the configuration.
func (cfg Config) Initialize(srcs map[string]sources.Source) (tools.Tool, error) {
	rawS, ok := srcs[cfg.Source]
	if !ok {
		return nil, fmt.Errorf("no source named %q configured", cfg.Source)
	}

	s, ok := rawS.(*httpsrc.Source)
	if !ok {
		return nil, fmt.Errorf("invalid source for %q tool: source kind must be `http`", kind)
	}

	combinedHeaders := make(map[string]string)
	maps.Copy(combinedHeaders, s.DefaultHeaders)
	maps.Copy(combinedHeaders, cfg.Headers)

	allParameters := slices.Concat(cfg.PathParams, cfg.HeaderParams)
	paramManifest := slices.Concat(cfg.PathParams.Manifest(), cfg.HeaderParams.Manifest())
	if paramManifest == nil {
		paramManifest = make([]tools.ParameterManifest, 0)
	}

	pathMcpManifest := cfg.PathParams.McpManifest()
	headerMcpManifest := cfg.HeaderParams.McpManifest()

	concatRequiredManifest := slices.Concat(
		pathMcpManifest.Required,
		headerMcpManifest.Required,
	)
	if concatRequiredManifest == nil {
		concatRequiredManifest = []string{}
	}

	concatPropertiesManifest := make(map[string]tools.ParameterMcpManifest)
	for name, p := range pathMcpManifest.Properties {
		concatPropertiesManifest[name] = p
	}
	for name, p := range headerMcpManifest.Properties {
		concatPropertiesManifest[name] = p
	}

	mcpManifest := tools.McpManifest{
		Name:        cfg.Name,
		Description: cfg.Description,
		InputSchema: tools.McpToolsSchema{
			Type:       "object",
			Properties: concatPropertiesManifest,
			Required:   concatRequiredManifest,
		},
	}

	return &Tool{
		Name:         cfg.Name,
		Kind:         kind,
		BaseURL:      s.BaseURL,
		Path:         cfg.Path,
		Method:       cfg.Method,
		Headers:      combinedHeaders,
		AuthRequired: cfg.AuthRequired,
		PathParams:   cfg.PathParams,
		HeaderParams: cfg.HeaderParams,
		Client:       s.Client,
		AllParams:    allParameters,
		manifest:     tools.Manifest{Description: cfg.Description, Parameters: paramManifest, AuthRequired: cfg.AuthRequired},
		mcpManifest:  mcpManifest,
	}, nil
}

// Tool represents the wait-for-operation tool.
type Tool struct {
	Name         string   `yaml:"name"`
	Kind         string   `yaml:"kind"`
	Description  string   `yaml:"description"`
	AuthRequired []string `yaml:"authRequired"`

	BaseURL      string            `yaml:"baseURL"`
	Path         string            `yaml:"path"`
	Method       tools.HTTPMethod  `yaml:"method"`
	Headers      map[string]string `yaml:"headers"`
	PathParams   tools.Parameters  `yaml:"pathParams"`
	HeaderParams tools.Parameters  `yaml:"headerParams"`
	AllParams    tools.Parameters  `yaml:"allParams"`

	Client      *http.Client
	manifest    tools.Manifest
	mcpManifest tools.McpManifest
}

// Invoke executes the tool's logic.
func (t *Tool) Invoke(ctx context.Context, params tools.ParamValues) (any, error) {
	paramsMap := params.AsMap()

	urlString, err := getURL(t.BaseURL, t.Path, t.PathParams, nil, nil, paramsMap)
	if err != nil {
		return nil, fmt.Errorf("error populating path parameters: %s", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()

	delay := 3 * time.Second    // Initial delay
	maxDelay := 4 * time.Minute // Maximum delay
	multiplier := 2.0           // Exponential backoff multiplier
	maxRetries := 10            // Maximum number of retries
	retries := 0                // Current number of retries

	for retries < maxRetries {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("timed out waiting for operation: %w", ctx.Err())
		default:
		}

		req, _ := http.NewRequest(string(t.Method), urlString, nil)

		allHeaders, err := getHeaders(t.HeaderParams, t.Headers, paramsMap)
		if err != nil {
			return nil, fmt.Errorf("error populating request headers: %s", err)
		}
		for k, v := range allHeaders {
			req.Header.Set(k, v)
		}

		resp, err := t.Client.Do(req)
		if err != nil {
			fmt.Printf("error making HTTP request during polling: %s, retrying in %v\n", err, delay)
		} else {
			body, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				return nil, fmt.Errorf("error reading response body during polling: %w", err)
			}

			if resp.StatusCode != http.StatusOK {
				return nil, fmt.Errorf("unexpected status code during polling: %d, response body: %s", resp.StatusCode, string(body))
			}

			var data map[string]any
			if err := json.Unmarshal(body, &data); err == nil {
				if val, ok := data["done"]; ok {
					if fmt.Sprintf("%v", val) == "true" {
						if _, ok := data["error"]; ok {
							return nil, fmt.Errorf("operation finished with error: %s", string(body))
						}

						if strings.Contains(t.Name, "alloydb") {
							if msg, ok := t.generateAlloyDBConnectionMessage(data); ok {
								return msg, nil
							}
						}

						return string(body), nil
					}
				}
			}
			fmt.Printf("Operation not complete, retrying in %v\n", delay)
		}

		time.Sleep(delay)
		delay = time.Duration(float64(delay) * multiplier)
		if delay > maxDelay {
			delay = maxDelay
		}
		retries++
	}
	return nil, fmt.Errorf("exceeded max retries waiting for operation")
}

func (t *Tool) generateAlloyDBConnectionMessage(opResponse map[string]any) (string, bool) {
	responseData, ok := opResponse["response"].(map[string]any)
	if !ok {
		return "", false
	}

	resourceName, ok := responseData["name"].(string)
	if !ok {
		return "", false
	}

	parts := strings.Split(resourceName, "/")
	var project, region, cluster, instance string

	// Expected format: projects/{project}/locations/{location}/clusters/{cluster}
	// or projects/{project}/locations/{location}/clusters/{cluster}/instances/{instance}
	if len(parts) < 6 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "clusters" {
		return "", false
	}

	project = parts[1]
	region = parts[3]
	cluster = parts[5]

	if len(parts) >= 8 && parts[6] == "instances" {
		instance = parts[7]
	} else {
		return "", false
	}

	var b strings.Builder
	b.WriteString("Your AlloyDB resource is ready.\n\n")
	b.WriteString("To connect, please configure your environment. The method depends on how you are running the toolbox:\n\n")

	b.WriteString("**If running locally via stdio:**\n")
	b.WriteString("Update the MCP server configuration with the following environment variables:\n")
	b.WriteString("```json\n")
	b.WriteString("{\n")
	b.WriteString("  \"mcpServers\": {\n")
	b.WriteString("    \"alloydb\": {\n")
	b.WriteString("      \"command\": \"./PATH/TO/toolbox\",\n")
	b.WriteString("      \"args\": [\"--prebuilt\",\"alloydb-postgres\",\"--stdio\"],\n")
	b.WriteString("      \"env\": {\n")
	b.WriteString(fmt.Sprintf("          \"ALLOYDB_POSTGRES_PROJECT\": \"%s\",\n", project))
	b.WriteString(fmt.Sprintf("          \"ALLOYDB_POSTGRES_REGION\": \"%s\",\n", region))
	b.WriteString(fmt.Sprintf("          \"ALLOYDB_POSTGRES_CLUSTER\": \"%s\",\n", cluster))
	if instance != "" {
		b.WriteString(fmt.Sprintf("          \"ALLOYDB_POSTGRES_INSTANCE\": \"%s\",\n", instance))
	}
	b.WriteString("          \"ALLOYDB_POSTGRES_DATABASE\": \"postgres\",\n")
	b.WriteString("          \"ALLOYDB_POSTGRES_USER\": \"<your-user>\",\n")
	b.WriteString("          \"ALLOYDB_POSTGRES_PASSWORD\": \"<your-password>\"\n")
	b.WriteString("      }\n")
	b.WriteString("    }\n")
	b.WriteString("  }\n")
	b.WriteString("}\n")
	b.WriteString("```\n\n")

	b.WriteString("**If running remotely:**\n")
	b.WriteString("For remote deployments, you will need to set the following environment variables in your deployment configuration. It is recommended to manage these as secrets.\n")
	b.WriteString("```\n")
	b.WriteString(fmt.Sprintf("ALLOYDB_POSTGRES_PROJECT=%s\n", project))
	b.WriteString(fmt.Sprintf("ALLOYDB_POSTGRES_REGION=%s\n", region))
	b.WriteString(fmt.Sprintf("ALLOYDB_POSTGRES_CLUSTER=%s\n", cluster))
	if instance != "" {
		b.WriteString(fmt.Sprintf("ALLOYDB_POSTGRES_INSTANCE=%s\n", instance))
	}
	b.WriteString("ALLOYDB_POSTGRES_DATABASE=postgres\n")
	b.WriteString("ALLOYDB_POSTGRES_USER=<your-user>\n")
	b.WriteString("ALLOYDB_POSTGRES_PASSWORD=<your-password>\n")
	b.WriteString("```\n\n")
	b.WriteString("Please refer to the official documentation for guidance on deploying the toolbox and managing secrets:\n")
	b.WriteString("- Deploying the Toolbox: https://googleapis.github.io/genai-toolbox/how-to/deploy_toolbox/\n")
	b.WriteString("- Deploying on GKE: https://googleapis.github.io/genai-toolbox/how-to/deploy_gke/\n")

	return b.String(), true
}

// ParseParams parses the parameters for the tool.
func (t *Tool) ParseParams(data map[string]any, claims map[string]map[string]any) (tools.ParamValues, error) {
	return tools.ParseParams(t.AllParams, data, claims)
}

// Manifest returns the tool's manifest.
func (t *Tool) Manifest() tools.Manifest {
	return t.manifest
}

// McpManifest returns the tool's MCP manifest.
func (t *Tool) McpManifest() tools.McpManifest {
	return t.mcpManifest
}

// Authorized checks if the tool is authorized.
func (t *Tool) Authorized(verifiedAuthServices []string) bool {
	return tools.IsAuthorized(t.AuthRequired, verifiedAuthServices)
}

func getHeaders(headerParams tools.Parameters, defaultHeaders map[string]string, paramsMap map[string]any) (map[string]string, error) {
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

func getURL(baseURL, path string, pathParams, queryParams tools.Parameters, defaultQueryParams map[string]string, paramsMap map[string]any) (string, error) {
	pathParamValues, err := tools.GetParams(pathParams, paramsMap)
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

	parsedURL, err := url.Parse(baseURL + templatedPath.String())
	if err != nil {
		return "", fmt.Errorf("error parsing URL: %s", err)
	}

	queryParameters := parsedURL.Query()
	for key, value := range defaultQueryParams {
		queryParameters.Add(key, value)
	}
	parsedURL.RawQuery = queryParameters.Encode()

	query := parsedURL.Query()
	for _, p := range queryParams {
		v := paramsMap[p.GetName()]
		if v == nil {
			v = ""
		}
		query.Add(p.GetName(), fmt.Sprintf("%v", v))
	}
	parsedURL.RawQuery = query.Encode()
	return parsedURL.String(), nil
}
