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

func getOpts(ipType, userAgent string) ([]alloydbconn.Option, error) {
	opts := []alloydbconn.Option{alloydbconn.WithUserAgent(userAgent)}
	switch strings.ToLower(ipType) {
	case "private":
		opts = append(opts, alloydbconn.WithDefaultDialOptions(alloydbconn.WithPrivateIP()))
	case "public":
		opts = append(opts, alloydbconn.WithDefaultDialOptions(alloydbconn.WithPublicIP()))
	default:
		return nil, fmt.Errorf("invalid ipType %s", ipType)
	}
	return opts, nil
}

// getIAMPrincipalEmailFromADC finds the email associated with Application Default Credentials.
func getIAMPrincipalEmailFromADC(ctx context.Context) (string, error) {
	client, err := google.DefaultClient(ctx,
		"https://www.googleapis.com/auth/userinfo.email")
	if err != nil {
		return "", fmt.Errorf("failed to call userinfo endpoint: %w", err)
	}

	// Call the userinfo endpoint
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return "", fmt.Errorf("failed to call userinfo endpoint: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("userinfo endpoint returned non-OK status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse the response
	var userInfo struct {
		Email         string `json:"email"`
		VerifiedEmail bool   `json:"verified_email"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return "", fmt.Errorf("failed to decode userinfo response: %w", err)
	}

	if userInfo.Email == "" {
		return "", fmt.Errorf("userinfo response did not contain an email address")
	}

	return userInfo.Email, nil
}

func initAlloyDBPgConnectionPool(ctx context.Context, tracer trace.Tracer, name, project, region, cluster, instance, ipType, user, pass, dbname string) (*pgxpool.Pool, error) {
	//nolint:all // Reassigned ctx
	ctx, span := sources.InitConnectionSpan(ctx, tracer, SourceKind, name)
	defer span.End()

	var dsn string
	var err error
	if user == "" {
		user, err = getIAMPrincipalEmailFromADC(ctx)
		if err != nil {
			return nil, fmt.Errorf("IAM user was not provided and could not be discovered from ADC: %w", err)
		}
	}
	if pass == "" {
		// Use IAM authentication for db connectionif no password provided
		dsn = fmt.Sprintf("user=%s dbname=%s sslmode=disable", user, dbname)

	} else {
		// Use username/password for db connection
		dsn = fmt.Sprintf("user=%s password=%s dbname=%s sslmode=disable", user, pass, dbname)

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
	opts, err := getOpts(ipType, userAgent)
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
