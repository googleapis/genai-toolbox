// Copyright 2026 Google LLC
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

package util

import (
	"context"
	"os"
	"testing"

	"google.golang.org/api/option"
)

func TestGetGDAEndpoint(t *testing.T) {
	tests := []struct {
		name             string
		useClientCert    string
		useMtlsEndpoint  string
		expectedEndpoint string
	}{
		{
			name:             "default behavior (no env vars)",
			useClientCert:    "",
			useMtlsEndpoint:  "",
			expectedEndpoint: gdaDefaultEndpoint,
		},
		{
			name:             "client cert enabled, mtls auto",
			useClientCert:    "true",
			useMtlsEndpoint:  "auto",
			expectedEndpoint: gdaMTLSEndpoint,
		},
		{
			name:             "client cert enabled, mtls unset",
			useClientCert:    "true",
			useMtlsEndpoint:  "",
			expectedEndpoint: gdaMTLSEndpoint,
		},
		{
			name:             "client cert disabled, mtls auto",
			useClientCert:    "false",
			useMtlsEndpoint:  "auto",
			expectedEndpoint: gdaDefaultEndpoint,
		},
		{
			name:             "client cert disabled, mtls always",
			useClientCert:    "false",
			useMtlsEndpoint:  "always",
			expectedEndpoint: gdaMTLSEndpoint,
		},
		{
			name:             "client cert enabled, mtls never",
			useClientCert:    "true",
			useMtlsEndpoint:  "never",
			expectedEndpoint: gdaDefaultEndpoint,
		},
		{
			name:             "client cert enabled, mtls invalid",
			useClientCert:    "true",
			useMtlsEndpoint:  "invalid-mode",
			expectedEndpoint: gdaMTLSEndpoint, // defaults to auto, so mtls endpoint
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Set environment variables
			if tc.useClientCert != "" {
				os.Setenv("GOOGLE_API_USE_CLIENT_CERTIFICATE", tc.useClientCert)
			} else {
				os.Unsetenv("GOOGLE_API_USE_CLIENT_CERTIFICATE")
			}

			if tc.useMtlsEndpoint != "" {
				os.Setenv("GOOGLE_API_USE_MTLS_ENDPOINT", tc.useMtlsEndpoint)
			} else {
				os.Unsetenv("GOOGLE_API_USE_MTLS_ENDPOINT")
			}

			// Clean up env vars
			defer func() {
				os.Unsetenv("GOOGLE_API_USE_CLIENT_CERTIFICATE")
				os.Unsetenv("GOOGLE_API_USE_MTLS_ENDPOINT")
			}()

			if endpoint := GetGDAEndpoint(); endpoint != tc.expectedEndpoint {
				t.Errorf("expected endpoint %q, got %q", tc.expectedEndpoint, endpoint)
			}
		})
	}
}

func TestNewGDAClient(t *testing.T) {
	ctx := context.Background()

	// Should be able to create a client with no options (uses ADC)
	client, err := NewGDAClient(ctx, option.WithoutAuthentication())
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
}
