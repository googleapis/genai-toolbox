package googleAuth

import (
	"github.com/googleapis/genai-toolbox/internal/authSources"
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

func (r AuthSource) Verify() {
}
