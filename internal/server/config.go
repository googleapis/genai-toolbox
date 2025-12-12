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
	"context"
	"fmt"
	"strings"

	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/auth"
	"github.com/googleapis/genai-toolbox/internal/auth/google"
	"github.com/googleapis/genai-toolbox/internal/prompts"
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
	// PromptConfigs defines what prompts are available
	PromptConfigs PromptConfigs
	// PromptsetConfigs defines what prompts are available
	PromptsetConfigs PromptsetConfigs
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
	// Specifies a list of origins permitted to access this server.
	AllowedOrigins []string
	// source file for /.well-known/oauth-protected-resource.
	OAuthProtectedResource string
	// source file for /.well-known/oauth-authorization-server.
	OAuthAuthorizationServer string
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

// SourceConfigs is a type used to allow unmarshal of the data source config map
type SourceConfigs map[string]sources.SourceConfig

// validate interface
var _ yaml.InterfaceUnmarshalerContext = &SourceConfigs{}

func (c *SourceConfigs) UnmarshalYAML(ctx context.Context, unmarshal func(interface{}) error) error {
	*c = make(SourceConfigs)
	// Parse the 'kind' fields for each source
	var raw map[string]util.DelayedUnmarshaler
	if err := unmarshal(&raw); err != nil {
		return err
	}

	for name, u := range raw {
		// Unmarshal to a general type that ensure it capture all fields
		var v map[string]any
		if err := u.Unmarshal(&v); err != nil {
			return fmt.Errorf("unable to unmarshal %q: %w", name, err)
		}

		kind, ok := v["kind"]
		if !ok {
			return fmt.Errorf("missing 'kind' field for source %q", name)
		}
		kindStr, ok := kind.(string)
		if !ok {
			return fmt.Errorf("invalid 'kind' field for source %q (must be a string)", name)
		}

		yamlDecoder, err := util.NewStrictDecoder(v)
		if err != nil {
			return fmt.Errorf("error creating YAML decoder for source %q: %w", name, err)
		}

		sourceConfig, err := sources.DecodeConfig(ctx, kindStr, name, yamlDecoder)
		if err != nil {
			return err
		}
		(*c)[name] = sourceConfig
	}
	return nil
}

// AuthServiceConfigs is a type used to allow unmarshal of the data authService config map
type AuthServiceConfigs map[string]auth.AuthServiceConfig

// validate interface
var _ yaml.InterfaceUnmarshalerContext = &AuthServiceConfigs{}

func (c *AuthServiceConfigs) UnmarshalYAML(ctx context.Context, unmarshal func(interface{}) error) error {
	*c = make(AuthServiceConfigs)
	// Parse the 'kind' fields for each authService
	var raw map[string]util.DelayedUnmarshaler
	if err := unmarshal(&raw); err != nil {
		return err
	}

	for name, u := range raw {
		var v map[string]any
		if err := u.Unmarshal(&v); err != nil {
			return fmt.Errorf("unable to unmarshal %q: %w", name, err)
		}

		kind, ok := v["kind"]
		if !ok {
			return fmt.Errorf("missing 'kind' field for %q", name)
		}

		dec, err := util.NewStrictDecoder(v)
		if err != nil {
			return fmt.Errorf("error creating decoder: %w", err)
		}
		switch kind {
		case google.AuthServiceKind:
			actual := google.Config{Name: name}
			if err := dec.DecodeContext(ctx, &actual); err != nil {
				return fmt.Errorf("unable to parse as %q: %w", kind, err)
			}
			(*c)[name] = actual
		default:
			return fmt.Errorf("%q is not a valid kind of auth source", kind)
		}
	}
	return nil
}

// ToolConfigs is a type used to allow unmarshal of the tool configs
type ToolConfigs map[string]tools.ToolConfig

// validate interface
var _ yaml.InterfaceUnmarshalerContext = &ToolConfigs{}

