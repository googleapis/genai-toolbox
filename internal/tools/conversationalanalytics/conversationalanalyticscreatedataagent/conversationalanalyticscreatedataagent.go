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

package conversationalanalyticscreatedataagent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	yaml "github.com/goccy/go-yaml"
	cloudgdads "github.com/googleapis/mcp-toolbox/internal/sources/cloudgda"
	"github.com/googleapis/mcp-toolbox/internal/tools"
	"github.com/googleapis/mcp-toolbox/internal/util"
	"github.com/googleapis/mcp-toolbox/internal/util/parameters"
	"golang.org/x/oauth2"
	"google.golang.org/api/option"
)

const resourceType string = "conversational-analytics-create-data-agent"

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

	dataAgentIdParameter := parameters.NewStringParameter("data_agent_id", "The ID to use for the new data agent.")
	payloadParameter := parameters.NewMapParameter("payload", "The JSON representation of the DataAgent resource to create.", "")
	params := parameters.Parameters{dataAgentIdParameter, payloadParameter}

	// finish tool setup
	return Tool{
		BaseTool: tools.NewBaseTool(
			cfg,
			tools.NewWriteAnnotations(),
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

	// Extract parameters from the map
	mapParams := params.AsMap()
	dataAgentId, _ := mapParams["data_agent_id"].(string)
	payloadMap, _ := mapParams["payload"].(map[string]any)

	payloadBytes, err := json.Marshal(payloadMap)
	if err != nil {
		return nil, util.NewClientServerError("invalid payload", http.StatusBadRequest, err)
	}

	// Construct URL
	projectID := source.GetProjectID()
	caURL := fmt.Sprintf("%s/v1/projects/%s/locations/%s/dataAgents?dataAgentId=%s", util.GetGDAEndpoint(), projectID, t.Cfg.Location, url.QueryEscape(dataAgentId))

	req, err := http.NewRequestWithContext(ctx, "POST", caURL, bytes.NewReader(payloadBytes))
	if err != nil {
		return nil, util.NewClientServerError("failed to create request", http.StatusInternalServerError, err)
	}
	req.Header.Set("X-Goog-API-Client", util.GDAClientID)
	req.Header.Set("Content-Type", "application/json")

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

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, util.NewAgentError(fmt.Sprintf("API returned error status: %d %s", resp.StatusCode, string(body)), nil)
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, util.NewClientServerError("failed to decode response", http.StatusInternalServerError, err)
	}

	opName, ok := result["name"].(string)
	if !ok {
		return nil, util.NewClientServerError("operation response missing name", http.StatusInternalServerError, nil)
	}

	if val, done, err := extractResult(result); done {
		if err != nil {
			return nil, err.(util.ToolboxError)
		}
		return val, nil
	}

	const pollInterval = 2 * time.Second
	const pollTimeout = 60 * time.Second

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	timeout := time.After(pollTimeout)

	var lastStatus string

	for {
		select {
		case <-ctx.Done():
			errMsg := fmt.Sprintf("context cancelled while waiting for data agent creation (op: %s): %v", opName, ctx.Err())
			if lastStatus != "" {
				errMsg += fmt.Sprintf(". Last status: %s", lastStatus)
			}
			return nil, util.NewClientServerError(errMsg, http.StatusInternalServerError, nil)
		case <-timeout:
			errMsg := fmt.Sprintf("timed out waiting for data agent creation (op: %s)", opName)
			if lastStatus != "" {
				errMsg += fmt.Sprintf(". Last status: %s", lastStatus)
			}
			return nil, util.NewClientServerError(errMsg, http.StatusGatewayTimeout, nil)
		case <-ticker.C:
			opUrl := fmt.Sprintf("%s/v1/%s", util.GetGDAEndpoint(), opName)
			opReq, err := http.NewRequestWithContext(ctx, http.MethodGet, opUrl, nil)
			if err != nil {
				lastStatus = fmt.Sprintf("request creation error: %v", err)
				continue
			}
			opReq.Header.Set("X-Goog-API-Client", util.GDAClientID)

			opResp, err := client.Do(opReq)
			if err != nil {
				lastStatus = fmt.Sprintf("network error: %v", err)
				continue
			}

			if opResp.StatusCode == 400 || opResp.StatusCode == 401 || opResp.StatusCode == 403 {
				body, _ := io.ReadAll(opResp.Body)
				opResp.Body.Close()
				return nil, util.NewClientServerError(fmt.Sprintf("polling failed with %d: %s", opResp.StatusCode, string(body)), opResp.StatusCode, nil)
			}

			if opResp.StatusCode != 200 {
				lastStatus = fmt.Sprintf("HTTP status %d", opResp.StatusCode)
				opResp.Body.Close()
				continue
			}

			opRespBody, _ := io.ReadAll(opResp.Body)
			opResp.Body.Close()

			var pollOp map[string]any
			if err := json.Unmarshal(opRespBody, &pollOp); err != nil {
				lastStatus = fmt.Sprintf("unmarshal error: %v", err)
				continue
			}

			if val, done, err := extractResult(pollOp); done {
				if err != nil {
					return nil, err.(util.ToolboxError)
				}
				return val, nil
			}
		}
	}
}

func (t Tool) RequiresClientAuthorization(primitiveMgr tools.SourceProvider) (bool, error) {
	source, err := tools.GetCompatibleSource[compatibleSource](primitiveMgr, t.Cfg.Source, t.Cfg.Name, t.Cfg.Type)
	if err != nil {
		return false, err
	}
	return source.UseClientAuthorization(), nil
}

func extractResult(result map[string]any) (any, bool, error) {
	if d, ok := result["done"].(bool); ok && d {
		if errVal, ok := result["error"]; ok && errVal != nil {
			return nil, true, util.NewClientServerError(fmt.Sprintf("data agent creation failed: %v", errVal), http.StatusInternalServerError, nil)
		}
		if responseVal, ok := result["response"].(map[string]any); ok {
			return responseVal, true, nil
		}
		return result, true, nil
	}
	return nil, false, nil
}
