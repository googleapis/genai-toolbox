// Copyright 2024 Google LLC
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

package google

import (
	"context"
	"fmt"

	authSources "github.com/googleapis/genai-toolbox/internal/auth"
	"google.golang.org/api/idtoken"
)

const AuthSourceKind string = "google"

// validate interface
var _ authSources.AuthSourceConfig = Config{}

type Config struct {
	Name     string `yaml:"name"`
	Kind     string `yaml:"kind"`
	ClientID string `yaml:"client_id"`
}

func (cfg Config) AuthSourceConfigKind() string {
	return AuthSourceKind
}

func (cfg Config) Initialize() (authSources.AuthSource, error) {
	a := &AuthSource{
		Name:     cfg.Name,
		Kind:     AuthSourceKind,
		ClientID: cfg.ClientID,
	}
	return a, nil
}

var _ authSources.AuthSource = AuthSource{}

type AuthSource struct {
	Name     string `yaml:"name"`
	Kind     string `yaml:"kind"`
	ClientID string `yaml:"client_id"`
}

func (a AuthSource) AuthSourceKind() string {
	return AuthSourceKind
}

func (a AuthSource) GetName() string {
	return a.Name
}

func (a AuthSource) Verify(token string) (map[string]interface{}, error) {
	payload, err := idtoken.Validate(context.Background(), token, a.ClientID)
	if err != nil {
		return nil, fmt.Errorf("Google ID token verification failure: %w", err)
	}
	return payload.Claims, nil
}
