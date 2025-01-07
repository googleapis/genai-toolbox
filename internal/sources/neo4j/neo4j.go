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

package neo4j

import (
	"context"
	"fmt"

	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

const SourceKind string = "neo4j"

// validate interface
var _ sources.SourceConfig = Config{}

type Config struct {
	Name     string `yaml:"name"`
	Kind     string `yaml:"kind"`
	Proto    string `yaml:"proto"`
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
}

func (r Config) SourceConfigKind() string {
	return SourceKind
}

func (r Config) Initialize() (sources.Source, error) {
	driver, err := initNeo4jDriver(r.Proto, r.Host, r.Port, r.User, r.Password)
	if err != nil {
		return nil, fmt.Errorf("Unable to create driver: %w", err)
	}

	err = driver.VerifyConnectivity(context.Background())
	if err != nil {
		return nil, fmt.Errorf("Unable to connect successfully: %w", err)
	}

	if r.Database == "" {
		r.Database = "neo4j"
	}
	s := &Source{
		Name:     r.Name,
		Kind:     SourceKind,
		Database: r.Database,
		Driver:   driver,
	}
	return s, nil
}

var _ sources.Source = &Source{}

type Source struct {
	Name     string `yaml:"name"`
	Kind     string `yaml:"kind"`
	Database string `yaml:"database"`
	Driver   neo4j.DriverWithContext
}

func (s *Source) SourceKind() string {
	return SourceKind
}

func (s *Source) Neo4jDriver() neo4j.DriverWithContext {
	return s.Driver
}

func (s *Source) Neo4jDatabase() string {
	return s.Database
}

func initNeo4jDriver(proto, host, port, user, password string) (neo4j.DriverWithContext, error) {
	// urlExample := "neo4j+s://localhost:7687"
	url := fmt.Sprintf("%s://%s:%s", proto, host, port)
	auth := neo4j.BasicAuth(user, password, "")
	driver, err := neo4j.NewDriverWithContext(url, auth)
	if err != nil {
		return nil, fmt.Errorf("Unable to create connection driver: %w", err)
	}
	return driver, nil
}
