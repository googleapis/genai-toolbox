// Copyright 2024 Google LLC
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

package alloydbpg

import (
	"context"
	"fmt"
	"net"
	"strings"

	"cloud.google.com/go/alloydbconn"
	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/util"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/trace"
)

const SourceType string = "alloydb-postgres"

// validate interface
var _ sources.SourceConfig = Config{}

func init() {
	if !sources.Register(SourceType, newConfig) {
		panic(fmt.Sprintf("source type %q already registered", SourceType))
	}
}

func newConfig(ctx context.Context, name string, decoder *yaml.Decoder) (sources.SourceConfig, error) {
	actual := Config{Name: name, IPType: "public"} // Default IPType
	if err := decoder.DecodeContext(ctx, &actual); err != nil {
		return nil, err
	}
	return actual, nil
}

type Config struct {
	Name     string         `yaml:"name" validate:"required"`
	Type     string         `yaml:"type" validate:"required"`
	Project  string         `yaml:"project" validate:"required"`
	Region   string         `yaml:"region" validate:"required"`
	Cluster  string         `yaml:"cluster" validate:"required"`
	Instance string         `yaml:"instance" validate:"required"`
	IPType   sources.IPType `yaml:"ipType" validate:"required"`
	User     string         `yaml:"user"`
	Password string         `yaml:"password"`
	Database string         `yaml:"database" validate:"required"`
}

func (r Config) SourceConfigType() string {
	return SourceType
}

func (r Config) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {
	pool, err := initAlloyDBPgConnectionPool(ctx, tracer, r.Name, r.Project, r.Region, r.Cluster, r.Instance, r.IPType.String(), r.User, r.Password, r.Database)
	if err != nil {
		return nil, fmt.Errorf("unable to create pool: %w", err)
	}

	err = pool.Ping(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to connect successfully: %w", err)
	}

	s := &Source{
		Name: r.Name,
		Type: SourceType,
		Pool: pool,
	}
	return s, nil
}

var _ sources.Source = &Source{}

type Source struct {
	Name string `yaml:"name"`
	Type string `yaml:"type"`
	Pool *pgxpool.Pool
}

func (s *Source) SourceType() string {
	return SourceType
}

func (s *Source) PostgresPool() *pgxpool.Pool {
	return s.Pool
}

func getOpts(ipType, userAgent string, useIAM bool) ([]alloydbconn.Option, error) {
	opts := []alloydbconn.Option{alloydbconn.WithUserAgent(userAgent)}
	switch strings.ToLower(ipType) {
	case "private":
		opts = append(opts, alloydbconn.WithDefaultDialOptions(alloydbconn.WithPrivateIP()))
	case "public":
		opts = append(opts, alloydbconn.WithDefaultDialOptions(alloydbconn.WithPublicIP()))
	default:
		return nil, fmt.Errorf("invalid ipType %s", ipType)
	}

	if useIAM {
		opts = append(opts, alloydbconn.WithIAMAuthN())
	}
	return opts, nil
}

func getConnectionConfig(ctx context.Context, user, pass, dbname string) (string, bool, error) {
	useIAM := true

	// If username and password both provided, use password authentication
	if user != "" && pass != "" {
		dsn := fmt.Sprintf("user=%s password=%s dbname=%s sslmode=disable", user, pass, dbname)
		useIAM = false
		return dsn, useIAM, nil
	}

	// If username is empty, fetch email from ADC
	// otherwise, use username as IAM email
	if user == "" {
		if pass != "" {
			// If password is provided without an username, raise an error
			return "", useIAM, fmt.Errorf("password is provided without a username. Please provide both a username and password, or leave both fields empty")
		}
		email, err := sources.GetIAMPrincipalEmailFromADC(ctx)
		if err != nil {
			return "", useIAM, fmt.Errorf("error getting email from ADC: %v", err)
		}
		user = email
	}

	// Construct IAM connection string with username
	dsn := fmt.Sprintf("user=%s dbname=%s sslmode=disable", user, dbname)
	return dsn, useIAM, nil
}

func initAlloyDBPgConnectionPool(ctx context.Context, tracer trace.Tracer, name, project, region, cluster, instance, ipType, user, pass, dbname string) (*pgxpool.Pool, error) {
	//nolint:all // Reassigned ctx
	ctx, span := sources.InitConnectionSpan(ctx, tracer, SourceType, name)
	defer span.End()

	dsn, useIAM, err := getConnectionConfig(ctx, user, pass, dbname)
	if err != nil {
		return nil, fmt.Errorf("unable to get AlloyDB connection config: %w", err)
	}

	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("unable to parse connection uri: %w", err)
	}
	// Create a new dialer with options
	userAgent, err := util.UserAgentFromContext(ctx)
	if err != nil {
		return nil, err
	}
	opts, err := getOpts(ipType, userAgent, useIAM)
	if err != nil {
		return nil, err
	}
	d, err := alloydbconn.NewDialer(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("unable to parse connection uri: %w", err)
	}

	// Tell the driver to use the AlloyDB Go Connector to create connections
	i := fmt.Sprintf("projects/%s/locations/%s/clusters/%s/instances/%s", project, region, cluster, instance)
	config.ConnConfig.DialFunc = func(ctx context.Context, _ string, instance string) (net.Conn, error) {
		return d.Dial(ctx, i)
	}

	// Interact with the driver directly as you normally would
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, err
	}
	return pool, nil
}
