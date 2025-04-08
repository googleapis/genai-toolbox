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
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"

	"cloud.google.com/go/alloydbconn"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/util"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/oauth2/google"
)

const SourceKind string = "alloydb-postgres"

// validate interface
var _ sources.SourceConfig = Config{}

type Config struct {
	Name     string         `yaml:"name" validate:"required"`
	Kind     string         `yaml:"kind" validate:"required"`
	Project  string         `yaml:"project" validate:"required"`
	Region   string         `yaml:"region" validate:"required"`
	Cluster  string         `yaml:"cluster" validate:"required"`
	Instance string         `yaml:"instance" validate:"required"`
	IPType   sources.IPType `yaml:"ipType" validate:"required"`
	User     string         `yaml:"user"`
	Password string         `yaml:"password"`
	Database string         `yaml:"database" validate:"required"`
}

func (r Config) SourceConfigKind() string {
	return SourceKind
}

func (r Config) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {
	pool, err := initAlloyDBPgConnectionPool(ctx, tracer, r.Name, r.Project, r.Region, r.Cluster, r.Instance, r.IPType.String(), r.User, r.Password, r.Database)
	if err != nil {
		return nil, fmt.Errorf("unable to create pool: %w", err)
	}

	err = pool.Ping(context.Background())
	if err != nil {
		return nil, fmt.Errorf("unable to connect successfully: %w", err)
	}

	s := &Source{
		Name: r.Name,
		Kind: SourceKind,
		Pool: pool,
	}
	return s, nil
}

var _ sources.Source = &Source{}

type Source struct {
	Name string `yaml:"name"`
	Kind string `yaml:"kind"`
	Pool *pgxpool.Pool
}

func (s *Source) SourceKind() string {
	return SourceKind
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

// getIAMPrincipalEmailFromADC finds the email associated with ADC
func getIAMPrincipalEmailFromADC(ctx context.Context) (string, error) {
	// Finds ADC and returns an HTTP client associated with it
	client, err := google.DefaultClient(ctx,
		"https://www.googleapis.com/auth/userinfo.email")
	if err != nil {
		return "", fmt.Errorf("failed to call userinfo endpoint: %w", err)
	}

	// Retrieve the email associated with the token
	resp, err := client.Get("https://oauth2.googleapis.com/tokeninfo")
	if err != nil {
		return "", fmt.Errorf("failed to call tokeninfo endpoint: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body %d: %s", resp.StatusCode, string(bodyBytes))
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("tokeninfo endpoint returned non-OK status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Unmarshal response body and get `email`
	var responseJSON map[string]any
	err = json.Unmarshal(bodyBytes, &responseJSON)
	if err != nil {

		return "", fmt.Errorf("error parsing JSON: %v", err)
	}

	emailValue, ok := responseJSON["email"]
	if !ok {
		return "", fmt.Errorf("email not found in response: %v", err)
	}
	// service account email used for IAM should trim the suffix
	email := strings.TrimSuffix(emailValue.(string), ".gserviceaccount.com")
	return email, nil
}

func getConnectionConfig(ctx context.Context, user, pass, dbname string) (string, bool, error) {
	useIAM := true

	// username and password both provided, use password authentication
	if user != "" && pass != "" {
		dsn := fmt.Sprintf("user=%s password=%s dbname=%s sslmode=disable", user, pass, dbname)
		useIAM = false
		return dsn, useIAM, nil
	}

	// Use IAM authentication if username is not provided
	if user == "" {
		if pass != "" {
			// If password is provided without an username, raise an error
			return "", useIAM, fmt.Errorf("password is provided without a username. Please provide both a username and password, or leave both fields empty")
		}
		email, err := getIAMPrincipalEmailFromADC(ctx)
		if err != nil {
			return "", useIAM, fmt.Errorf("error getting email from ADC: %v", err)
		}
		user = email
	}

	// Construct IAM db connection string
	dsn := fmt.Sprintf("user=%s dbname=%s sslmode=disable", user, dbname)
	return dsn, useIAM, nil
}

func initAlloyDBPgConnectionPool(ctx context.Context, tracer trace.Tracer, name, project, region, cluster, instance, ipType, user, pass, dbname string) (*pgxpool.Pool, error) {
	//nolint:all // Reassigned ctx
	ctx, span := sources.InitConnectionSpan(ctx, tracer, SourceKind, name)
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
	d, err := alloydbconn.NewDialer(context.Background(), opts...)
	if err != nil {
		return nil, fmt.Errorf("unable to parse connection uri: %w", err)
	}

	// Tell the driver to use the AlloyDB Go Connector to create connections
	i := fmt.Sprintf("projects/%s/locations/%s/clusters/%s/instances/%s", project, region, cluster, instance)
	config.ConnConfig.DialFunc = func(ctx context.Context, _ string, instance string) (net.Conn, error) {
		return d.Dial(ctx, i)
	}

	// Interact with the driver directly as you normally would
	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, err
	}
	return pool, nil
}