func (c *ToolConfigs) UnmarshalYAML(ctx context.Context, unmarshal func(interface{}) error) error {
	*c = make(ToolConfigs)
	// Parse the 'kind' fields for each source
	var raw map[string]util.DelayedUnmarshaler
	if err := unmarshal(&raw); err != nil {
		return err
	}

	for name, u := range raw {
		var v map[string]any
		if err := u.Unmarshal(&v); err != nil {
			return fmt.Errorf("unable to unmarshal %q: %w", name, err)
		}

		// `authRequired` and `useClientOAuth` cannot be specified together
		if v["authRequired"] != nil && v["useClientOAuth"] == true {
			return fmt.Errorf("`authRequired` and `useClientOAuth` are mutually exclusive. Choose only one authentication method")
		}

		// Make `authRequired` an empty list instead of nil for Tool manifest
		if v["authRequired"] == nil {
			v["authRequired"] = []string{}
		}

		kindVal, ok := v["kind"]
		if !ok {
			return fmt.Errorf("missing 'kind' field for tool %q", name)
		}
		kindStr, ok := kindVal.(string)
		if !ok {
			return fmt.Errorf("invalid 'kind' field for tool %q (must be a string)", name)
		}

		yamlDecoder, err := util.NewStrictDecoder(v)
		if err != nil {
			return fmt.Errorf("error creating YAML decoder for tool %q: %w", name, err)
		}

		toolCfg, err := tools.DecodeConfig(ctx, kindStr, name, yamlDecoder)
		if err != nil {
			return err
		}
		(*c)[name] = toolCfg
	}
	return nil
}

// ToolsetConfigs is a type used to allow unmarshal of the toolset configs
type ToolsetConfigs map[string]tools.ToolsetConfig

// validate interface
var _ yaml.InterfaceUnmarshalerContext = &ToolsetConfigs{}

func (c *ToolsetConfigs) UnmarshalYAML(ctx context.Context, unmarshal func(interface{}) error) error {
	*c = make(ToolsetConfigs)

	var raw map[string][]string
	if err := unmarshal(&raw); err != nil {
		return err
	}

	for name, toolList := range raw {
		(*c)[name] = tools.ToolsetConfig{Name: name, ToolNames: toolList}
	}
	return nil
}

// PromptConfigs is a type used to allow unmarshal of the prompt configs
type PromptConfigs map[string]prompts.PromptConfig

// validate interface
var _ yaml.InterfaceUnmarshalerContext = &PromptConfigs{}

func (c *PromptConfigs) UnmarshalYAML(ctx context.Context, unmarshal func(interface{}) error) error {
	*c = make(PromptConfigs)
	var raw map[string]util.DelayedUnmarshaler
	if err := unmarshal(&raw); err != nil {
		return err
	}

	for name, u := range raw {
		var v map[string]any
		if err := u.Unmarshal(&v); err != nil {
			return fmt.Errorf("unable to unmarshal prompt %q: %w", name, err)
		}

		// Look for the 'kind' field. If it's not present, kindStr will be an
		// empty string, which prompts.DecodeConfig will correctly default to "custom".
		var kindStr string
		if kindVal, ok := v["kind"]; ok {
			var isString bool
			kindStr, isString = kindVal.(string)
			if !isString {
				return fmt.Errorf("invalid 'kind' field for prompt %q (must be a string)", name)
			}
		}

		// Create a new, strict decoder for this specific prompt's data.
		yamlDecoder, err := util.NewStrictDecoder(v)
		if err != nil {
			return fmt.Errorf("error creating YAML decoder for prompt %q: %w", name, err)
		}

		// Use the central registry to decode the prompt based on its kind.
		promptCfg, err := prompts.DecodeConfig(ctx, kindStr, name, yamlDecoder)
		if err != nil {
			return err
		}
		(*c)[name] = promptCfg
	}
	return nil
}

// PromptsetConfigs is a type used to allow unmarshal of the PromptsetConfigs configs
type PromptsetConfigs map[string]prompts.PromptsetConfig

// validate interface
var _ yaml.InterfaceUnmarshalerContext = &PromptsetConfigs{}

func (c *PromptsetConfigs) UnmarshalYAML(ctx context.Context, unmarshal func(interface{}) error) error {
	*c = make(PromptsetConfigs)

	var raw map[string][]string
	if err := unmarshal(&raw); err != nil {
		return err
	}

	for name, promptList := range raw {
		(*c)[name] = prompts.PromptsetConfig{Name: name, PromptNames: promptList}
	}
	return nil
}
