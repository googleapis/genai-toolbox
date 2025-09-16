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
package opengemini

import (
	"context"
	"fmt"

	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/openGemini/opengemini-client-go/opengemini"
	"go.opentelemetry.io/otel/trace"
)

const SourceKind string = "opengemini"

const (
	AuthTypePwd = iota + 1
	AuthTypeToken
)

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
	Name            string `yaml:"name" validate:"required"`
	Kind            string `yaml:"kind" validate:"required"`
	Host            string `yaml:"host" validate:"required"`
	Port            int    `yaml:"port" validate:"required"`
	Database        string `yaml:"database" validate:"required"`
	RetentionPolicy string `yaml:"retentionpolicy" validate:"required"`
	AuthType        int    `yaml:"authtype"`
	User            string `yaml:"user"`
	Password        string `yaml:"password"`
	Token           string `yaml:"token"`
}

func (r Config) SourceConfigKind() string {
	return SourceKind
}

func (r Config) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {
	client, err := initOpenGeminiClient(ctx, r)
	if err != nil {
		return nil, fmt.Errorf("error initializing opengemini client: %s", err)
	}

	s := &Source{
		Name:            r.Name,
		Kind:            SourceKind,
		Client:          client,
		Database:        r.Database,
		RetentionPolicy: r.RetentionPolicy,
	}
	return s, nil
}

var _ sources.Source = &Source{}

type Source struct {
	Name            string `yaml:"name"`
	Kind            string `yaml:"kind"`
	Client          opengemini.Client
	Database        string
	RetentionPolicy string
}

func (s *Source) SourceKind() string {
	return SourceKind
}

func (s *Source) OpenGeminiClient() opengemini.Client {
	return s.Client
}

func (s *Source) GetDatabase() string {
	return s.Database
}

func (s *Source) GetRetentionPolicy() string {
	return s.RetentionPolicy
}

func initOpenGeminiClient(ctx context.Context, r Config) (opengemini.Client, error) {

	var authConfig *opengemini.AuthConfig
	if r.AuthType == AuthTypePwd || len(r.User) != 0 {
		authConfig = &opengemini.AuthConfig{
			AuthType: opengemini.AuthTypePassword,
			Username: r.User,
			Password: r.Password,
		}
	} else if r.AuthType == AuthTypeToken || len(r.Token) != 0 {
		authConfig = &opengemini.AuthConfig{
			AuthType: opengemini.AuthTypeToken,
			Token:    r.Token,
		}
	}

	config := &opengemini.Config{
		Addresses: []opengemini.Address{
			{
				Host: r.Host,
				Port: r.Port,
			},
		},
		AuthConfig: authConfig,
	}

	client, err := opengemini.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to opengemini client: %w", err)
	}

	if err = client.Ping(0); err != nil {
		return nil, fmt.Errorf("unable to connect to opengemini: %s", err)
	}

	return client, nil
}
