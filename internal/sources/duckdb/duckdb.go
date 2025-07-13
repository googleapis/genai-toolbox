package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	_ "github.com/marcboeker/go-duckdb"
	"go.opentelemetry.io/otel/trace"
)

const SourceKind string = "duckdb"

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

type DuckDbSource struct {
	Name string `yaml:"name"`
	Kind string `yaml:"kind"`
	Db   *sql.DB
}

// SourceKind implements sources.Source.
func (s *DuckDbSource) SourceKind() string {
	return SourceKind
}

func (s *DuckDbSource) DuckDb() *sql.DB {
	return s.Db
}

// validate Source
var _ sources.Source = &DuckDbSource{}

type Config struct {
	Name           string            `yaml:"name" validate:"required"`
	Kind           string            `yaml:"kind" validate:"required"`
	DatabaseFile   string            `yaml:"dbFilePath,omitempty"`
	Configurations map[string]string `yaml:"configurations,omitempty"`
}

func (r Config) SourceConfigKind() string {
	return SourceKind
}

func (r Config) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {
	db, err := initDuckDbConnection(ctx, tracer, r.Name, r.DatabaseFile, r.Configurations)
	if err != nil {
		return nil, fmt.Errorf("unable to create db connection: %w", err)
	}

	err = db.PingContext(context.Background())
	if err != nil {
		return nil, fmt.Errorf("unable to connect sucessfully: %w", err)
	}

	s := &DuckDbSource{
		Name: r.Name,
		Kind: r.Kind,
		Db:   db,
	}
	return s, nil
}

// validate interface
var _ sources.SourceConfig = Config{}

func initDuckDbConnection(ctx context.Context, tracer trace.Tracer, name string, dbFilePath string, duckdbConfiguration map[string]string) (*sql.DB, error) {
	//nolint:all // Reassigned ctx
	ctx, span := sources.InitConnectionSpan(ctx, tracer, SourceKind, name)
	defer span.End()

	var configStr string = getDuckDbConfiguration(dbFilePath, duckdbConfiguration)

	//Open database connection
	db, err := sql.Open("duckdb", configStr)
	if err != nil {
		return nil, fmt.Errorf("unable to open duckdb connection: %w", err)
	}
	return db, nil
}

func getDuckDbConfiguration(dbFilePath string, duckdbConfiguration map[string]string) string {
	if dbFilePath == "" && len(duckdbConfiguration) == 0 {
		return ""
	}
	var configStr strings.Builder
	if dbFilePath != "" {
		configStr.WriteString(dbFilePath)
	}
	configStr.WriteString("?")
	first := true
	for key, value := range duckdbConfiguration {
		if !first {
			configStr.WriteString("&")
		}
		configStr.WriteString(url.QueryEscape(key))
		configStr.WriteString("=")
		configStr.WriteString(url.QueryEscape(value))
		first = false
	}
	return configStr.String()
}
