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
	"net/http"

	"github.com/googleapis/genai-toolbox/internal/auth"
	"google.golang.org/api/idtoken"
)

const AuthSourceKind string = "google"

// validate interface
var _ auth.AuthSourceConfig = Config{}

// Auth source configuration
type Config struct {
	Name     string `yaml:"name"`
	Kind     string `yaml:"kind"`
	ClientID string `yaml:"clientId"`
}

// Returns the auth source kind
func (cfg Config) AuthSourceConfigKind() string {
	return AuthSourceKind
}

// Initialize a Google auth source
func (cfg Config) Initialize() (auth.AuthSource, error) {
	a := &AuthSource{
		Name:     cfg.Name,
		Kind:     AuthSourceKind,
		ClientID: cfg.ClientID,
	}
	return a, nil
}

var _ auth.AuthSource = AuthSource{}

// struct used to store auth source info
type AuthSource struct {
	Name     string `yaml:"name"`
	Kind     string `yaml:"kind"`
	ClientID string `yaml:"clientId"`
}

// Returns the auth source kind
func (a AuthSource) AuthSourceKind() string {
	return AuthSourceKind
}

// Returns the name of the auth source
func (a AuthSource) GetName() string {
	return a.Name
}

// Verifies Google ID token and return claims
func (a AuthSource) GetClaimsFromHeader(h http.Header) (map[string]any, error) {
	if token := h.Get(a.Name + "_token"); token != "" {
		payload, err := idtoken.Validate(context.Background(), token, a.ClientID)
		if err != nil {
			return nil, fmt.Errorf("Google ID token verification failure: %w", err)
		}
		return payload.Claims, nil
	}
	return nil, nil
}
