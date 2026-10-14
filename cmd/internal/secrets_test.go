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
	"path/filepath"
	"strings"
	"testing"

	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/gax-go/v2"
	"github.com/googleapis/mcp-toolbox/internal/testutils"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
)

type fakeSecretManagerClient struct {
	location string
	secrets  map[string]*secretmanagerpb.SecretPayload
	accessed []string
}

func (f *fakeSecretManagerClient) AccessSecretVersion(_ context.Context, req *secretmanagerpb.AccessSecretVersionRequest, _ ...gax.CallOption) (*secretmanagerpb.AccessSecretVersionResponse, error) {
	f.accessed = append(f.accessed, req.GetName())
	payload, ok := f.secrets[req.GetName()]
	if !ok {
		return nil, fmt.Errorf("secret version %q not found", req.GetName())
	}
	return &secretmanagerpb.AccessSecretVersionResponse{
		Name:    req.GetName(),
		Payload: payload,
	}, nil
}

func (f *fakeSecretManagerClient) Close() error {
	return nil
}

func makePayload(val string) *secretmanagerpb.SecretPayload {
	data := []byte(val)
	crc := int64(crc32.Checksum(data, crc32.MakeTable(crc32.Castagnoli)))
	return &secretmanagerpb.SecretPayload{
		Data:       data,
		DataCrc32C: &crc,
	}
}

