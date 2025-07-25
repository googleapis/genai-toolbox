package cassandra

import (
	"context"
	"fmt"
	"log"

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

// TODO: add additonal configurations
type Config struct {
	Name  string   `yaml:"name" validate:"required"`
	Kind  string   `yaml:"kind" validate:"required"`
	Hosts []string `yaml:"host" validate:"required"`
}

// Initialize implements sources.SourceConfig.
func (c Config) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {
	cluster := gocql.NewCluster("localhost")

	// Create session
	session, err := cluster.CreateSession()
	if err != nil {
		log.Fatalf("Failed to create session: %v", err)
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
