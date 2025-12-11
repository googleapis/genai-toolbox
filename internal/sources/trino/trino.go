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

package trino

import (
	"context"
	"crypto/tls"
	"database/sql"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	trinogo "github.com/trinodb/trino-go-client/trino"
	"go.opentelemetry.io/otel/trace"
)

const SourceKind string = "trino"

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
	return actual, nil
}

type Config struct {
	Name                   string `yaml:"name" validate:"required"`
	Kind                   string `yaml:"kind" validate:"required"`
	Host                   string `yaml:"host" validate:"required"`
	Port                   string `yaml:"port" validate:"required"`
	User                   string `yaml:"user"`
	Password               string `yaml:"password"`
	Catalog                string `yaml:"catalog" validate:"required"`
	Schema                 string `yaml:"schema" validate:"required"`
	QueryTimeout           string `yaml:"queryTimeout"`
	AccessToken            string `yaml:"accessToken"`
	KerberosEnabled        bool   `yaml:"kerberosEnabled"`
	SSLEnabled             bool   `yaml:"sslEnabled"`
	SSLCertPath            string `yaml:"sslCertPath"`
	SSLCert                string `yaml:"sslCert"`
	DisableSslVerification bool   `yaml:"disableSslVerification"`
}

func (r Config) SourceConfigKind() string {
	return SourceKind
}

func (r Config) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {
	pool, err := initTrinoConnectionPool(ctx, tracer, r.Name, r.Host, r.Port, r.User, r.Password, r.Catalog, r.Schema, r.QueryTimeout, r.AccessToken, r.KerberosEnabled, r.SSLEnabled, r.SSLCertPath, r.SSLCert, r.DisableSslVerification)
	if err != nil {
		return nil, fmt.Errorf("unable to create pool: %w", err)
	}

	err = pool.PingContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to connect successfully: %w", err)
	}

	s := &Source{
		Config: r,
		Pool:   pool,
	}
	return s, nil
}

var _ sources.Source = &Source{}

type Source struct {
	Config
	Pool *sql.DB
}

func (s *Source) SourceKind() string {
	return SourceKind
}

func (s *Source) ToConfig() sources.SourceConfig {
	return s.Config
}

func (s *Source) TrinoDB() *sql.DB {
	return s.Pool
}

func initTrinoConnectionPool(ctx context.Context, tracer trace.Tracer, name, host, port, user, password, catalog, schema, queryTimeout, accessToken string, kerberosEnabled, sslEnabled bool, sslCertPath, sslCert string, disableSslVerification bool) (*sql.DB, error) {
	//nolint:all // Reassigned ctx
	ctx, span := sources.InitConnectionSpan(ctx, tracer, SourceKind, name)
	defer span.End()

	// Build Trino DSN
	dsn, err := buildTrinoDSN(host, port, user, password, catalog, schema, queryTimeout, accessToken, kerberosEnabled, sslEnabled, sslCertPath, sslCert)
	if err != nil {
		return nil, fmt.Errorf("failed to build DSN: %w", err)
	}

	if disableSslVerification {
		fmt.Printf("WARNING: SSL verification is disabled for trino source %q. This is an insecure setting and should not be used in production.", name)
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		client := &http.Client{Transport: tr}
		clientName := fmt.Sprintf("insecure_trino_client_%s", name)
		if err := trinogo.RegisterCustomClient(clientName, client); err != nil {
			return nil, fmt.Errorf("failed to register custom client: %w", err)
		}
		dsn = fmt.Sprintf("%s&custom_client=%s", dsn, clientName)
	}

	db, err := sql.Open("trino", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open connection: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	return db, nil
}

func buildTrinoDSN(host, port, user, password, catalog, schema, queryTimeout, accessToken string, kerberosEnabled, sslEnabled bool, sslCertPath, sslCert string) (string, error) {
	// Build query parameters
	query := url.Values{}
	query.Set("catalog", catalog)
	query.Set("schema", schema)
	if queryTimeout != "" {
		query.Set("queryTimeout", queryTimeout)
	}
	if accessToken != "" {
		query.Set("accessToken", accessToken)
	}
	if kerberosEnabled {
		query.Set("KerberosEnabled", "true")
	}
	if sslCertPath != "" {
		query.Set("sslCertPath", sslCertPath)
	}
	if sslCert != "" {
		query.Set("sslCert", sslCert)
	}

	// Build URL
	scheme := "http"
	if sslEnabled {
		scheme = "https"
	}

	u := &url.URL{
		Scheme:   scheme,
		Host:     fmt.Sprintf("%s:%s", host, port),
		RawQuery: query.Encode(),
	}

	// Only set user and password if not empty
	if user != "" && password != "" {
		u.User = url.UserPassword(user, password)
	} else if user != "" {
		u.User = url.User(user)
	}

	return u.String(), nil
}
