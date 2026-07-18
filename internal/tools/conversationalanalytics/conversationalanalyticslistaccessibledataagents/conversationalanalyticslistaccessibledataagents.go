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

package conversationalanalyticslistaccessibledataagents

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/mcp-toolbox/internal/sources"
	cloudgdads "github.com/googleapis/mcp-toolbox/internal/sources/cloudgda"
	"github.com/googleapis/mcp-toolbox/internal/tools"
	"github.com/googleapis/mcp-toolbox/internal/util"
	"github.com/googleapis/mcp-toolbox/internal/util/parameters"
	"golang.org/x/oauth2"
	"google.golang.org/api/option"
)

const resourceType string = "conversational-analytics-list-accessible-data-agents"

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
	GoogleCloudTokenSourceWithScope(ctx context.Context, scope string) (oauth2.TokenSource, error)
	GetProjectID() string
	UseClientAuthorization() bool
}

// validate compatible sources are still compatible
var _ compatibleSource = &cloudgdads.Source{}

type Config struct {
	tools.ConfigBase `yaml:",inline"`
	Type             string `yaml:"type" validate:"required"`
	Source           string `yaml:"source" validate:"required"`
	Location         string `yaml:"location"`
}

// validate interface
var _ tools.ToolConfig = Config{}

func (cfg Config) ToolConfigType() string {
	return resourceType
}

func (cfg Config) Initialize(context.Context) (tools.Tool, error) {
	if cfg.Description == "" {
		return nil, fmt.Errorf("description is required for tool %q", cfg.Name)
	}

	if cfg.Location == "" {
		cfg.Location = "global"
	}

	params := parameters.Parameters{}

	// finish tool setup
	return Tool{
		BaseTool: tools.NewBaseTool(
			cfg,
			nil,
			tools.Manifest{Description: cfg.Description, Parameters: params.Manifest(), AuthRequired: cfg.AuthRequired},
			params,
		),
	}, nil
}

// validate interface
var _ tools.Tool = Tool{}

type Tool struct {
	tools.BaseTool[Config]
}

func (t Tool) ToConfig() tools.ToolConfig {
	return t.Cfg
}

func (t Tool) validate(srcs map[string]sources.Source) error {
	_, err := tools.GetCompatibleSourceFromMap[compatibleSource](srcs, t.Cfg.Source, t.Cfg.Name, t.Cfg.Type)
	return err
}

func (t Tool) GetParameters(srcs map[string]sources.Source) (parameters.Parameters, error) {
	if err := t.validate(srcs); err != nil {
		return nil, err
	}
	return t.BaseTool.GetParameters(srcs)
}

func (t Tool) Manifest(srcs map[string]sources.Source) (tools.Manifest, error) {
	if err := t.validate(srcs); err != nil {
		return tools.Manifest{}, err
	}
	return t.BaseTool.Manifest(srcs)
}

func (t Tool) Invoke(ctx context.Context, primitiveMgr tools.SourceProvider, params parameters.ParamValues, accessToken tools.AccessToken) (any, util.ToolboxError) {
	source, err := tools.GetCompatibleSource[compatibleSource](primitiveMgr, t.Cfg.Source, t.Cfg.Name, t.Cfg.Type)
	if err != nil {
		return nil, util.NewClientServerError("source used is not compatible with the tool", http.StatusInternalServerError, err)
	}

	var tokenSource oauth2.TokenSource

	// Get credentials for the API call
	if source.UseClientAuthorization() {
		// Use client-side access token
		if accessToken == "" {
			return nil, util.NewClientServerError("tool is configured for client OAuth but no token was provided in the request header", http.StatusUnauthorized, nil)
		}
		tokenStr, err := accessToken.ParseBearerToken()
		if err != nil {
			return nil, util.NewClientServerError("error parsing access token", http.StatusUnauthorized, err)
		}
		tokenSource = oauth2.StaticTokenSource(&oauth2.Token{AccessToken: tokenStr})
	} else {
		// Get a token source for the Gemini Data Analytics API.
		tokenSource, err = source.GoogleCloudTokenSourceWithScope(ctx, "")
		if err != nil {
			return nil, util.NewClientServerError("failed to get token source", http.StatusInternalServerError, err)
		}

		// Use cloud-platform token source for Gemini Data Analytics API
		if tokenSource == nil {
			return nil, util.NewClientServerError("cloud-platform token source is missing", http.StatusInternalServerError, nil)
		}
	}

	// Construct URL
	projectID := source.GetProjectID()
	caURL := fmt.Sprintf("%s/v1/projects/%s/locations/%s/dataAgents:listAccessible", util.GetGDAEndpoint(), projectID, t.Cfg.Location)

	req, err := http.NewRequest("GET", caURL, nil)
	if err != nil {
		return nil, util.NewClientServerError("failed to create request", http.StatusInternalServerError, err)
	}
	req.Header.Set("X-Goog-API-Client", util.GDAClientID)

	client, err := util.NewGDAClient(ctx, option.WithTokenSource(tokenSource))
	if err != nil {
		return nil, util.NewClientServerError("failed to create GDA client", http.StatusInternalServerError, err)
	}
	client.Timeout = 30 * time.Second
	resp, err := client.Do(req)
	if err != nil {
		return nil, util.NewClientServerError("failed to send request", http.StatusInternalServerError, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, util.NewAgentError(fmt.Sprintf("API returned non-200 status: %d %s", resp.StatusCode, string(body)), nil)
	}

	var result any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, util.NewClientServerError("failed to decode response", http.StatusInternalServerError, err)
	}

	return result, nil
}

func (t Tool) RequiresClientAuthorization(primitiveMgr tools.SourceProvider) (bool, error) {
	source, err := tools.GetCompatibleSource[compatibleSource](primitiveMgr, t.Cfg.Source, t.Cfg.Name, t.Cfg.Type)
	if err != nil {
		return false, err
	}
	return source.UseClientAuthorization(), nil
}
