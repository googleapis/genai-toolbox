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
	actual := &KuzuDbConfig{Name: name}
	if err := decoder.DecodeContext(ctx, &actual); err != nil {
		return nil, err
	}
	return actual, nil
}

type KuzuDbConfig struct {
	Name          string            `yaml:"name" validate:"required"`
	Kind          string            `yaml:"kind" validate:"required"`
	Database      string            `yaml:"database" validate:"required"`
	Configuration kuzu.SystemConfig `yaml:"configuration,omitempty"`
}

// SourceKind implements sources.Source.
func (c KuzuDbConfig) SourceKind() string {
	return KuzuDbKind
}

// Initialize implements sources.SourceConfig.
func (c KuzuDbConfig) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {
	conn, err := initKuzuDbConnection(ctx, tracer, c.Name, c.Database)
	if err != nil {
		return nil, fmt.Errorf("unable to open a database connection: %w", err)
	}

	source := &KuzuDbSource{
		Kind:       KuzuDbKind,
		Connection: conn,
	}
	return source, nil
}

// SourceConfigKind implements sources.SourceConfig.
func (c KuzuDbConfig) SourceConfigKind() string {
	return KuzuDbKind
}

var _ sources.SourceConfig = KuzuDbConfig{}

type KuzuDbSource struct {
	Kind       string `yaml:"kind" validated:"required"`
	Connection *kuzu.Connection
}

// SourceKind implements sources.Source.
func (k KuzuDbSource) SourceKind() string {
	return KuzuDbKind
}

func (k KuzuDbSource) KuzuDB() *kuzu.Connection {
	return k.Connection
}

var _ sources.Source = KuzuDbSource{}

// TODO configure custom config
func initKuzuDbConnection(ctx context.Context, tracer trace.Tracer, name, database string) (*kuzu.Connection, error) {
	//nolint:all // Reassigned ctx
	ctx, span := sources.InitConnectionSpan(ctx, tracer, KuzuDbKind, name)
	defer span.End()
	systemConfig := kuzu.DefaultSystemConfig()
	systemConfig.BufferPoolSize = 1024 * 1024 * 1024
	db, err := kuzu.OpenDatabase(database, systemConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %w", err)
	}

	conn, err := kuzu.OpenConnection(db)
	if err != nil {
		return nil, fmt.Errorf("unable to open a database connection: %w", err)
	}
	return conn, nil
}
