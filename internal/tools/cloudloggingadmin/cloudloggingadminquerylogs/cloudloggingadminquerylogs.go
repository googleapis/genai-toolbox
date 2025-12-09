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
package cloudloggingadminquerylogs

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/logging/logadmin"
	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	cla "github.com/googleapis/genai-toolbox/internal/sources/cloudloggingadmin"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
	"google.golang.org/api/iterator"
)

const (
	kind string = "cloud-logging-admin-query-logs"

	defaultLimit               int = 200
	defaultStartTimeOffsetDays int = 30
)

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

type compatibleSource interface {
	LogAdminClient() *logadmin.Client
	LogAdminClientCreator() cla.LogAdminClientCreator
	UseClientAuthorization() bool
}

var _ compatibleSource = &cla.Source{}

var compatibleSources = [...]string{cla.SourceKind}

type Config struct {
	Name         string   `yaml:"name" validate:"required"`
	Kind         string   `yaml:"kind" validate:"required"`
	Source       string   `yaml:"source" validate:"required"`
	Description  string   `yaml:"description" validate:"required"`
	AuthRequired []string `yaml:"authRequired"`
}

// validate interface
var _ tools.ToolConfig = Config{}

func (cfg Config) ToolConfigKind() string {
	return kind
}

func (cfg Config) Initialize(srcs map[string]sources.Source) (tools.Tool, error) {
	rawS, ok := srcs[cfg.Source]
	if !ok {
		return nil, fmt.Errorf("no source named %q configured", cfg.Source)
	}

	// verify the source is compatible
	s, ok := rawS.(compatibleSource)
	if !ok {
		return nil, fmt.Errorf("invalid source for %q tool: source kind must be one of %q", kind, compatibleSources)
	}

	startTimeDescription := fmt.Sprintf("Start time in RFC3339 format (e.g., 2025-12-09T00:00:00Z). Defaults to %d days ago.", defaultStartTimeOffsetDays)
	limitDescription := fmt.Sprintf("Maximum number of log entries to return (default: %d).", defaultLimit)
	params := parameters.Parameters{
		parameters.NewStringParameterWithRequired(
			"filter",
			"Cloud Logging filter query. Common fields: resource.type, resource.labels.*, logName, severity, textPayload, jsonPayload.*, protoPayload.*, labels.*, httpRequest.*. Operators: =, !=, <, <=, >, >=, :, =~, AND, OR, NOT.",
			false,
		),
		parameters.NewBooleanParameterWithRequired("newestFirst", "Set to true for newest logs first. Defaults to oldest first.", false),
		parameters.NewStringParameterWithRequired("startTime", startTimeDescription, false),
		parameters.NewStringParameterWithRequired("endTime", "End time in RFC3339 format (e.g., 2025-12-09T23:59:59Z). Defaults to now.", false),
		parameters.NewBooleanParameterWithRequired("verbose", "Include additional fields (insertId, trace, spanId, httpRequest, labels, operation, sourceLocation). Defaults to false.", false),
		parameters.NewIntParameterWithRequired("limit", limitDescription, false),
	}

	mcpManifest := tools.GetMcpManifest(cfg.Name, cfg.Description, cfg.AuthRequired, params)

	t := Tool{
		Config:      cfg,
		source:      s,
		manifest:    tools.Manifest{Description: cfg.Description, Parameters: params.Manifest(), AuthRequired: cfg.AuthRequired},
		mcpManifest: mcpManifest,
		params:      params,
	}
	return t, nil
}

// validate interface
var _ tools.Tool = Tool{}

type Tool struct {
	Config

	source      compatibleSource
	manifest    tools.Manifest
	mcpManifest tools.McpManifest
	params      parameters.Parameters
}

