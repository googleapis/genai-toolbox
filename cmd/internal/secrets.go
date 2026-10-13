// Copyright 2026 Google LLC
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

package internal

import (
	"context"
	"fmt"
	"hash/crc32"
	"os"
	"regexp"
	"strconv"
	"strings"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/googleapis/gax-go/v2"
	"github.com/googleapis/mcp-toolbox/internal/util"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
)

const secretManagerCloudPlatformScope = "https://www.googleapis.com/auth/cloud-platform"

var (
	envVarNameRegex     = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
	shortSecretRefRegex = regexp.MustCompile(`^([a-zA-Z0-9_-]+)(?::([a-zA-Z0-9_-]+))?$`)
	globalSecretRegex   = regexp.MustCompile(`^projects/([^/]+)/secrets/([^/]+)/versions/([^/]+)$`)
	regionalSecretRegex = regexp.MustCompile(`^projects/([^/]+)/locations/([^/]+)/secrets/([^/]+)/versions/([^/]+)$`)
)

// SecretManagerClient abstracts the Google Cloud Secret Manager client for testing.
type SecretManagerClient interface {
	AccessSecretVersion(ctx context.Context, req *secretmanagerpb.AccessSecretVersionRequest, opts ...gax.CallOption) (*secretmanagerpb.AccessSecretVersionResponse, error)
	Close() error
}

// newSecretManagerClient creates a Global or Regional Secret Manager client.
// When location is non-empty, requests are routed to the regional endpoint
// (secretmanager.<location>.rep.googleapis.com:443).
var newSecretManagerClient = func(ctx context.Context, location string, opts ...option.ClientOption) (SecretManagerClient, error) {
	if location != "" {
		endpoint := fmt.Sprintf("secretmanager.%s.rep.googleapis.com:443", location)
		opts = append(opts, option.WithEndpoint(endpoint))
	}
	return secretmanager.NewClient(ctx, opts...)
}

// SetSecretManagerClientFactoryForTest overrides the Secret Manager client factory in tests
// and returns a restore function.
func SetSecretManagerClientFactoryForTest(fn func(ctx context.Context, location string, opts ...option.ClientOption) (SecretManagerClient, error)) func() {
	orig := newSecretManagerClient
	newSecretManagerClient = fn
	return func() {
		newSecretManagerClient = orig
	}
}

// findDefaultCredentials looks up Application Default Credentials (ADC).
var findDefaultCredentials = google.FindDefaultCredentials

// SecretMapping represents a parsed --set-secrets entry (ENV_VAR=SECRET_REF).
type SecretMapping struct {
	EnvVar    string
	SecretRef string
	Project   string
	Location  string // empty for Global Secret Manager, non-empty for Regional Secret Manager
	SecretID  string
	Version   string
	IsAlias   bool // true if Version is a version alias (e.g. "latest"), false if static numeric version
}

// ResourceName returns the full Secret Manager secret version resource path.
func (m SecretMapping) ResourceName() string {
	if m.Location != "" {
		return fmt.Sprintf("projects/%s/locations/%s/secrets/%s/versions/%s", m.Project, m.Location, m.SecretID, m.Version)
	}
	return fmt.Sprintf("projects/%s/secrets/%s/versions/%s", m.Project, m.SecretID, m.Version)
}

// isVersionAlias returns true when version is a version alias (such as "latest"
// or a user-defined alias) rather than a static numeric version.
func isVersionAlias(version string) bool {
	if version == "" {
		return false
	}
	_, err := strconv.ParseUint(version, 10, 64)
	return err != nil
}

