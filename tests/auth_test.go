//go:build integration

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
package tests

import (
	"io"
	"net/http"

	"testing"

	"github.com/googleapis/genai-toolbox/internal/auth"
	"github.com/googleapis/genai-toolbox/internal/auth/google"
)

// Returns an ID token of the service account
func getGoogleIdToken(audience string) (string, error) {
	url := "http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/identity?audience=" + audience
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Metadata-Flavor", "Google")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	return string(body), nil
}

func TestGoogleAuthVerification(t *testing.T) {
	clientId := "32555940559.apps.googleusercontent.com"
	tcs := []struct {
		authSource auth.AuthSource
		isErr      bool
	}{
		{
			authSource: google.AuthSource{
				Name:     "my-google-auth",
				Kind:     google.AuthSourceKind,
				ClientID: clientId,
			},
			isErr: false,
		},
		{
			authSource: google.AuthSource{
				Name:     "err-google-auth",
				Kind:     google.AuthSourceKind,
				ClientID: "random-client-id",
			},
			isErr: true,
		},
	}
	for _, tc := range tcs {

		token, err := getGoogleIdToken(clientId)
		if err != nil {
			t.Fatalf("ID token generation error: %s", err)
		}
		headers := http.Header{}
		headers.Add("my-google-auth_token", token)
		claims, err := tc.authSource.GetClaimsFromHeader(headers)

		if err != nil {
			if tc.isErr {
				return
			} else {
				t.Fatalf("Error getting claims from token: %s", err)
			}
		}

		_, ok := claims["sub"]
		if !ok {
			t.Fatalf("Invalid claims.")
		}
	}
}
