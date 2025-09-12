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

package alloydbwaitforoperation

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"text/template"
	"time"

	"maps"

	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	httpsrc "github.com/googleapis/genai-toolbox/internal/sources/http"
	"github.com/googleapis/genai-toolbox/internal/tools"
)

const kind string = "alloydb-wait-for-operation"

var alloyDBConnectionMessageTemplate = `Your AlloyDB resource is ready.

To connect, please configure your environment. The method depends on how you are running the toolbox:

**If running locally via stdio:**
Update the MCP server configuration with the following environment variables:
` + "```json" + `
{
  "mcpServers": {
    "alloydb": {
      "command": "./PATH/TO/toolbox",
      "args": ["--prebuilt","alloydb-postgres","--stdio"],
      "env": {
          "ALLOYDB_POSTGRES_PROJECT": "{{.Project}}",
          "ALLOYDB_POSTGRES_REGION": "{{.Region}}",
          "ALLOYDB_POSTGRES_CLUSTER": "{{.Cluster}}",
{{if .Instance}}          "ALLOYDB_POSTGRES_INSTANCE": "{{.Instance}}",
{{end}}          "ALLOYDB_POSTGRES_DATABASE": "postgres",
          "ALLOYDB_POSTGRES_USER": ""{{.User}}",",
          "ALLOYDB_POSTGRES_PASSWORD": ""{{.Password}}",
      }
    }
  }
}
` + "```" + `

**If running remotely:**
For remote deployments, you will need to set the following environment variables in your deployment configuration:
` + "```" + `
ALLOYDB_POSTGRES_PROJECT={{.Project}}
ALLOYDB_POSTGRES_REGION={{.Region}}
ALLOYDB_POSTGRES_CLUSTER={{.Cluster}}
{{if .Instance}}ALLOYDB_POSTGRES_INSTANCE={{.Instance}}
{{end}}ALLOYDB_POSTGRES_DATABASE=postgres
ALLOYDB_POSTGRES_USER=<your-user>
ALLOYDB_POSTGRES_PASSWORD=<your-password>
` + "```" + `

Please refer to the official documentation for guidance on deploying the toolbox:
- Deploying the Toolbox: https://googleapis.github.io/genai-toolbox/how-to/deploy_toolbox/
- Deploying on GKE: https://googleapis.github.io/genai-toolbox/how-to/deploy_gke/
`

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
	Headers      map[string]string `yaml:"headers"`

	// Polling configuration
	Delay      string  `yaml:"delay"`
	MaxDelay   string  `yaml:"maxDelay"`
	Multiplier float64 `yaml:"multiplier"`
	MaxRetries int     `yaml:"maxRetries"`
}

// validate interface
var _ tools.ToolConfig = Config{}

// ToolConfigKind returns the kind of the tool.
func (cfg Config) ToolConfigKind() string {
	return kind
}

