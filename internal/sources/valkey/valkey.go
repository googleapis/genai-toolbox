// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package valkey

import (
	"context"
	"fmt"
	"log"

	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/valkey-io/valkey-go"
	"go.opentelemetry.io/otel/trace"
)

const SourceKind string = "valkey"

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
	Name         string   `yaml:"name" validate:"required"`
	Kind         string   `yaml:"kind" validate:"required"`
	Address      []string `yaml:"address" validate:"required"`
	Username     string   `yaml:"username"`
	Password     string   `yaml:"password"`
	Database     int      `yaml:"database"`
	UseGCPIAM    bool     `yaml:"useGCPIAM"`
	DisableCache bool     `yaml:"disableCache"`
}

func (r Config) SourceConfigKind() string {
	return SourceKind
}

func (r Config) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {

	client, err := initValkeyClient(ctx, r)
	if err != nil {
		return nil, fmt.Errorf("error initializing Valkey client: %s", err)
	}
	s := &Source{
		Name:   r.Name,
		Kind:   SourceKind,
		Client: client,
	}
	return s, nil
}

func initValkeyClient(ctx context.Context, r Config) (valkey.Client, error) {
	var authFn func(valkey.AuthCredentialsContext) (valkey.AuthCredentials, error)
	if r.UseGCPIAM {
		// Pass in an access token getter fn for IAM auth
		authFn = func(valkey.AuthCredentialsContext) (valkey.AuthCredentials, error) {
			token, err := sources.GetIAMAccessToken(ctx)
			creds := valkey.AuthCredentials{Username: "default", Password: token}
			if err != nil {
				return creds, err
			}
			return creds, nil
		}
	}

	client, err := valkey.NewClient(valkey.ClientOption{
		InitAddress:       r.Address,
		SelectDB:          r.Database,
		Username:          r.Username,
		Password:          r.Password,
		AuthCredentialsFn: authFn,
		DisableCache:      r.DisableCache,
	})

	if err != nil {
		log.Fatalf("error creating Valkey client: %v", err)
	}

	// Ping the server to check connectivity
	pingCmd := client.B().Ping().Build()
	_, err = client.Do(ctx, pingCmd).ToString()
	if err != nil {
		log.Fatalf("Failed to execute PING command: %v", err)
	}
	return client, nil
}

var _ sources.Source = &Source{}

type Source struct {
	Name   string `yaml:"name"`
	Kind   string `yaml:"kind"`
	Client valkey.Client
}

func (s *Source) SourceKind() string {
	return SourceKind
}

func (s *Source) ValkeyClient() valkey.Client {
	return s.Client
}
