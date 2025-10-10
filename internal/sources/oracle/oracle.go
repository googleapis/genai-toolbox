// Copyright © 2025, Oracle and/or its affiliates.
package oracle

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"

	"github.com/goccy/go-yaml"
	_ "github.com/godror/godror"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/util"
	"go.opentelemetry.io/otel/trace"
)

const SourceKind string = "oracle"

// validate interface
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

	// Validate that we have one of: tns_alias, connection_string, or host+service_name
	if err := actual.validate(); err != nil {
		return nil, fmt.Errorf("invalid Oracle configuration: %w", err)
	}

	return actual, nil
}

type Config struct {
	Name             string `yaml:"name" validate:"required"`
	Kind             string `yaml:"kind" validate:"required"`
	ConnectionString string `yaml:"connectionString,omitempty"` // Direct connection string (hostname[:port]/servicename)
	TnsAlias         string `yaml:"tnsAlias,omitempty"`         // TNS alias from tnsnames.ora
	Host             string `yaml:"host,omitempty"`             // Optional when using connectionString/tnsAlias
	Port             int    `yaml:"port,omitempty"`             // Explicit port support
	ServiceName      string `yaml:"serviceName,omitempty"`      // Optional when using connectionString/tnsAlias
	User             string `yaml:"user" validate:"required"`
	Password         string `yaml:"password" validate:"required"`
	TnsAdmin         string `yaml:"tnsAdmin,omitempty"` // Optional: override TNS_ADMIN environment variable
}

// validate ensures we have one of: tns_alias, connection_string, or host+service_name
func (c Config) validate() error {
	hasTnsAlias := strings.TrimSpace(c.TnsAlias) != ""
	hasConnStr := strings.TrimSpace(c.ConnectionString) != ""
	hasHostService := strings.TrimSpace(c.Host) != "" && strings.TrimSpace(c.ServiceName) != ""

	connectionMethods := 0
	if hasTnsAlias {
		connectionMethods++
	}
	if hasConnStr {
		connectionMethods++
	}
	if hasHostService {
		connectionMethods++
	}

	if connectionMethods == 0 {
		return fmt.Errorf("must provide one of: 'tns_alias', 'connection_string', or both 'host' and 'service_name'")
	}

	if connectionMethods > 1 {
		return fmt.Errorf("provide only one connection method: 'tns_alias', 'connection_string', or 'host'+'service_name'")
	}

	return nil
}

func (r Config) SourceConfigKind() string {
	return SourceKind
}

func (r Config) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {
	db, err := initOracleConnection(ctx, tracer, r)
	if err != nil {
		return nil, fmt.Errorf("unable to create Oracle connection: %w", err)
	}

	err = db.PingContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to Oracle successfully: %w", err)
	}

	s := &Source{
		Name: r.Name,
		Kind: SourceKind,
		DB:   db,
	}
	return s, nil
}

var _ sources.Source = &Source{}

type Source struct {
	Name string `yaml:"name"`
	Kind string `yaml:"kind"`
	DB   *sql.DB
}

func (s *Source) SourceKind() string {
	return SourceKind
}

func (s *Source) OracleDB() *sql.DB {
	return s.DB
}

func initOracleConnection(ctx context.Context, tracer trace.Tracer, config Config) (*sql.DB, error) {
	//nolint:all // Reassigned ctx
	ctx, span := sources.InitConnectionSpan(ctx, tracer, SourceKind, config.Name)
	defer span.End()

	var connectString string
	logger, err := util.LoggerFromContext(ctx)
	if err != nil {
		panic(err)
	}

	// Set TNS_ADMIN environment variable if specified in config
	if config.TnsAdmin != "" {
		originalTnsAdmin := os.Getenv("TNS_ADMIN")
		os.Setenv("TNS_ADMIN", config.TnsAdmin)
		logger.DebugContext(ctx, fmt.Sprintf("Setting TNS_ADMIN to: %s\n", config.TnsAdmin))
		// Restore original TNS_ADMIN after connection
		defer func() {
			if originalTnsAdmin != "" {
				os.Setenv("TNS_ADMIN", originalTnsAdmin)
			} else {
				os.Unsetenv("TNS_ADMIN")
			}
		}()
	}

	// Determine the connection string to use (priority order)
	if config.TnsAlias != "" {
		// Use TNS alias - godror will resolve from tnsnames.ora
		connectString = strings.TrimSpace(config.TnsAlias)
	} else if config.ConnectionString != "" {
		// Use provided connection string directly (hostname[:port]/servicename format)
		connectString = strings.TrimSpace(config.ConnectionString)
	} else {
		// Build connection string from host and service_name
		if config.Port > 0 {
			connectString = fmt.Sprintf("%s:%d/%s", config.Host, config.Port, config.ServiceName)
		} else {
			connectString = fmt.Sprintf("%s/%s", config.Host, config.ServiceName)
		}
	}

	// Build the full Oracle connection string for godror driver
	connStr := fmt.Sprintf(`user="%s" password="%s" connectString="%s"`,
		config.User, config.Password, connectString)

	db, err := sql.Open("godror", connStr)
	if err != nil {
		return nil, fmt.Errorf("unable to open Oracle connection: %w", err)
	}

	return db, nil
}