// Initialize initializes the tool from the configuration.
func (cfg Config) Initialize(srcs map[string]sources.Source) (tools.Tool, error) {
	s, ok := srcs[cfg.Source].(*httpsrc.Source)
	if !ok {
		return nil, fmt.Errorf("invalid or missing source for %q tool: source kind must be `http`", kind)
	}

	combinedHeaders := make(map[string]string)
	maps.Copy(combinedHeaders, s.DefaultHeaders)
	maps.Copy(combinedHeaders, cfg.Headers)

	allParameters := tools.Parameters{
		tools.NewStringParameter("project", "The project ID"),
		tools.NewStringParameter("location", "The location ID"),
		tools.NewStringParameter("operation_id", "The operation ID"),
	}
	paramManifest := allParameters.Manifest()

	inputSchema := allParameters.McpManifest()
	inputSchema.Required = []string{"project", "location", "operation_id"}

	mcpManifest := tools.McpManifest{
		Name:        cfg.Name,
		Description: cfg.Description,
		InputSchema: inputSchema,
	}

	var delay time.Duration
	if cfg.Delay == "" {
		delay = 3 * time.Second
	} else {
		var err error
		delay, err = time.ParseDuration(cfg.Delay)
		if err != nil {
			return nil, fmt.Errorf("invalid value for delay: %w", err)
		}
	}

	var maxDelay time.Duration
	if cfg.MaxDelay == "" {
		maxDelay = 4 * time.Minute
	} else {
		var err error
		maxDelay, err = time.ParseDuration(cfg.MaxDelay)
		if err != nil {
			return nil, fmt.Errorf("invalid value for maxDelay: %w", err)
		}
	}

	multiplier := cfg.Multiplier
	if multiplier == 0 {
		multiplier = 2.0
	}

	maxRetries := cfg.MaxRetries
	if maxRetries == 0 {
		maxRetries = 10
	}

	return &Tool{
		Name:         cfg.Name,
		Kind:         kind,
		BaseURL:      s.BaseURL,
		Headers:      combinedHeaders,
		AuthRequired: cfg.AuthRequired,
		Client:       s.Client,
		AllParams:    allParameters,
		manifest:     tools.Manifest{Description: cfg.Description, Parameters: paramManifest, AuthRequired: cfg.AuthRequired},
		mcpManifest:  mcpManifest,
		Delay:        delay,
		MaxDelay:     maxDelay,
		Multiplier:   multiplier,
		MaxRetries:   maxRetries,
	}, nil
}

// Tool represents the wait-for-operation tool.
type Tool struct {
	Name         string   `yaml:"name"`
	Kind         string   `yaml:"kind"`
	Description  string   `yaml:"description"`
	AuthRequired []string `yaml:"authRequired"`

	BaseURL   string            `yaml:"baseURL"`
	Headers   map[string]string `yaml:"headers"`
	AllParams tools.Parameters  `yaml:"allParams"`

	// Polling configuration
	Delay      time.Duration
	MaxDelay   time.Duration
	Multiplier float64
	MaxRetries int

	Client      *http.Client
	manifest    tools.Manifest
	mcpManifest tools.McpManifest
}

// Invoke executes the tool's logic.
func (t *Tool) Invoke(ctx context.Context, params tools.ParamValues) (any, error) {
	paramsMap := params.AsMap()

	project, ok := paramsMap["project"].(string)
	if !ok {
		return nil, fmt.Errorf("missing 'project' parameter")
	}
	location, ok := paramsMap["location"].(string)
	if !ok {
		return nil, fmt.Errorf("missing 'location' parameter")
	}
	operationID, ok := paramsMap["operation_id"].(string)
	if !ok {
		return nil, fmt.Errorf("missing 'operation_id' parameter")
	}

	name := fmt.Sprintf("projects/%s/locations/%s/operations/%s", project, location, operationID)
	urlString := fmt.Sprintf("%s/v1beta/%s", t.BaseURL, name)

	ctx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()

	delay := t.Delay
	maxDelay := t.MaxDelay
	multiplier := t.Multiplier
	maxRetries := t.MaxRetries
	retries := 0

	for retries < maxRetries {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("timed out waiting for operation: %w", ctx.Err())
		default:
		}

		req, _ := http.NewRequest(http.MethodGet, urlString, nil)

		for k, v := range t.Headers {
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

						if msg, ok := t.generateAlloyDBConnectionMessage(data); ok {
							return msg, nil
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

	tmpl, err := template.New("alloydb-connection").Parse(alloyDBConnectionMessageTemplate)
	if err != nil {
		// This should not happen with a static template
		return fmt.Sprintf("template parsing error: %v", err), false
	}

	data := struct {
		Project  string
		Region   string
		Cluster  string
		Instance string
	}{
		Project:  project,
		Region:   region,
		Cluster:  cluster,
		Instance: instance,
	}

	var b strings.Builder
	if err := tmpl.Execute(&b, data); err != nil {
		return fmt.Sprintf("template execution error: %v", err), false
	}

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
