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
	actual := KuzuDbConfig{Name: name}
	if err := decoder.DecodeContext(ctx, &actual); err != nil {
		return nil, err
	}
	return actual, nil
}

type KuzuDbConfig struct {
	Name              string `yaml:"name" validate:"required" `
	Kind              string `yaml:"kind" validate:"required"`
	Database          string `yaml:"database"`
	BufferPoolSize    uint64 `yaml:"bufferPoolSize"`
	MaxNumThreads     uint64 `yaml:"maxNumThreads"`
	EnableCompression bool   `yaml:"enableCompression"`
	ReadOnly          bool   `yaml:"readOnly"`
	MaxDbSize         uint64 `yaml:"maxDbSize"`
}

func (c KuzuDbConfig) SourceConfigKind() string {
	return KuzuDbKind
}

func (c KuzuDbConfig) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {
	conn, err := initKuzuDbConnection(ctx, tracer, c)
	if err != nil {
		return nil, fmt.Errorf("unable to open a database connection: %w", err)
	}

	source := &KuzuDbSource{
		Name:       c.Name,
		Kind:       KuzuDbKind,
		Connection: conn,
	}
	return source, nil
}

var _ sources.SourceConfig = KuzuDbConfig{}

type KuzuDbSource struct {
	Name       string `yaml:"name"`
	Kind       string `yaml:"kind"`
	Connection *kuzu.Connection
}

// SourceKind implements sources.Source.
func (k *KuzuDbSource) SourceKind() string {
	return KuzuDbKind
}

func (k *KuzuDbSource) KuzuDB() *kuzu.Connection {
	return k.Connection
}

var _ sources.Source = &KuzuDbSource{}

func initKuzuDbConnection(ctx context.Context, tracer trace.Tracer, config KuzuDbConfig) (*kuzu.Connection, error) {
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