// ParseSecretMappings parses --set-secrets flag entries into normalized SecretMapping structs.
// Supports both comma-separated entries and repeated flags. If the same environment variable
// is specified multiple times, the last mapping takes precedence.
func ParseSecretMappings(rawEntries []string) ([]SecretMapping, error) {
	var allParts []string
	for _, entry := range rawEntries {
		for _, part := range strings.Split(entry, ",") {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" {
				allParts = append(allParts, trimmed)
			}
		}
	}

	var mappings []SecretMapping
	indexByEnvVar := make(map[string]int)

	for _, part := range allParts {
		envVar, secretRef, ok := strings.Cut(part, "=")
		envVar = strings.TrimSpace(envVar)
		secretRef = strings.TrimSpace(secretRef)
		if !ok || envVar == "" || secretRef == "" {
			return nil, fmt.Errorf("invalid --set-secrets entry %q: expected format 'ENV_VAR=SECRET_REF'", part)
		}
		if !envVarNameRegex.MatchString(envVar) {
			return nil, fmt.Errorf("invalid environment variable name %q in --set-secrets entry %q", envVar, part)
		}

		mapping, err := parseSecretRef(envVar, secretRef)
		if err != nil {
			return nil, err
		}

		if idx, exists := indexByEnvVar[envVar]; exists {
			mappings[idx] = mapping
		} else {
			indexByEnvVar[envVar] = len(mappings)
			mappings = append(mappings, mapping)
		}
	}

	return mappings, nil
}

func parseSecretRef(envVar, secretRef string) (SecretMapping, error) {
	if m := regionalSecretRegex.FindStringSubmatch(secretRef); m != nil {
		version := m[4]
		return SecretMapping{
			EnvVar:    envVar,
			SecretRef: secretRef,
			Project:   m[1],
			Location:  m[2],
			SecretID:  m[3],
			Version:   version,
			IsAlias:   isVersionAlias(version),
		}, nil
	}

	if m := globalSecretRegex.FindStringSubmatch(secretRef); m != nil {
		version := m[3]
		return SecretMapping{
			EnvVar:    envVar,
			SecretRef: secretRef,
			Project:   m[1],
			Location:  "",
			SecretID:  m[2],
			Version:   version,
			IsAlias:   isVersionAlias(version),
		}, nil
	}

	if m := shortSecretRefRegex.FindStringSubmatch(secretRef); m != nil {
		version := m[2]
		if version == "" {
			version = "latest"
		}
		return SecretMapping{
			EnvVar:    envVar,
			SecretRef: secretRef,
			Project:   "",
			Location:  "",
			SecretID:  m[1],
			Version:   version,
			IsAlias:   isVersionAlias(version),
		}, nil
	}

	return SecretMapping{}, fmt.Errorf(
		"invalid secret reference %q for environment variable %q: must be '<secret_name>', '<secret_name>:<version>', 'projects/<project>/secrets/<secret>/versions/<version>', or 'projects/<project>/locations/<location>/secrets/<secret>/versions/<version>'",
		secretRef,
		envVar,
	)
}

// resolveDefaultProject resolves the default Google Cloud project ID for short secret references.
func resolveDefaultProject(ctx context.Context, explicitProject string) (string, error) {
	if explicitProject != "" {
		return explicitProject, nil
	}
	for _, envKey := range []string{"GOOGLE_CLOUD_PROJECT", "GCLOUD_PROJECT", "CLOUDSDK_CORE_PROJECT"} {
		if val := strings.TrimSpace(os.Getenv(envKey)); val != "" {
			return val, nil
		}
	}
	creds, err := findDefaultCredentials(ctx, secretManagerCloudPlatformScope)
	if err == nil && creds != nil && strings.TrimSpace(creds.ProjectID) != "" {
		return strings.TrimSpace(creds.ProjectID), nil
	}
	return "", fmt.Errorf("unable to determine default Google Cloud project for short secret reference: set GOOGLE_CLOUD_PROJECT environment variable or use a full secret resource reference ('projects/<project>/secrets/<secret>/versions/<version>')")
}

