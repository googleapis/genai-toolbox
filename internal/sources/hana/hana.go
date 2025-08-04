package hana

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"

	_ "github.com/SAP/go-hdb/driver"
	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"go.opentelemetry.io/otel/trace"
)

// SourceKind is the identifier used in YAML configuration files.
const SourceKind string = "hana"

// Ensure Config implements the sources.SourceConfig interface.
var _ sources.SourceConfig = Config{}

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

// Config defines the YAML schema for a SAP HANA source.
// Example:
//
//	name: my-hana
//	kind: hana
//	host: hana.private.corp
//	port: "39015"
//	user: SYSTEM
//	password: secret
//	database: HXE
//
// All fields except QueryTimeout are required.
// QueryTimeout follows the same semantics as other SQL drivers (e.g. "30s").
// When set, it is mapped to the "timeout" DSN query parameter.
//
// See: https://github.com/SAP/go-hdb#hana-cloud-connection for details on the DSN format.
//
// NOTE: The go-hdb driver automatically negotiates TLS when required (e.g. HANA Cloud).
// Additional TLS parameters can be passed via the DSN using standard go-hdb query parameters
// (e.g. TLSServerName, TLSRootCAFile, TLSInsecureSkipVerify).
// These can be appended manually via the `extraParams` field if needed in the future.
// Currently only the most common parameters are exposed for simplicity.
//
// If more advanced options are required later we can extend this struct.
// For now, mirroring the pattern used by other sources is sufficient.
//
// All string fields use string types to avoid YAML unmarshalling quirks with numbers.
// This mirrors existing source implementations.
//
// Validation is handled elsewhere using `validator` tags.
type Config struct {
	Name         string `yaml:"name" validate:"required"`
	Kind         string `yaml:"kind" validate:"required"`
	Host         string `yaml:"host" validate:"required"`
	Port         string `yaml:"port" validate:"required"`
	User         string `yaml:"user" validate:"required"`
	Password     string `yaml:"password" validate:"required"`
	Database     string `yaml:"database" validate:"required"`
	QueryTimeout string `yaml:"queryTimeout"`
}

func (c Config) SourceConfigKind() string {
	return SourceKind
}

func (c Config) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {
	db, err := initHanaConnection(ctx, tracer, c.Name, c.Host, c.Port, c.User, c.Password, c.Database, c.QueryTimeout)
	if err != nil {
		return nil, fmt.Errorf("unable to create SAP HANA connection: %w", err)
	}

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("unable to connect successfully: %w", err)
	}

	return &Source{
		Name: c.Name,
		Kind: SourceKind,
		Db:   db,
	}, nil
}

// Source wraps a *sql.DB backed by the go-hdb driver.
var _ sources.Source = &Source{}

type Source struct {
	Name string `yaml:"name"`
	Kind string `yaml:"kind"`
	Db   *sql.DB
}

func (s *Source) SourceKind() string { return SourceKind }

// HanaDB exposes the underlying *sql.DB so that tools can reuse a shared pool.
func (s *Source) HanaDB() *sql.DB { return s.Db }

// initHanaConnection creates a connection pool using the go-hdb driver.
func initHanaConnection(ctx context.Context, tracer trace.Tracer, name, host, port, user, pass, dbname, queryTimeout string) (*sql.DB, error) {
	//nolint:all // Span end handled below; ctx reassignment intentional.
	ctx, span := sources.InitConnectionSpan(ctx, tracer, SourceKind, name)
	defer span.End()

	// Construct DSN: hdb://user:pass@host:port?databaseName=<dbname>&timeout=<dur>
	dsnURL := &url.URL{
		Scheme: "hdb",
		User:   url.UserPassword(user, pass),
		Host:   fmt.Sprintf("%s:%s", host, port),
	}

	qs := url.Values{}
	if dbname != "" {
		qs.Add("databaseName", dbname)
	}
	if queryTimeout != "" {
		qs.Add("timeout", queryTimeout)
	}
	if len(qs) > 0 {
		dsnURL.RawQuery = qs.Encode()
	}

	db, err := sql.Open("hdb", dsnURL.String())
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}
	return db, nil
}
