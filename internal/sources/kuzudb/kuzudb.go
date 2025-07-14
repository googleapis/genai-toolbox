package kuzudb

import (
	"context"

	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/kuzudb/go-kuzu"
	"go.opentelemetry.io/otel/trace"
)

var KuzuDbKind string = "kuzudb"

func init() {

}

type KuzuDbConfig struct {
	Kind     string `yaml:"kind" validated:"required"`
	Database string `yaml:"database" validated:"required"`
}

// SourceKind implements sources.Source.
func (c KuzuDbConfig) SourceKind() string {
	return KuzuDbKind
}

// Initialize implements sources.SourceConfig.
func (c KuzuDbConfig) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {
	systemConfig := kuzu.DefaultSystemConfig()
	systemConfig.BufferPoolSize = 1024 * 1024 * 1024
	db, err := kuzu.OpenDatabase("example.kuzu", systemConfig)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	conn, err := kuzu.OpenConnection(db)
	if err != nil {
		panic(err)
	}
}

// SourceConfigKind implements sources.SourceConfig.
func (c KuzuDbConfig) SourceConfigKind() string {
	return KuzuDbKind
}

var _ sources.SourceConfig = KuzuDbConfig{}

type KuzuDbSource struct {
	Kind string `yaml:"kind" validated:"required"`
}

var _ sources.Source = KuzuDbConfig{}
