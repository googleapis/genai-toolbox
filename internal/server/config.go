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
	"fmt"
	"strings"

	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/auth"
	"github.com/googleapis/genai-toolbox/internal/auth/google"
	"github.com/googleapis/genai-toolbox/internal/sources"
	alloydbpgsrc "github.com/googleapis/genai-toolbox/internal/sources/alloydbpg"
	cloudsqlpgsrc "github.com/googleapis/genai-toolbox/internal/sources/cloudsqlpg"
	neo4jrc "github.com/googleapis/genai-toolbox/internal/sources/neo4j"
	postgressrc "github.com/googleapis/genai-toolbox/internal/sources/postgres"
	spannersrc "github.com/googleapis/genai-toolbox/internal/sources/spanner"
	"github.com/googleapis/genai-toolbox/internal/tools"
	neo4jtool "github.com/googleapis/genai-toolbox/internal/tools/neo4j"
	"github.com/googleapis/genai-toolbox/internal/tools/postgressql"
	"github.com/googleapis/genai-toolbox/internal/tools/spanner"
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
	// AuthSourceConfigs defines what sources of authentication are available for tools.
	AuthSourceConfigs AuthSourceConfigs
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
var _ yaml.InterfaceUnmarshaler = &SourceConfigs{}

func (c *SourceConfigs) UnmarshalYAML(unmarshal func(interface{}) error) error {
	*c = make(SourceConfigs)
	// Parse the 'kind' fields for each source
	var raw map[string]util.DelayedUnmarshaler
	if err := unmarshal(&raw); err != nil {
		return err
	}

	for name, u := range raw {
		var k struct {
			Kind string `yaml:"kind"`
		}
		err := u.Unmarshal(&k)
		if err != nil {
			return fmt.Errorf("missing 'kind' field for %q", k)
		}
		switch k.Kind {
		case alloydbpgsrc.SourceKind:
			actual := alloydbpgsrc.Config{Name: name, IPType: "public"}
			if err := u.Unmarshal(&actual); err != nil {
				return fmt.Errorf("unable to parse as %q: %w", k.Kind, err)
			}
			(*c)[name] = actual
		case cloudsqlpgsrc.SourceKind:
			actual := cloudsqlpgsrc.Config{Name: name, IPType: "public"}
			if err := u.Unmarshal(&actual); err != nil {
				return fmt.Errorf("unable to parse as %q: %w", k.Kind, err)
			}
			(*c)[name] = actual
		case postgressrc.SourceKind:
			actual := postgressrc.Config{Name: name}
			if err := u.Unmarshal(&actual); err != nil {
				return fmt.Errorf("unable to parse as %q: %w", k.Kind, err)
			}
			(*c)[name] = actual
		case spannersrc.SourceKind:
			actual := spannersrc.Config{Name: name, Dialect: "googlesql"}
			if err := u.Unmarshal(&actual); err != nil {
				return fmt.Errorf("unable to parse as %q: %w", k.Kind, err)
			}
			(*c)[name] = actual
		case neo4jrc.SourceKind:
			actual := neo4jrc.Config{Name: name}
			if err := u.Unmarshal(&actual); err != nil {
				return fmt.Errorf("unable to parse as %q: %w", k.Kind, err)
			}
			(*c)[name] = actual
		default:
			return fmt.Errorf("%q is not a valid kind of data source", k.Kind)
		}

	}
	return nil
}

// AuthSourceConfigs is a type used to allow unmarshal of the data authSource config map
type AuthSourceConfigs map[string]auth.AuthSourceConfig

// validate interface
var _ yaml.InterfaceUnmarshaler = &AuthSourceConfigs{}

func (c *AuthSourceConfigs) UnmarshalYAML(unmarshal func(interface{}) error) error {
	*c = make(AuthSourceConfigs)
	// Parse the 'kind' fields for each authSource
	var raw map[string]util.DelayedUnmarshaler
	if err := unmarshal(&raw); err != nil {
		return err
	}

	for name, u := range raw {
		var k struct {
			Kind string `yaml:"kind"`
		}
		err := u.Unmarshal(&k)
		if err != nil {
			return fmt.Errorf("missing 'kind' field for %q", k)
		}
		switch k.Kind {
		case google.AuthSourceKind:
			actual := google.Config{Name: name}
			if err := u.Unmarshal(&actual); err != nil {
				return fmt.Errorf("unable to parse as %q: %w", k.Kind, err)
			}
			(*c)[name] = actual
		default:
			return fmt.Errorf("%q is not a valid kind of auth source", k.Kind)
		}
	}
	return nil
}

// ToolConfigs is a type used to allow unmarshal of the tool configs
type ToolConfigs map[string]tools.ToolConfig

// validate interface
var _ yaml.InterfaceUnmarshaler = &ToolConfigs{}

func (c *ToolConfigs) UnmarshalYAML(unmarshal func(interface{}) error) error {
	*c = make(ToolConfigs)
	// Parse the 'kind' fields for each source
	var raw map[string]util.DelayedUnmarshaler
	if err := unmarshal(&raw); err != nil {
		return err
	}

	for name, u := range raw {
		var k struct {
			Kind string `yaml:"kind"`
		}
		err := u.Unmarshal(&k)
		if err != nil {
			return fmt.Errorf("missing 'kind' field for %q", name)
		}
		switch k.Kind {
		case postgressql.ToolKind:
			actual := postgressql.Config{Name: name}
			if err := u.Unmarshal(&actual); err != nil {
				return fmt.Errorf("unable to parse as %q: %w", k.Kind, err)
			}
			(*c)[name] = actual
		case spanner.ToolKind:
			actual := spanner.Config{Name: name}
			if err := u.Unmarshal(&actual); err != nil {
				return fmt.Errorf("unable to parse as %q: %w", k.Kind, err)
			}
			(*c)[name] = actual
		case neo4jtool.ToolKind:
			actual := neo4jtool.Config{Name: name}
			if err := u.Unmarshal(&actual); err != nil {
				return fmt.Errorf("unable to parse as %q: %w", k.Kind, err)
			}
			(*c)[name] = actual
		default:
			return fmt.Errorf("%q is not a valid kind of tool", k.Kind)
		}

	}
	return nil
}

// ToolConfigs is a type used to allow unmarshal of the toolset configs
type ToolsetConfigs map[string]tools.ToolsetConfig

// validate interface
var _ yaml.InterfaceUnmarshaler = &ToolsetConfigs{}

func (c *ToolsetConfigs) UnmarshalYAML(unmarshal func(interface{}) error) error {
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
