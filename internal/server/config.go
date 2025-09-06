// Copyright 2024 Google LLC
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
package server

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/auth"
	"github.com/googleapis/genai-toolbox/internal/auth/google"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/util"
)

type ServerConfig struct {
	// Server version
	Version string
	// Address is the address of the interface the server will listen on.
	Address string
	// Port is the port the server will listen on.
	Port int
	// SourceConfigs defines what sources of data are available for tools.
	SourceConfigs SourceConfigs
	// AuthServiceConfigs defines what sources of authentication are available for tools.
	AuthServiceConfigs AuthServiceConfigs
	// ToolConfigs defines what tools are available.
	ToolConfigs ToolConfigs
	// ToolsetConfigs defines what tools are available.
	ToolsetConfigs ToolsetConfigs
	// LoggingFormat defines whether structured loggings are used.
	LoggingFormat logFormat
	// LogLevel defines the levels to log.
	LogLevel StringLevel
	// TelemetryGCP defines whether GCP exporter is used.
	TelemetryGCP bool
	// TelemetryOTLP defines OTLP collector url for telemetry exports.
	TelemetryOTLP string
	// TelemetryServiceName defines the value of service.name resource attribute.
	TelemetryServiceName string
	// Stdio indicates if Toolbox is listening via MCP stdio.
	Stdio bool
	// DisableReload indicates if the user has disabled dynamic reloading for Toolbox.
	DisableReload bool
	// UI indicates if Toolbox UI endpoints (/ui) are available
	UI bool
}

type logFormat string

// String is used by both fmt.Print and by Cobra in help text
func (f *logFormat) String() string {
	if string(*f) != "" {
		return strings.ToLower(string(*f))
	}
	return "standard"
}

// validate logging format flag
func (f *logFormat) Set(v string) error {
	switch strings.ToLower(v) {
	case "standard", "json":
		*f = logFormat(v)
		return nil
	default:
		return fmt.Errorf(`log format must be one of "standard", or "json"`)
	}
}

// Type is used in Cobra help text
func (f *logFormat) Type() string {
	return "logFormat"
}

type StringLevel string

// String is used by both fmt.Print and by Cobra in help text
func (s *StringLevel) String() string {
	if string(*s) != "" {
		return strings.ToLower(string(*s))
	}
	return "info"
}

// validate log level flag
func (s *StringLevel) Set(v string) error {
	switch strings.ToLower(v) {
	case "debug", "info", "warn", "error":
		*s = StringLevel(v)
		return nil
	default:
		return fmt.Errorf(`log level must be one of "debug", "info", "warn", or "error"`)
	}
}

// Type is used in Cobra help text
func (s *StringLevel) Type() string {
	return "stringLevel"
}

func UnmarshalResourceConfig(ctx context.Context, raw []byte) (SourceConfigs, AuthServiceConfigs, ToolConfigs, ToolsetConfigs, error) {
	// prepare configs map
	sourceConfigs := make(SourceConfigs)
	authServiceConfigs := make(AuthServiceConfigs)
	toolConfigs := make(ToolConfigs)
	toolsetConfigs := make(ToolsetConfigs)

	decoder := yaml.NewDecoder(bytes.NewReader(raw))
	// for loop to unmarshal documents with the `---` separator
	for {
		var resource map[string]any
		if err := decoder.DecodeContext(ctx, &resource); err != nil {
			if err.Error() == "EOF" {
				break
			}
			return nil, nil, nil, nil, fmt.Errorf("unable to parse kind: %s", err)
		}
		var kind, name string
		var ok bool
		if kind, ok = resource["kind"].(string); !ok {
			return nil, nil, nil, nil, fmt.Errorf("missing 'kind' field or it is not a string")
		}
		if name, ok = resource["name"].(string); !ok {
			return nil, nil, nil, nil, fmt.Errorf("missing 'name' field or it is not a string")
		}
		// remove 'kind' from map for strict unmarshaling
		delete(resource, "kind")

		switch kind {
		case "sources":
			c, err := UnmarshalYAMLSourceConfig(ctx, name, resource)
			if err != nil {
				return nil, nil, nil, nil, fmt.Errorf("error unmarshaling %s: %s", kind, err)
			}
			sourceConfigs[name] = c
		case "authServices":
			c, err := UnmarshalYAMLAuthServiceConfig(ctx, name, resource)
			if err != nil {
				return nil, nil, nil, nil, fmt.Errorf("error unmarshaling %s: %s", kind, err)
			}
			authServiceConfigs[name] = c
		case "tools":
			c, err := UnmarshalYAMLToolConfig(ctx, name, resource)
			if err != nil {
				return nil, nil, nil, nil, fmt.Errorf("error unmarshaling %s: %s", kind, err)
			}
			toolConfigs[name] = c
		case "toolsets":
			c, err := UnmarshalYAMLToolsetConfig(ctx, name, resource)
			if err != nil {
				return nil, nil, nil, nil, fmt.Errorf("error unmarshaling %s: %s", kind, err)
			}
			toolsetConfigs[name] = c
		default:
			return nil, nil, nil, nil, fmt.Errorf("invalid kind %s", kind)
		}
	}
	return sourceConfigs, authServiceConfigs, toolConfigs, toolsetConfigs, nil
}

