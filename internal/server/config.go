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
	"github.com/googleapis/genai-toolbox/internal/sources"
	alloydbpgsrc "github.com/googleapis/genai-toolbox/internal/sources/alloydbpg"
	bigquerysrc "github.com/googleapis/genai-toolbox/internal/sources/bigquery"
	bigtablesrc "github.com/googleapis/genai-toolbox/internal/sources/bigtable"
	cloudsqlmssqlsrc "github.com/googleapis/genai-toolbox/internal/sources/cloudsqlmssql"
	cloudsqlmysqlsrc "github.com/googleapis/genai-toolbox/internal/sources/cloudsqlmysql"
	cloudsqlpgsrc "github.com/googleapis/genai-toolbox/internal/sources/cloudsqlpg"
	couchbasesrc "github.com/googleapis/genai-toolbox/internal/sources/couchbase"
	dgraphsrc "github.com/googleapis/genai-toolbox/internal/sources/dgraph"
	httpsrc "github.com/googleapis/genai-toolbox/internal/sources/http"
	mssqlsrc "github.com/googleapis/genai-toolbox/internal/sources/mssql"
	mysqlsrc "github.com/googleapis/genai-toolbox/internal/sources/mysql"
	neo4jsrc "github.com/googleapis/genai-toolbox/internal/sources/neo4j"
	postgressrc "github.com/googleapis/genai-toolbox/internal/sources/postgres"
	spannersrc "github.com/googleapis/genai-toolbox/internal/sources/spanner"
	sqlitesrc "github.com/googleapis/genai-toolbox/internal/sources/sqlite"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/tools/alloydbainl"
	"github.com/googleapis/genai-toolbox/internal/tools/bigtable"
	couchbasetool "github.com/googleapis/genai-toolbox/internal/tools/couchbase"
	"github.com/googleapis/genai-toolbox/internal/tools/dgraph"
	httptool "github.com/googleapis/genai-toolbox/internal/tools/http"
	"github.com/googleapis/genai-toolbox/internal/tools/mssqlsql"
	"github.com/googleapis/genai-toolbox/internal/tools/mysqlsql"
	neo4jtool "github.com/googleapis/genai-toolbox/internal/tools/neo4j"
	"github.com/googleapis/genai-toolbox/internal/tools/postgresexecutesql"
	"github.com/googleapis/genai-toolbox/internal/tools/postgressql"
	"github.com/googleapis/genai-toolbox/internal/tools/spanner"
	"github.com/googleapis/genai-toolbox/internal/tools/sqlitesql"
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
			return fmt.Errorf("missing 'kind' field for %q", name)
		}

		dec, err := util.NewStrictDecoder(v)
		if err != nil {
			return fmt.Errorf("error creating decoder: %w", err)
		}
		switch kind {
		case alloydbpgsrc.SourceKind:
			actual := alloydbpgsrc.Config{Name: name, IPType: "public"}
			if err := dec.DecodeContext(ctx, &actual); err != nil {
				return fmt.Errorf("unable to parse as %q: %w", kind, err)
			}
			(*c)[name] = actual
		case bigtablesrc.SourceKind:
			actual := bigtablesrc.Config{Name: name}
			if err := dec.DecodeContext(ctx, &actual); err != nil {
				return fmt.Errorf("unable to parse as %q: %w", kind, err)
			}
			(*c)[name] = actual
		case cloudsqlpgsrc.SourceKind:
			actual := cloudsqlpgsrc.Config{Name: name, IPType: "public"}
			if err := dec.DecodeContext(ctx, &actual); err != nil {
				return fmt.Errorf("unable to parse as %q: %w", kind, err)
			}
			(*c)[name] = actual
		case postgressrc.SourceKind:
			actual := postgressrc.Config{Name: name}
			if err := dec.DecodeContext(ctx, &actual); err != nil {
				return fmt.Errorf("unable to parse as %q: %w", kind, err)
			}
			(*c)[name] = actual
		case cloudsqlmysqlsrc.SourceKind:
			actual := cloudsqlmysqlsrc.Config{Name: name, IPType: "public"}
			if err := dec.DecodeContext(ctx, &actual); err != nil {
				return fmt.Errorf("unable to parse as %q: %w", kind, err)
			}
			(*c)[name] = actual
		case mysqlsrc.SourceKind:
			actual := mysqlsrc.Config{Name: name}
			if err := dec.DecodeContext(ctx, &actual); err != nil {
				return fmt.Errorf("unable to parse as %q: %w", kind, err)
			}
			(*c)[name] = actual
		case spannersrc.SourceKind:
			actual := spannersrc.Config{Name: name, Dialect: "googlesql"}
			if err := dec.DecodeContext(ctx, &actual); err != nil {
				return fmt.Errorf("unable to parse as %q: %w", kind, err)
			}
			(*c)[name] = actual
		case neo4jsrc.SourceKind:
			actual := neo4jsrc.Config{Name: name}
			if err := dec.DecodeContext(ctx, &actual); err != nil {
				return fmt.Errorf("unable to parse as %q: %w", kind, err)
			}
			(*c)[name] = actual
		case cloudsqlmssqlsrc.SourceKind:
			actual := cloudsqlmssqlsrc.Config{Name: name, IPType: "public"}
			if err := dec.DecodeContext(ctx, &actual); err != nil {
				return fmt.Errorf("unable to parse as %q: %w", kind, err)
			}
			(*c)[name] = actual
		case mssqlsrc.SourceKind:
			actual := mssqlsrc.Config{Name: name}
			if err := dec.DecodeContext(ctx, &actual); err != nil {
				return fmt.Errorf("unable to parse as %q: %w", kind, err)
			}
			(*c)[name] = actual
		case dgraphsrc.SourceKind:
			actual := dgraphsrc.Config{Name: name}
			if err := dec.DecodeContext(ctx, &actual); err != nil {
				return fmt.Errorf("unable to parse as %q: %w", kind, err)
			}
			(*c)[name] = actual
		case httpsrc.SourceKind:
			actual := httpsrc.DefaultConfig(name)
			if err := dec.DecodeContext(ctx, &actual); err != nil {
				return fmt.Errorf("unable to parse as %q: %w", kind, err)
			}
			(*c)[name] = actual
		case bigquerysrc.SourceKind:
			actual := bigquerysrc.Config{Name: name}
			if err := dec.DecodeContext(ctx, &actual); err != nil {
				return fmt.Errorf("unable to parse as %q: %w", kind, err)
			}
			(*c)[name] = actual
		case sqlitesrc.SourceKind:
			actual := sqlitesrc.Config{Name: name}
			if err := dec.DecodeContext(ctx, &actual); err != nil {
		case couchbasesrc.SourceKind:
			actual := couchbasesrc.Config{Name: name}
			if err := dec.DecodeContext(ctx, &actual); err != nil {
				return fmt.Errorf("unable to parse as %q: %w", kind, err)
			}
			(*c)[name] = actual
		case couchbasesrc.SourceKind:
			actual := couchbasesrc.Config{Name: name}
			if err := dec.DecodeContext(ctx, &actual); err != nil {
				return fmt.Errorf("unable to parse as %q: %w", kind, err)
			}
			(*c)[name] = actual
		default:
			return fmt.Errorf("%q is not a valid kind of data source", kind)
		}

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

		// Make `authRequired` an empty list instead of nil for Tool manifest
		if v["authRequired"] == nil {
			v["authRequired"] = []string{}
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
		case bigtable.ToolKind:
			actual := bigtable.Config{Name: name}
			if err := dec.DecodeContext(ctx, &actual); err != nil {
				return fmt.Errorf("unable to parse as %q: %w", kind, err)
			}
			(*c)[name] = actual
		case postgressql.ToolKind:
			actual := postgressql.Config{Name: name}
			if err := dec.DecodeContext(ctx, &actual); err != nil {
				return fmt.Errorf("unable to parse as %q: %w", kind, err)
			}
			(*c)[name] = actual
		case alloydbainl.ToolKind:
			actual := alloydbainl.Config{Name: name}
			if err := dec.DecodeContext(ctx, &actual); err != nil {
				return fmt.Errorf("unable to parse as %q: %w", kind, err)
			}
			(*c)[name] = actual
		case mysqlsql.ToolKind:
			actual := mysqlsql.Config{Name: name}
			if err := dec.DecodeContext(ctx, &actual); err != nil {
				return fmt.Errorf("unable to parse as %q: %w", kind, err)
			}
			(*c)[name] = actual
		case spanner.ToolKind:
			actual := spanner.Config{Name: name}
			if err := dec.DecodeContext(ctx, &actual); err != nil {
				return fmt.Errorf("unable to parse as %q: %w", kind, err)
			}
			(*c)[name] = actual
		case neo4jtool.ToolKind:
			actual := neo4jtool.Config{Name: name}
			if err := dec.DecodeContext(ctx, &actual); err != nil {
				return fmt.Errorf("unable to parse as %q: %w", kind, err)
			}
			(*c)[name] = actual
		case mssqlsql.ToolKind:
			actual := mssqlsql.Config{Name: name}
			if err := dec.DecodeContext(ctx, &actual); err != nil {
				return fmt.Errorf("unable to parse as %q: %w", kind, err)
			}
			(*c)[name] = actual
		case dgraph.ToolKind:
			actual := dgraph.Config{Name: name}
			if err := dec.DecodeContext(ctx, &actual); err != nil {
				return fmt.Errorf("unable to parse as %q: %w", kind, err)
			}
			(*c)[name] = actual
		case httptool.ToolKind:
			actual := httptool.Config{Name: name}
			if err := dec.DecodeContext(ctx, &actual); err != nil {
				return fmt.Errorf("unable to parse as %q: %w", kind, err)
			}
			(*c)[name] = actual
		case bigquery.ToolKind:
			actual := bigquery.Config{Name: name}
			if err := dec.DecodeContext(ctx, &actual); err != nil {
				return fmt.Errorf("unable to parse as %q: %w", kind, err)
			}
			(*c)[name] = actual
		case sqlitesql.ToolKind:
			actual := sqlitesql.Config{Name: name}
			if err := dec.DecodeContext(ctx, &actual); err != nil {
		case couchbasetool.ToolKind:
			actual := couchbasetool.Config{Name: name}
			if err := dec.DecodeContext(ctx, &actual); err != nil {
				return fmt.Errorf("unable to parse as %q: %w", kind, err)
			}
			(*c)[name] = actual
		case postgresexecutesql.ToolKind:
			actual := postgresexecutesql.Config{Name: name}
			if err := dec.DecodeContext(ctx, &actual); err != nil {
				return fmt.Errorf("unable to parse as %q: %w", kind, err)
			}
			(*c)[name] = actual
		case couchbasetool.ToolKind:
			actual := couchbasetool.Config{Name: name}
			if err := dec.DecodeContext(ctx, &actual); err != nil {
				return fmt.Errorf("unable to parse as %q: %w", kind, err)
			}
			(*c)[name] = actual
		default:
			return fmt.Errorf("%q is not a valid kind of tool", kind)
		}

	}
	return nil
}

// ToolConfigs is a type used to allow unmarshal of the toolset configs
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
