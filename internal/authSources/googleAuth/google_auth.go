package googleAuth

import (
	"context"
	"fmt"

	"github.com/googleapis/genai-toolbox/internal/authSources"
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
