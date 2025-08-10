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

package cassandra

import (
	"context"
	"fmt"

	"github.com/goccy/go-yaml"
	"github.com/gocql/gocql"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"go.opentelemetry.io/otel/trace"
)

const SourceKind string = "cassandra"

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
	Name                   string   `yaml:"name" validate:"required"`
	Kind                   string   `yaml:"kind" validate:"required"`
	Hosts                  []string `yaml:"hosts" validate:"required"`
	Keyspace               string   `yaml:"keyspace"`
	ProtoVersion           int      `yaml:"protoVersion"`
	Username               string   `yaml:"username"`
	Password               string   `yaml:"password"`
	CAPath                 string   `yaml:"caPath"`
	CertPath               string   `yaml:"certPath"`
	KeyPath                string   `yaml:"keyPath"`
	EnableHostVerification bool     `yaml:"enableHostVerification"`
}

// Initialize implements sources.SourceConfig.
func (c Config) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {
	session, err := initCassandraSession(ctx, tracer, c)
	if err != nil {
		return nil, fmt.Errorf("unable to create session: %v", err)
	}
	s := &Source{
		Name:    c.Name,
		Kind:    SourceKind,
		Session: session,
	}
	return s, nil
}

// SourceConfigKind implements sources.SourceConfig.
func (c Config) SourceConfigKind() string {
	return SourceKind
}

var _ sources.SourceConfig = Config{}

type Source struct {
	Name    string `yaml:"name"`
	Kind    string `yaml:"kind"`
	Session *gocql.Session
}

// CassandraSession implements cassandra.compatibleSource.
func (s *Source) CassandraSession() *gocql.Session {
	return s.Session
}

// SourceKind implements sources.Source.
func (s Source) SourceKind() string {
	return SourceKind
}

var _ sources.Source = &Source{}

func initCassandraSession(ctx context.Context, tracer trace.Tracer, c Config) (*gocql.Session, error) {
	//nolint:all // Reassigned ctx
	ctx, span := sources.InitConnectionSpan(ctx, tracer, SourceKind, c.Name)
	defer span.End()

	cluster := gocql.NewCluster(c.Hosts...)
	cluster.ProtoVersion = c.ProtoVersion
	cluster.Keyspace = c.Keyspace
	if c.Username != "" && c.Password != "" {
		cluster.Authenticator = gocql.PasswordAuthenticator{
			Username: c.Username,
			Password: c.Password,
		}
	}
	if c.CAPath != "" || c.CertPath != "" || c.KeyPath != "" || c.EnableHostVerification {
		cluster.SslOpts = &gocql.SslOptions{
			CaPath:                 c.CAPath,
			CertPath:               c.CertPath,
			KeyPath:                c.KeyPath,
			EnableHostVerification: c.EnableHostVerification,
		}
	}
	// Create session
	session, err := cluster.CreateSession()
	if err != nil {
		return nil, err
	}
	return session, nil
}