// FetchSecretMappings resolves and fetches payloads from Secret Manager for the given mappings.
func FetchSecretMappings(ctx context.Context, mappings []SecretMapping, fallbackProject string) (map[string]string, error) {
	if len(mappings) == 0 {
		return nil, nil
	}

	var defaultProject string
	for i := range mappings {
		if mappings[i].Project == "" {
			if defaultProject == "" {
				var err error
				defaultProject, err = resolveDefaultProject(ctx, fallbackProject)
				if err != nil {
					return nil, fmt.Errorf("resolving secret %q for %q: %w", mappings[i].SecretRef, mappings[i].EnvVar, err)
				}
			}
			mappings[i].Project = defaultProject
		}
	}

	var baseOpts []option.ClientOption
	if ua, err := util.UserAgentFromContext(ctx); err == nil && ua != "" {
		baseOpts = append(baseOpts, option.WithUserAgent(ua))
	}

	clients := make(map[string]SecretManagerClient)
	defer func() {
		for _, c := range clients {
			_ = c.Close()
		}
	}()

	crc32cTable := crc32.MakeTable(crc32.Castagnoli)
	resolved := make(map[string]string, len(mappings))

	for _, m := range mappings {
		client, ok := clients[m.Location]
		if !ok {
			var err error
			client, err = newSecretManagerClient(ctx, m.Location, baseOpts...)
			if err != nil {
				return nil, fmt.Errorf("failed to initialize Secret Manager client for %q: %w", m.EnvVar, err)
			}
			clients[m.Location] = client
		}

		resName := m.ResourceName()
		resp, err := client.AccessSecretVersion(ctx, &secretmanagerpb.AccessSecretVersionRequest{
			Name: resName,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to access secret %q for environment variable %q: %w", resName, m.EnvVar, err)
		}

		payload := resp.GetPayload()
		if payload == nil {
			return nil, fmt.Errorf("secret %q for environment variable %q returned an empty payload", resName, m.EnvVar)
		}

		data := payload.GetData()
		if payload.DataCrc32C != nil {
			checksum := int64(crc32.Checksum(data, crc32cTable))
			if checksum != payload.GetDataCrc32C() {
				return nil, fmt.Errorf("secret %q for environment variable %q failed CRC32C integrity check", resName, m.EnvVar)
			}
		}

		resolved[m.EnvVar] = string(data)
	}

	return resolved, nil
}

// LoadSecretsIntoEnv fetches configured secrets from Google Cloud Secret Manager
// and populates them into process environment variables prior to parsing tools.yaml.
// Secret Manager values take precedence over pre-existing OS environment variables.
func (opts *ToolboxOptions) LoadSecretsIntoEnv(ctx context.Context) error {
	if len(opts.Secrets) == 0 || opts.SkipSecretFetch {
		return nil
	}

	mappings, err := ParseSecretMappings(opts.Secrets)
	if err != nil {
		return err
	}
	if len(mappings) == 0 {
		return nil
	}

	resolved, err := FetchSecretMappings(ctx, mappings, opts.Cfg.TelemetryGCPProject)
	if err != nil {
		return err
	}

	if opts.ResolvedSecrets == nil {
		opts.ResolvedSecrets = make(map[string]string, len(resolved))
	}
	for envVar, val := range resolved {
		if err := os.Setenv(envVar, val); err != nil {
			return fmt.Errorf("failed to set environment variable %q from secret: %w", envVar, err)
		}
		opts.ResolvedSecrets[envVar] = val
	}

	return nil
}

// HasAliasSecrets returns true if at least one secret in opts.Secrets uses a
// version alias (such as "latest" or a user-defined alias) rather than a static numeric version.
func (opts *ToolboxOptions) HasAliasSecrets() bool {
	if len(opts.Secrets) == 0 {
		return false
	}
	mappings, err := ParseSecretMappings(opts.Secrets)
	if err != nil {
		return false
	}
	for _, m := range mappings {
		if m.IsAlias {
			return true
		}
	}
	return false
}

// PollAliasSecrets queries Secret Manager for all alias-referenced secrets and
// compares their payloads against lastSeen. Static numeric versions are skipped.
func (opts *ToolboxOptions) PollAliasSecrets(ctx context.Context, lastSeen map[string]string) (map[string]string, bool, error) {
	if len(opts.Secrets) == 0 {
		return nil, false, nil
	}

	mappings, err := ParseSecretMappings(opts.Secrets)
	if err != nil {
		return nil, false, err
	}

	var aliasMappings []SecretMapping
	for _, m := range mappings {
		if m.IsAlias {
			aliasMappings = append(aliasMappings, m)
		}
	}
	if len(aliasMappings) == 0 {
		return nil, false, nil
	}

	fetched, err := FetchSecretMappings(ctx, aliasMappings, opts.Cfg.TelemetryGCPProject)
	if err != nil {
		return nil, false, err
	}

	changed := false
	for envVar, newVal := range fetched {
		if oldVal, exists := lastSeen[envVar]; !exists || oldVal != newVal {
			changed = true
		}
	}

	return fetched, changed, nil
}
