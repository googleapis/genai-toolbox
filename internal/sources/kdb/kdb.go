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

package kdb

import (
	"context"
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	kdbgo "github.com/sv/kdbgo"
	"go.opentelemetry.io/otel/trace"
)

const SourceKind string = "kdb"

// validate interface
var _ sources.SourceConfig = Config{}

func init() {
	if !sources.Register(SourceKind, newConfig) {
		panic(fmt.Sprintf("source kind %q already registered", SourceKind))
	}
}

var validate = validator.New()

func newConfig(ctx context.Context, name string, decoder *yaml.Decoder) (sources.SourceConfig, error) {
	actual := &Config{Name: name}
	if err := decoder.DecodeContext(ctx, actual); err != nil {
		return nil, err
	}
	if err := validate.Struct(actual); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}
	return actual, nil
}

type Config struct {
	Name     string
	Kind     string `yaml:"kind" validate:"required"`
	Host     string `yaml:"host" validate:"required"`
	Port     int    `yaml:"port" validate:"required"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// SourceConfigKind returns the source kind.
func (c Config) SourceConfigKind() string {
	return SourceKind
}

func (c Config) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {
	db, err := initKDBConnection(ctx, tracer, c.Name, c.Host, c.Port, c.Username, c.Password)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection: %w", err)
	}

	// Ping the database to check the connection.
	if _, err := db.Call(".z.p"); err != nil {
		return nil, fmt.Errorf("unable to connect successfully: %w", err)
	}

	s := &Source{
		Name: c.Name,
		Kind: SourceKind,
		DB:   db,
	}
	return s, nil
}

func initKDBConnection(ctx context.Context, tracer trace.Tracer, name, host string, port int, user, pass string) (*kdbgo.KDBConn, error) {
	_, span := sources.InitConnectionSpan(ctx, tracer, SourceKind, name)
	defer span.End()

	var auth string
	if user != "" {
		auth = user + ":" + pass
	}

	db, err := kdbgo.DialKDB(host, port, auth)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to kdb: %w", err)
	}

	return db, nil
}

var _ sources.Source = &Source{}

// Source is a kdb+ source.
type Source struct {
	Name string
	Kind string `yaml:"kind"`
	DB   *kdbgo.KDBConn
}

func (s *Source) SourceKind() string {
	return SourceKind
}

func (s *Source) KDB() *kdbgo.KDBConn {
	return s.DB
}

func (s *Source) KdbConnection() *kdbgo.KDBConn {
	return s.DB
}
