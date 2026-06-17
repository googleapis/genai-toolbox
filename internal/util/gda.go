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
	"fmt"
	"net/http"
	"os"
	"strings"

	"google.golang.org/api/option"
	htransport "google.golang.org/api/transport/http"
)

const (
	gdaDefaultEndpoint = "https://geminidataanalytics.googleapis.com"
	gdaMTLSEndpoint    = "https://geminidataanalytics.mtls.googleapis.com"
)

// GetGDAEndpoint returns the Gemini Data Analytics API endpoint,
// choosing the mTLS endpoint if mTLS is enabled.
func GetGDAEndpoint() string {
	mtlsMode := getMTLSMode()
	if mtlsMode == "always" {
		return gdaMTLSEndpoint
	}
	if mtlsMode == "never" {
		return gdaDefaultEndpoint
	}
	// Default mode is "auto"
	if isClientCertificateEnabled() {
		return gdaMTLSEndpoint
	}
	return gdaDefaultEndpoint
}

// NewGDAClient returns an HTTP client configured for Gemini Data Analytics.
// It handles mTLS and authentication if a token source is provided.
func NewGDAClient(ctx context.Context, opts ...option.ClientOption) (*http.Client, error) {
	// Default options for GDA
	defaultOpts := []option.ClientOption{
		option.WithEndpoint(GetGDAEndpoint()),
		option.WithScopes("https://www.googleapis.com/auth/cloud-platform"),
	}

	allOpts := append(defaultOpts, opts...)

	client, _, err := htransport.NewClient(ctx, allOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create GDA HTTP client: %w", err)
	}
	return client, nil
}

func isClientCertificateEnabled() bool {
	return strings.ToLower(os.Getenv("GOOGLE_API_USE_CLIENT_CERTIFICATE")) == "true"
}

func getMTLSMode() string {
	mode := os.Getenv("GOOGLE_API_USE_MTLS_ENDPOINT")
	if mode == "" {
		mode = os.Getenv("GOOGLE_API_USE_MTLS") // Deprecated
	}
	if mode == "" {
		return "auto"
	}
	return strings.ToLower(mode)
}