func (t Tool) Invoke(ctx context.Context, params parameters.ParamValues, accessToken tools.AccessToken) (any, error) {
	var client *logadmin.Client

	if t.source.UseClientAuthorization() {
		tokenString, err := accessToken.ParseBearerToken()
		if err != nil {
			return nil, fmt.Errorf("failed to parse access token: %w", err)
		}
		client, err = t.source.LogAdminClientCreator()(tokenString)
		if err != nil {
			return nil, fmt.Errorf("failed to create client: %w", err)
		}
	} else {
		client = t.source.LogAdminClient()
		if client == nil {
			return nil, fmt.Errorf("source client is not initialized")
		}
	}

	// build filter and opts.
	limit := defaultLimit
	paramsMap := params.AsMap()
	newestFirst, _ := paramsMap["newestFirst"].(bool)

	// check and set limit
	if val, ok := paramsMap["limit"].(int); ok && val > 0 {
		limit = val
	} else if ok && val < 0 {
		return nil, fmt.Errorf("limit must be greater than or equal to 1")
	}

	// check for verbosity of output
	verbose, _ := paramsMap["verbose"].(bool)

	// build filter
	var filterParts []string
	if filter, ok := paramsMap["filter"].(string); ok {
		if len(filter) == 0 {
			return nil, fmt.Errorf("filter cannot be empty if provided")
		}
		filterParts = append(filterParts, filter)
	}

	var startTime string
	if val, ok := paramsMap["startTime"].(string); ok && val != "" {
		if _, err := time.Parse(time.RFC3339, val); err != nil {
			return nil, fmt.Errorf("startTime must be in RFC3339 format (e.g., 2025-12-09T00:00:00Z): %w", err)
		}
		startTime = val
	} else {
		startTime = time.Now().AddDate(0, 0, -defaultStartTimeOffsetDays).Format(time.RFC3339)
	}
	filterParts = append(filterParts, fmt.Sprintf(`timestamp>="%s"`, startTime))

	if endTime, ok := paramsMap["endTime"].(string); ok && endTime != "" {
		if _, err := time.Parse(time.RFC3339, endTime); err != nil {
			return nil, fmt.Errorf("endTime must be in RFC3339 format (e.g., 2025-12-09T23:59:59Z): %w", err)
		}
		filterParts = append(filterParts, fmt.Sprintf(`timestamp<="%s"`, endTime))
	}
	combinedFilter := strings.Join(filterParts, " AND ")

	// add opts.
	opts := []logadmin.EntriesOption{
		logadmin.Filter(combinedFilter),
	}

	// set order.
	if newestFirst {
		opts = append(opts, logadmin.NewestFirst())
	}

	// set up iterator.
	it := client.Entries(ctx, opts...)

	var results []map[string]any
	for len(results) < limit {
		entry, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to iterate entries: %w", err)
		}

		result := map[string]any{
			"logName":   entry.LogName,
			"timestamp": entry.Timestamp.Format(time.RFC3339),
			"severity":  entry.Severity.String(),
			"resource": map[string]any{
				"type":   entry.Resource.Type,
				"labels": entry.Resource.Labels,
			},
		}

		if entry.Payload != nil {
			result["payload"] = entry.Payload
		}
		if verbose {
			result["insertId"] = entry.InsertID

			if len(entry.Labels) > 0 {
				result["labels"] = entry.Labels
			}

			if entry.HTTPRequest != nil {
				result["httpRequest"] = map[string]any{
					"requestMethod": entry.HTTPRequest.Request.Method,
					"requestUrl":    entry.HTTPRequest.Request.URL.String(),
					"status":        entry.HTTPRequest.Status,
					"latency":       entry.HTTPRequest.Latency.String(),
					"remoteIp":      entry.HTTPRequest.RemoteIP,
					"userAgent":     entry.HTTPRequest.Request.UserAgent(),
				}
			}

			if entry.Trace != "" {
				result["trace"] = entry.Trace
			}

			if entry.SpanID != "" {
				result["spanId"] = entry.SpanID
			}

			if entry.Operation != nil {
				result["operation"] = map[string]any{
					"id":       entry.Operation.Id,
					"producer": entry.Operation.Producer,
					"first":    entry.Operation.First,
					"last":     entry.Operation.Last,
				}
			}

			if entry.SourceLocation != nil {
				result["sourceLocation"] = map[string]any{
					"file":     entry.SourceLocation.File,
					"line":     entry.SourceLocation.Line,
					"function": entry.SourceLocation.Function,
				}
			}
		}
		results = append(results, result)
	}
	return results, nil
}

func (t Tool) ParseParams(data map[string]any, claimsMap map[string]map[string]any) (parameters.ParamValues, error) {
	return parameters.ParseParams(t.params, data, claimsMap)
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
	return t.source.UseClientAuthorization()
}

func (t Tool) ToConfig() tools.ToolConfig {
	return t.Config
}

func (t Tool) GetAuthTokenHeaderName() string {
	return "Authorization"
}