func TestParseSecretMappings(t *testing.T) {
	tcs := []struct {
		desc    string
		entries []string
		want    []SecretMapping
		wantErr string
	}{
		{
			desc:    "short secret name only defaults to latest",
			entries: []string{"DB_PASSWORD=my-db-secret"},
			want: []SecretMapping{
				{
					EnvVar:    "DB_PASSWORD",
					SecretRef: "my-db-secret",
					Project:   "",
					Location:  "",
					SecretID:  "my-db-secret",
					Version:   "latest",
					IsAlias:   true,
				},
			},
		},
		{
			desc:    "short secret name with numeric version",
			entries: []string{"DB_PASSWORD=my-db-secret:3"},
			want: []SecretMapping{
				{
					EnvVar:    "DB_PASSWORD",
					SecretRef: "my-db-secret:3",
					Project:   "",
					Location:  "",
					SecretID:  "my-db-secret",
					Version:   "3",
					IsAlias:   false,
				},
			},
		},
		{
			desc:    "short secret name with version alias",
			entries: []string{"DB_PASSWORD=my-db-secret:prod-alias"},
			want: []SecretMapping{
				{
					EnvVar:    "DB_PASSWORD",
					SecretRef: "my-db-secret:prod-alias",
					Project:   "",
					Location:  "",
					SecretID:  "my-db-secret",
					Version:   "prod-alias",
					IsAlias:   true,
				},
			},
		},
		{
			desc:    "full reference global Secret Manager",
			entries: []string{"DB_PASSWORD=projects/shared-security-proj/secrets/my-db-secret/versions/1"},
			want: []SecretMapping{
				{
					EnvVar:    "DB_PASSWORD",
					SecretRef: "projects/shared-security-proj/secrets/my-db-secret/versions/1",
					Project:   "shared-security-proj",
					Location:  "",
					SecretID:  "my-db-secret",
					Version:   "1",
					IsAlias:   false,
				},
			},
		},
		{
			desc:    "full reference regional Secret Manager",
			entries: []string{"DB_PASSWORD=projects/my-project/locations/us-central1/secrets/my-db-secret/versions/latest"},
			want: []SecretMapping{
				{
					EnvVar:    "DB_PASSWORD",
					SecretRef: "projects/my-project/locations/us-central1/secrets/my-db-secret/versions/latest",
					Project:   "my-project",
					Location:  "us-central1",
					SecretID:  "my-db-secret",
					Version:   "latest",
					IsAlias:   true,
				},
			},
		},
		{
			desc:    "comma-separated list in a single flag",
			entries: []string{"PG_PASSWORD=pg-secret:latest,MYSQL_PASSWORD=mysql-secret:2,EXTERNAL_API_KEY=api-key-secret"},
			want: []SecretMapping{
				{
					EnvVar:    "PG_PASSWORD",
					SecretRef: "pg-secret:latest",
					SecretID:  "pg-secret",
					Version:   "latest",
					IsAlias:   true,
				},
				{
					EnvVar:    "MYSQL_PASSWORD",
					SecretRef: "mysql-secret:2",
					SecretID:  "mysql-secret",
					Version:   "2",
					IsAlias:   false,
				},
				{
					EnvVar:    "EXTERNAL_API_KEY",
					SecretRef: "api-key-secret",
					SecretID:  "api-key-secret",
					Version:   "latest",
					IsAlias:   true,
				},
			},
		},
		{
			desc:    "repeated flags and duplicate env var override",
			entries: []string{"PG_PASSWORD=pg-secret:1", "PG_PASSWORD=pg-secret:latest"},
			want: []SecretMapping{
				{
					EnvVar:    "PG_PASSWORD",
					SecretRef: "pg-secret:latest",
					SecretID:  "pg-secret",
					Version:   "latest",
					IsAlias:   true,
				},
			},
		},
		{
			desc:    "missing equals sign",
			entries: []string{"INVALID_ENTRY"},
			wantErr: `invalid --set-secrets entry "INVALID_ENTRY": expected format 'ENV_VAR=SECRET_REF'`,
		},
		{
			desc:    "invalid env var name",
			entries: []string{"123_INVALID=my-secret"},
			wantErr: `invalid environment variable name "123_INVALID"`,
		},
		{
			desc:    "malformed full secret path",
			entries: []string{"DB_PASSWORD=projects/my-proj/secrets/only-secret"},
			wantErr: `invalid secret reference "projects/my-proj/secrets/only-secret"`,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			got, err := ParseSecretMappings(tc.entries)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tc.wantErr)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("expected error containing %q, got %q", tc.wantErr, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("ParseSecretMappings mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestLoadSecretsIntoEnvAndConfigPrecedence(t *testing.T) {
	origNewClient := newSecretManagerClient
	origFindCreds := findDefaultCredentials
	origDetectEnv := detectRuntimeEnvironment
	t.Cleanup(func() {
		newSecretManagerClient = origNewClient
		findDefaultCredentials = origFindCreds
		detectRuntimeEnvironment = origDetectEnv
	})
	detectRuntimeEnvironment = func(context.Context) (RuntimeEnvironment, error) {
		return RuntimeEnvironment{}, nil
	}

	clientsByLocation := make(map[string]*fakeSecretManagerClient)
	secretsData := map[string]*secretmanagerpb.SecretPayload{
		"projects/default-proj/secrets/pg-secret/versions/latest":                          makePayload("sm-pg-pass"),
		"projects/shared-proj/secrets/api-key/versions/2":                                  makePayload("sm-api-key"),
		"projects/reg-proj/locations/us-central1/secrets/reg-secret/versions/custom-alias": makePayload("sm-regional-pass"),
	}

	newSecretManagerClient = func(_ context.Context, location string, _ ...option.ClientOption) (SecretManagerClient, error) {
		c := &fakeSecretManagerClient{
			location: location,
			secrets:  secretsData,
		}
		clientsByLocation[location] = c
		return c, nil
	}

	t.Setenv("GOOGLE_CLOUD_PROJECT", "default-proj")
	// Set pre-existing environment variable to verify Secret Manager takes precedence
	t.Setenv("PG_PASSWORD", "env-pg-pass-should-be-overridden")

	ctx, err := testutils.ContextWithNewLogger()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	configYAML := `
kind: source
name: my-http-source
type: http
baseUrl: https://example.com/${PG_PASSWORD}/${API_KEY}/${REG_SECRET}
`
	cfgFile := filepath.Join(t.TempDir(), "tools.yaml")
	if err := os.WriteFile(cfgFile, []byte(configYAML), 0644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	opts := &ToolboxOptions{
		Config: cfgFile,
		Secrets: []string{
			"PG_PASSWORD=pg-secret",
			"API_KEY=projects/shared-proj/secrets/api-key/versions/2,REG_SECRET=projects/reg-proj/locations/us-central1/secrets/reg-secret/versions/custom-alias",
		},
	}

	parser := &ConfigParser{}
	if _, err := opts.LoadConfig(ctx, parser); err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if got := os.Getenv("PG_PASSWORD"); got != "sm-pg-pass" {
		t.Errorf("expected Secret Manager value to override env var: got %q, want %q", got, "sm-pg-pass")
	}
	if got := parser.EnvVars["PG_PASSWORD"]; got != "sm-pg-pass" {
		t.Errorf("expected parser.EnvVars[PG_PASSWORD] = %q, got %q", "sm-pg-pass", got)
	}
	if got := parser.EnvVars["API_KEY"]; got != "sm-api-key" {
		t.Errorf("expected parser.EnvVars[API_KEY] = %q, got %q", "sm-api-key", got)
	}
	if got := parser.EnvVars["REG_SECRET"]; got != "sm-regional-pass" {
		t.Errorf("expected parser.EnvVars[REG_SECRET] = %q, got %q", "sm-regional-pass", got)
	}

	if _, ok := clientsByLocation[""]; !ok {
		t.Errorf("expected Global Secret Manager client (location=\"\") to be used")
	}
	if _, ok := clientsByLocation["us-central1"]; !ok {
		t.Errorf("expected Regional Secret Manager client (location=\"us-central1\") to be used")
	}
}

func TestLoadSecretsIntoEnv_Errors(t *testing.T) {
	origNewClient := newSecretManagerClient
	origFindCreds := findDefaultCredentials
	origDetectEnv := detectRuntimeEnvironment
	t.Cleanup(func() {
		newSecretManagerClient = origNewClient
		findDefaultCredentials = origFindCreds
		detectRuntimeEnvironment = origDetectEnv
	})
	detectRuntimeEnvironment = func(context.Context) (RuntimeEnvironment, error) {
		return RuntimeEnvironment{}, nil
	}

	t.Run("missing default project for short reference fails fast", func(t *testing.T) {
		t.Setenv("GOOGLE_CLOUD_PROJECT", "")
		t.Setenv("GCLOUD_PROJECT", "")
		t.Setenv("CLOUDSDK_CORE_PROJECT", "")
		findDefaultCredentials = func(context.Context, ...string) (*google.Credentials, error) {
			return &google.Credentials{ProjectID: ""}, nil
		}

		opts := &ToolboxOptions{
			Secrets: []string{"DB_PASSWORD=my-short-secret"},
		}
		err := opts.LoadSecretsIntoEnv(context.Background())
		if err == nil || !strings.Contains(err.Error(), "unable to determine default Google Cloud project") {
			t.Fatalf("expected missing default project error, got: %v", err)
		}
	})

	t.Run("ADC fallback project resolution succeeds", func(t *testing.T) {
		t.Setenv("GOOGLE_CLOUD_PROJECT", "")
		t.Setenv("GCLOUD_PROJECT", "")
		t.Setenv("CLOUDSDK_CORE_PROJECT", "")
		findDefaultCredentials = func(context.Context, ...string) (*google.Credentials, error) {
			return &google.Credentials{ProjectID: "adc-project"}, nil
		}
		newSecretManagerClient = func(_ context.Context, _ string, _ ...option.ClientOption) (SecretManagerClient, error) {
			return &fakeSecretManagerClient{
				secrets: map[string]*secretmanagerpb.SecretPayload{
					"projects/adc-project/secrets/my-secret/versions/latest": makePayload("adc-secret-val"),
				},
			}, nil
		}

		opts := &ToolboxOptions{
			Secrets: []string{"ADC_SECRET_VAR=my-secret"},
		}
		if err := opts.LoadSecretsIntoEnv(context.Background()); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := os.Getenv("ADC_SECRET_VAR"); got != "adc-secret-val" {
			t.Errorf("got %q, want %q", got, "adc-secret-val")
		}
	})

	t.Run("corrupted CRC32C checksum fails fast", func(t *testing.T) {
		t.Setenv("GOOGLE_CLOUD_PROJECT", "test-proj")
		badCRC := int64(99999)
		newSecretManagerClient = func(_ context.Context, _ string, _ ...option.ClientOption) (SecretManagerClient, error) {
			return &fakeSecretManagerClient{
				secrets: map[string]*secretmanagerpb.SecretPayload{
					"projects/test-proj/secrets/corrupt-secret/versions/1": {
						Data:       []byte("secret-data"),
						DataCrc32C: &badCRC,
					},
				},
			}, nil
		}

		opts := &ToolboxOptions{
			Secrets: []string{"CORRUPT_VAR=corrupt-secret:1"},
		}
		err := opts.LoadSecretsIntoEnv(context.Background())
		if err == nil || !strings.Contains(err.Error(), "CRC32C integrity check") {
			t.Fatalf("expected CRC32C error, got: %v", err)
		}
	})
}

func TestCloudRunAndGKERegionalSecretEnforcement(t *testing.T) {
	origNewClient := newSecretManagerClient
	origDetectEnv := detectRuntimeEnvironment
	t.Cleanup(func() {
		newSecretManagerClient = origNewClient
		detectRuntimeEnvironment = origDetectEnv
	})

	t.Setenv("GOOGLE_CLOUD_PROJECT", "test-proj")

	newSecretManagerClient = func(_ context.Context, location string, _ ...option.ClientOption) (SecretManagerClient, error) {
		return &fakeSecretManagerClient{
			location: location,
			secrets: map[string]*secretmanagerpb.SecretPayload{
				"projects/test-proj/locations/us-central1/secrets/reg-secret/versions/latest": makePayload("same-region-secret-val"),
			},
		}, nil
	}

	for _, platform := range []string{"Cloud Run", "GKE"} {
		t.Run(platform+" rejects short global secret", func(t *testing.T) {
			detectRuntimeEnvironment = func(context.Context) (RuntimeEnvironment, error) {
				return RuntimeEnvironment{Platform: platform, Region: "us-central1"}, nil
			}
			opts := &ToolboxOptions{
				Secrets: []string{"DB_PASSWORD=my-global-secret:latest"},
			}
			err := opts.LoadSecretsIntoEnv(context.Background())
			if err == nil || !strings.Contains(err.Error(), "global secret") || !strings.Contains(err.Error(), platform) {
				t.Fatalf("expected global secret rejection on %s, got: %v", platform, err)
			}
		})

		t.Run(platform+" rejects full global secret", func(t *testing.T) {
			detectRuntimeEnvironment = func(context.Context) (RuntimeEnvironment, error) {
				return RuntimeEnvironment{Platform: platform, Region: "us-central1"}, nil
			}
			opts := &ToolboxOptions{
				Secrets: []string{"DB_PASSWORD=projects/test-proj/secrets/my-global-secret/versions/1"},
			}
			err := opts.LoadSecretsIntoEnv(context.Background())
			if err == nil || !strings.Contains(err.Error(), "global secret") || !strings.Contains(err.Error(), platform) {
				t.Fatalf("expected global secret rejection on %s, got: %v", platform, err)
			}
		})

		t.Run(platform+" rejects cross-region secret", func(t *testing.T) {
			detectRuntimeEnvironment = func(context.Context) (RuntimeEnvironment, error) {
				return RuntimeEnvironment{Platform: platform, Region: "us-central1"}, nil
			}
			opts := &ToolboxOptions{
				Secrets: []string{"DB_PASSWORD=projects/test-proj/locations/europe-west1/secrets/reg-secret/versions/latest"},
			}
			err := opts.LoadSecretsIntoEnv(context.Background())
			if err == nil || !strings.Contains(err.Error(), "cross-region secret") || !strings.Contains(err.Error(), "europe-west1") {
				t.Fatalf("expected cross-region secret rejection on %s, got: %v", platform, err)
			}
		})

		t.Run(platform+" allows regional secret in the same location", func(t *testing.T) {
			detectRuntimeEnvironment = func(context.Context) (RuntimeEnvironment, error) {
				return RuntimeEnvironment{Platform: platform, Region: "us-central1"}, nil
			}
			opts := &ToolboxOptions{
				Secrets: []string{"SAME_REGION_VAR=projects/test-proj/locations/us-central1/secrets/reg-secret/versions/latest"},
			}
			if err := opts.LoadSecretsIntoEnv(context.Background()); err != nil {
				t.Fatalf("expected same-region secret to succeed on %s, got error: %v", platform, err)
			}
			if got := os.Getenv("SAME_REGION_VAR"); got != "same-region-secret-val" {
				t.Errorf("got %q, want %q", got, "same-region-secret-val")
			}
		})
	}
}

func TestRegionFromLocation(t *testing.T) {
	tcs := []struct {
		in   string
		want string
	}{
		{in: "projects/123456/regions/us-central1", want: "us-central1"},
		{in: "/projects/123456/regions/europe-west1", want: "europe-west1"},
		{in: "us-central1", want: "us-central1"},
		{in: "us-central1-a", want: "us-central1"},
		{in: "projects/123456/zones/asia-east1-b", want: "asia-east1"},
		{in: "", want: ""},
		{in: "invalid", want: ""},
	}
	for _, tc := range tcs {
		if got := regionFromLocation(tc.in); got != tc.want {
			t.Errorf("regionFromLocation(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestPollAliasSecrets(t *testing.T) {
	origNewClient := newSecretManagerClient
	origDetectEnv := detectRuntimeEnvironment
	t.Cleanup(func() {
		newSecretManagerClient = origNewClient
		detectRuntimeEnvironment = origDetectEnv
	})
	detectRuntimeEnvironment = func(context.Context) (RuntimeEnvironment, error) {
		return RuntimeEnvironment{}, nil
	}

	t.Setenv("GOOGLE_CLOUD_PROJECT", "poll-proj")

	fakeClient := &fakeSecretManagerClient{
		secrets: map[string]*secretmanagerpb.SecretPayload{
			"projects/poll-proj/secrets/alias-secret/versions/latest": makePayload("v1-payload"),
			"projects/poll-proj/secrets/pinned-secret/versions/2":     makePayload("pinned-payload"),
		},
	}
	newSecretManagerClient = func(_ context.Context, _ string, _ ...option.ClientOption) (SecretManagerClient, error) {
		return fakeClient, nil
	}

	pinnedOnly := &ToolboxOptions{
		Secrets: []string{"PINNED_VAR=pinned-secret:2"},
	}
	if pinnedOnly.HasAliasSecrets() {
		t.Errorf("expected HasAliasSecrets() = false for numeric version")
	}

	opts := &ToolboxOptions{
		Secrets: []string{
			"ALIAS_VAR=alias-secret:latest",
			"PINNED_VAR=pinned-secret:2",
		},
	}
	if !opts.HasAliasSecrets() {
		t.Errorf("expected HasAliasSecrets() = true when an alias version is present")
	}

	lastSeen := map[string]string{
		"ALIAS_VAR":  "v1-payload",
		"PINNED_VAR": "pinned-payload",
	}

	// 1. Poll when payload is unchanged -> changed == false, and only alias-secret is queried
	fakeClient.accessed = nil
	_, changed, err := opts.PollAliasSecrets(context.Background(), lastSeen)
	if err != nil {
		t.Fatalf("unexpected poll error: %v", err)
	}
	if changed {
		t.Errorf("expected changed = false when secret payload has not changed")
	}
	wantAccessed := []string{"projects/poll-proj/secrets/alias-secret/versions/latest"}
	if diff := cmp.Diff(wantAccessed, fakeClient.accessed); diff != "" {
		t.Errorf("expected only alias secret to be polled (-want +got):\n%s", diff)
	}

	// 2. Rotate alias secret payload -> changed == true
	fakeClient.secrets["projects/poll-proj/secrets/alias-secret/versions/latest"] = makePayload("v2-rotated-payload")
	updated, changed, err := opts.PollAliasSecrets(context.Background(), lastSeen)
	if err != nil {
		t.Fatalf("unexpected poll error: %v", err)
	}
	if !changed {
		t.Errorf("expected changed = true after secret rotation")
	}
	if got := updated["ALIAS_VAR"]; got != "v2-rotated-payload" {
		t.Errorf("got updated[ALIAS_VAR] = %q, want %q", got, "v2-rotated-payload")
	}
}
