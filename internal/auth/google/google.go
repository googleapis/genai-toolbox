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

const AuthServiceKind string = "google"

// validate interface
var _ auth.AuthServiceConfig = Config{}

// Auth service configuration
type Config struct {
	Name     string `yaml:"name" validate:"required"`
	Kind     string `yaml:"kind" validate:"required"`
	ClientID string `yaml:"clientId" validate:"required"`
}

// Returns the auth service kind
func (cfg Config) AuthServiceConfigKind() string {
	return AuthServiceKind
}

// Initialize a Google auth service
func (cfg Config) Initialize() (auth.AuthService, error) {
	a := &AuthService{
		Name:     cfg.Name,
		Kind:     AuthServiceKind,
		ClientID: cfg.ClientID,
	}
	return a, nil
}

var _ auth.AuthService = AuthService{}

// struct used to store auth service info
type AuthService struct {
	Name     string `yaml:"name"`
	Kind     string `yaml:"kind"`
	ClientID string `yaml:"clientId"`
}

// Returns the auth service kind
func (a AuthService) AuthServiceKind() string {
	return AuthServiceKind
}

// Returns the name of the auth service
func (a AuthService) GetName() string {
	return a.Name
}

// Verifies Google ID token and return claims
func (a AuthService) GetClaimsFromHeader(ctx context.Context, h http.Header) (map[string]any, error) {
	if token := h.Get(a.Name + "_token"); token != "" {
		payload, err := idtoken.Validate(ctx, token, a.ClientID)
		if err != nil {
			return nil, fmt.Errorf("Google ID token verification failure: %w", err)
		}
		return payload.Claims, nil
	}
	return nil, nil
}
