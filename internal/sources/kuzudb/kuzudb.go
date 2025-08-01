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

package kuzudb

import (
	"context"
	"fmt"

	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/kuzudb/go-kuzu"
	"go.opentelemetry.io/otel/trace"
)

var KuzuDbKind string = "kuzudb"

func init() {
	if !sources.Register(KuzuDbKind, newConfig) {
		panic(fmt.Sprintf("source kind %q already registered", KuzuDbKind))
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
	Name              string `yaml:"name" validate:"required" `
	Kind              string `yaml:"kind" validate:"required"`
	Database          string `yaml:"database"`
	BufferPoolSize    uint64 `yaml:"bufferPoolSize"`
	MaxNumThreads     uint64 `yaml:"maxNumThreads"`
	EnableCompression bool   `yaml:"enableCompression"`
	ReadOnly          bool   `yaml:"readOnly"`
	MaxDbSize         uint64 `yaml:"maxDbSize"`
}

func (cfg Config) SourceConfigKind() string {
	return KuzuDbKind
}

func (cfg Config) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {
	conn, err := initKuzuDbConnection(ctx, tracer, cfg)
	if err != nil {
		return nil, fmt.Errorf("unable to open a database connection: %w", err)
	}

	source := &Source{
		Name:       cfg.Name,
		Kind:       KuzuDbKind,
		Connection: conn,
	}
	return source, nil
}

var _ sources.SourceConfig = Config{}

type Source struct {
	Name       string `yaml:"name"`
	Kind       string `yaml:"kind"`
	Connection *kuzu.Connection
}

// SourceKind implements sources.Source.
func (s *Source) SourceKind() string {
	return KuzuDbKind
}

func (s *Source) KuzuDB() *kuzu.Connection {
	return s.Connection
}

var _ sources.Source = &Source{}

func initKuzuDbConnection(ctx context.Context, tracer trace.Tracer, config Config) (*kuzu.Connection, error) {
	//nolint:all // Reassigned ctx
	ctx, span := sources.InitConnectionSpan(ctx, tracer, KuzuDbKind, config.Name)
	defer span.End()
	systemConfig := kuzu.DefaultSystemConfig()
	if config.BufferPoolSize != 0 {
		systemConfig.BufferPoolSize = config.BufferPoolSize
	}
	if config.EnableCompression {
		systemConfig.EnableCompression = config.EnableCompression
	}
	if config.MaxDbSize != 0 {
		systemConfig.MaxDbSize = config.MaxDbSize
	}
	if config.ReadOnly {
		systemConfig.ReadOnly = config.ReadOnly
	}
	if config.MaxNumThreads != 0 {
		systemConfig.MaxNumThreads = config.MaxNumThreads
	}
	var db *kuzu.Database
	var err error
	if config.Database != "" {
		db, err = kuzu.OpenDatabase(config.Database, systemConfig)
	} else {
		db, err = kuzu.OpenInMemoryDatabase(systemConfig)
	}
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %w", err)
	}

	conn, err := kuzu.OpenConnection(db)
	if err != nil {
		return nil, fmt.Errorf("unable to open a database connection: %w", err)
	}
	return conn, nil
}
