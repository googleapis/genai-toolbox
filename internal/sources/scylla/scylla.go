// Copyright 2025 Google LLC
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

package scylla

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/gocql/gocql"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"go.opentelemetry.io/otel/trace"
)

const SourceKind string = "scylla"

func init() {
	if !sources.Register(SourceKind, newConfig) {
		panic(fmt.Sprintf("source kind %q already registered", SourceKind))
	}
}

func newConfig(ctx context.Context, name string, decoder *yaml.Decoder) (sources.SourceConfig, error) {
	actual := Config{Name: name}
	if err := decoder.DecodeContext(ctx, &actual); err != nil {
		return nil, err
	}
	return actual, nil
}

type Config struct {
	Name                     string   `yaml:"name" validate:"required"`
	Kind                     string   `yaml:"kind" validate:"required"`
	Hosts                    []string `yaml:"hosts" validate:"required"`
	Port                     string   `yaml:"port" validate:"required"`
	Keyspace                 string   `yaml:"keyspace" validate:"required"`
	Username                 string   `yaml:"username"`
	Password                 string   `yaml:"password"`
	Consistency              string   `yaml:"consistency"`
	ConnectTimeout           string   `yaml:"connectTimeout"`
	Timeout                  string   `yaml:"timeout"`
	DisableInitialHostLookup bool     `yaml:"disableInitialHostLookup"`
	NumConnections           int      `yaml:"numConnections"`
	ProtoVersion             int      `yaml:"protoVersion"`
	SSLEnabled               bool     `yaml:"sslEnabled"`
}

// validate interface
var _ sources.SourceConfig = Config{}

func (cfg Config) SourceConfigKind() string {
	return SourceKind
}

func (cfg Config) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {
	session, err := createScyllaSession(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create Scylla session: %w", err)
	}

	return &Source{
		Name:    cfg.Name,
		session: session,
	}, nil
}

type Source struct {
	Name    string
	session *gocql.Session
}

// validate interface
var _ sources.Source = &Source{}

func (s *Source) SourceKind() string {
	return SourceKind
}

func (s *Source) ScyllaSession() *gocql.Session {
	return s.session
}

func (s *Source) Close() error {
	if s.session != nil {
		s.session.Close()
	}
	return nil
}

func createScyllaSession(cfg Config) (*gocql.Session, error) {
	// Parse port
	port, err := strconv.Atoi(cfg.Port)
	if err != nil {
		return nil, fmt.Errorf("invalid port: %w", err)
	}

	// Create cluster configuration
	cluster := gocql.NewCluster(cfg.Hosts...)
	cluster.Port = port
	cluster.Keyspace = cfg.Keyspace

	// Set authentication
	if cfg.Username != "" && cfg.Password != "" {
		cluster.Authenticator = gocql.PasswordAuthenticator{
			Username: cfg.Username,
			Password: cfg.Password,
		}
	}

	// Set consistency level
	if cfg.Consistency != "" {
		consistency, err := parseConsistency(cfg.Consistency)
		if err != nil {
			return nil, fmt.Errorf("invalid consistency level: %w", err)
		}
		cluster.Consistency = consistency
	}

	// Set timeouts
	if cfg.ConnectTimeout != "" {
		duration, err := time.ParseDuration(cfg.ConnectTimeout)
		if err != nil {
			return nil, fmt.Errorf("invalid connect timeout: %w", err)
		}
		cluster.ConnectTimeout = duration
	}

	if cfg.Timeout != "" {
		duration, err := time.ParseDuration(cfg.Timeout)
		if err != nil {
			return nil, fmt.Errorf("invalid timeout: %w", err)
		}
		cluster.Timeout = duration
	}

	// Set connection pool configuration
	if cfg.NumConnections > 0 {
		cluster.NumConns = cfg.NumConnections
	}

	// Set protocol version
	if cfg.ProtoVersion > 0 {
		cluster.ProtoVersion = cfg.ProtoVersion
	}

	// Disable initial host lookup if configured
	if cfg.DisableInitialHostLookup {
		cluster.DisableInitialHostLookup = true
	}

	// SSL configuration
	if cfg.SSLEnabled {
		cluster.SslOpts = &gocql.SslOptions{
			EnableHostVerification: true,
		}
	}

	// Create session
	session, err := cluster.CreateSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return session, nil
}

func parseConsistency(consistency string) (gocql.Consistency, error) {
	switch strings.ToUpper(consistency) {
	case "ANY":
		return gocql.Any, nil
	case "ONE":
		return gocql.One, nil
	case "TWO":
		return gocql.Two, nil
	case "THREE":
		return gocql.Three, nil
	case "QUORUM":
		return gocql.Quorum, nil
	case "ALL":
		return gocql.All, nil
	case "LOCAL_QUORUM":
		return gocql.LocalQuorum, nil
	case "EACH_QUORUM":
		return gocql.EachQuorum, nil
	case "LOCAL_ONE":
		return gocql.LocalOne, nil
	default:
		return 0, fmt.Errorf("unknown consistency level: %s", consistency)
	}
}
