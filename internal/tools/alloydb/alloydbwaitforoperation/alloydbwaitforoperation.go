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
	"net/http"
	"strings"
	"text/template"
	"time"

	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	alloydbadmin "github.com/googleapis/genai-toolbox/internal/sources/alloydbadmin"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
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
	Name         string   `yaml:"name" validate:"required"`
	Kind         string   `yaml:"kind" validate:"required"`
	Source       string   `yaml:"source" validate:"required"`
	Description  string   `yaml:"description"`
	AuthRequired []string `yaml:"authRequired"`

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
	rawS, ok := srcs[cfg.Source]
	if !ok {
		return nil, fmt.Errorf("no source named %q configured", cfg.Source)
	}

	s, ok := rawS.(*alloydbadmin.Source)
	if !ok {
		return nil, fmt.Errorf("invalid source for %q tool: source kind must be `%s`", kind, alloydbadmin.SourceKind)
	}

	project := s.DefaultProject
	var projectParam parameters.Parameter
	if project != "" {
		projectParam = parameters.NewStringParameterWithDefault("project", project, "The GCP project ID. This is pre-configured; do not ask for it unless the user explicitly provides a different one.")
	} else {
		projectParam = parameters.NewStringParameter("project", "The project ID")
	}

	allParameters := parameters.Parameters{
		projectParam,
		parameters.NewStringParameter("location", "The location ID"),
		parameters.NewStringParameter("operation", "The operation ID"),
	}
	paramManifest := allParameters.Manifest()

	description := cfg.Description
	if description == "" {
		description = "This will poll on operations API until the operation is done. For checking operation status we need projectId, locationID and operationId. Once instance is created give follow up steps on how to use the variables to bring data plane MCP server up in local and remote setup."
	}

	mcpManifest := tools.GetMcpManifest(cfg.Name, description, cfg.AuthRequired, allParameters, nil)

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

	return Tool{
		Config:      cfg,
		Source:      s,
		AllParams:   allParameters,
		manifest:    tools.Manifest{Description: description, Parameters: paramManifest, AuthRequired: cfg.AuthRequired},
		mcpManifest: mcpManifest,
		Delay:       delay,
		MaxDelay:    maxDelay,
		Multiplier:  multiplier,
		MaxRetries:  maxRetries,
	}, nil
}

// Tool represents the wait-for-operation tool.
type Tool struct {
	Config

	Source    *alloydbadmin.Source
	AllParams parameters.Parameters `yaml:"allParams"`

	// Polling configuration
	Delay      time.Duration
	MaxDelay   time.Duration
	Multiplier float64
	MaxRetries int

	Client      *http.Client
	manifest    tools.Manifest
	mcpManifest tools.McpManifest
}

func (t Tool) ToConfig() tools.ToolConfig {
	return t.Config
}

// Invoke executes the tool's logic.
func (t Tool) Invoke(ctx context.Context, params parameters.ParamValues, accessToken tools.AccessToken) (any, error) {
	paramsMap := params.AsMap()

	project, ok := paramsMap["project"].(string)
	if !ok {
		return nil, fmt.Errorf("missing 'project' parameter")
	}
	location, ok := paramsMap["location"].(string)
	if !ok {
		return nil, fmt.Errorf("missing 'location' parameter")
	}
	operation, ok := paramsMap["operation"].(string)
	if !ok {
		return nil, fmt.Errorf("missing 'operation' parameter")
	}

	service, err := t.Source.GetService(ctx, string(accessToken))
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()

	name := fmt.Sprintf("projects/%s/locations/%s/operations/%s", project, location, operation)

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

		op, err := service.Projects.Locations.Operations.Get(name).Do()
		if err != nil {
			fmt.Printf("error getting operation: %s, retrying in %v\n", err, delay)
		} else {
			if op.Done {
				if op.Error != nil {
					var errorBytes []byte
					errorBytes, err = json.Marshal(op.Error)
					if err != nil {
						return nil, fmt.Errorf("operation finished with error but could not marshal error object: %w", err)
					}
					return nil, fmt.Errorf("operation finished with error: %s", string(errorBytes))
				}

				var opBytes []byte
				opBytes, err = op.MarshalJSON()
				if err != nil {
					return nil, fmt.Errorf("could not marshal operation: %w", err)
				}

				if msg, ok := t.generateAlloyDBConnectionMessage(map[string]any{"response": op.Response}); ok {
					return msg, nil
				}

				return string(opBytes), nil
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

func (t Tool) generateAlloyDBConnectionMessage(responseData map[string]any) (string, bool) {
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
func (t Tool) ParseParams(data map[string]any, claims map[string]map[string]any) (parameters.ParamValues, error) {
	return parameters.ParseParams(t.AllParams, data, claims)
}

// Manifest returns the tool's manifest.
func (t Tool) Manifest() tools.Manifest {
	return t.manifest
}

// McpManifest returns the tool's MCP manifest.
func (t Tool) McpManifest() tools.McpManifest {
	return t.mcpManifest
}

// Authorized checks if the tool is authorized.
func (t Tool) Authorized(verifiedAuthServices []string) bool {
	return true
}

func (t Tool) RequiresClientAuthorization() bool {
	return t.Source.UseClientAuthorization()
}

func (t Tool) GetAuthTokenHeaderName() string {
	return "Authorization"
}