// SourceConfigs is a type used to allow unmarshal of the data source config map
type SourceConfigs map[string]sources.SourceConfig

func UnmarshalYAMLSourceConfig(ctx context.Context, name string, r map[string]any) (sources.SourceConfig, error) {
	typeStr, ok := r["type"].(string)
	if !ok {
		return nil, fmt.Errorf("missing 'name' field or it is not a string")
	}
	dec, err := util.NewStrictDecoder(r)
	if err != nil {
		return nil, fmt.Errorf("error creating decoder: %s", err)
	}
	sourceConfig, err := sources.DecodeConfig(ctx, typeStr, name, dec)
	if err != nil {
		return nil, err
	}
	return sourceConfig, nil
}

// AuthServiceConfigs is a type used to allow unmarshal of the data authService config map
type AuthServiceConfigs map[string]auth.AuthServiceConfig

func UnmarshalYAMLAuthServiceConfig(ctx context.Context, name string, r map[string]any) (auth.AuthServiceConfig, error) {
	typeStr, ok := r["type"].(string)
	if !ok {
		return nil, fmt.Errorf("missing 'name' field or it is not a string")
	}

	if typeStr != google.AuthServiceType {
		return nil, fmt.Errorf("%s is not a valid type of auth source", typeStr)
	}
	dec, err := util.NewStrictDecoder(r)
	if err != nil {
		return nil, fmt.Errorf("error creating decoder: %s", err)
	}
	actual := google.Config{Name: name}
	if err := dec.DecodeContext(ctx, &actual); err != nil {
		return nil, fmt.Errorf("unable to parse as %s: %w", name, err)
	}
	return actual, nil
}

// ToolConfigs is a type used to allow unmarshal of the tool configs
type ToolConfigs map[string]tools.ToolConfig

func UnmarshalYAMLToolConfig(ctx context.Context, name string, r map[string]any) (tools.ToolConfig, error) {
	typeStr, ok := r["type"].(string)
	if !ok {
		return nil, fmt.Errorf("missing 'name' field or it is not a string")
	}

	// `authRequired` and `useClientOAuth` cannot be specified together
	if r["authRequired"] != nil && r["useClientOAuth"] == true {
		return nil, fmt.Errorf("`authRequired` and `useClientOAuth` are mutually exclusive. Choose only one authentication method")
	}

	// Make `authRequired` an empty list instead of nil for Tool manifest
	if r["authRequired"] == nil {
		r["authRequired"] = []string{}
	}

	dec, err := util.NewStrictDecoder(r)
	if err != nil {
		return nil, fmt.Errorf("error creating decoder: %s", err)
	}
	toolCfg, err := tools.DecodeConfig(ctx, typeStr, name, dec)
	if err != nil {
		return nil, err
	}
	return toolCfg, nil
}

// ToolConfigs is a type used to allow unmarshal of the toolset configs
type ToolsetConfigs map[string]tools.ToolsetConfig

func UnmarshalYAMLToolsetConfig(ctx context.Context, name string, r map[string]any) (tools.ToolsetConfig, error) {
	var toolsetConfig tools.ToolsetConfig
	justTools := map[string]any{"tools": r["tools"]}
	dec, err := util.NewStrictDecoder(justTools)
	if err != nil {
		return toolsetConfig, fmt.Errorf("error creating decoder: %s", err)
	}
	var raw map[string][]string
	if err := dec.DecodeContext(ctx, &raw); err != nil {
		return toolsetConfig, fmt.Errorf("unable to unmarshal tools: %s", err)
	}
	return tools.ToolsetConfig{Name: name, ToolNames: raw["tools"]}, nil
}
