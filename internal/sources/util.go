// Copyright 2025 Google LLC
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

package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"cloud.google.com/go/cloudsqlconn"
	"golang.org/x/oauth2/google"
)

// GetCloudSQLDialOpts retrieve dial options with the right ip type and user agent for cloud sql
// databases.
func GetCloudSQLOpts(ipType, userAgent string, useIAM bool) ([]cloudsqlconn.Option, error) {
	opts := []cloudsqlconn.Option{cloudsqlconn.WithUserAgent(userAgent)}
	switch strings.ToLower(ipType) {
	case "private":
		opts = append(opts, cloudsqlconn.WithDefaultDialOptions(cloudsqlconn.WithPrivateIP()))
	case "public":
		opts = append(opts, cloudsqlconn.WithDefaultDialOptions(cloudsqlconn.WithPublicIP()))
	default:
		return nil, fmt.Errorf("invalid ipType %s", ipType)
	}

	if useIAM {
		opts = append(opts, cloudsqlconn.WithIAMAuthN())
	}
	return opts, nil
}

// GetIAMPrincipalEmailFromADC finds the email associated with ADC
func GetIAMPrincipalEmailFromADC(ctx context.Context) (string, error) {
	// Finds ADC and returns an HTTP client associated with it
	client, err := google.DefaultClient(ctx,
		"https://www.googleapis.com/auth/userinfo.email")
	if err != nil {
		return "", fmt.Errorf("failed to call userinfo endpoint: %w", err)
	}

	// Retrieve the email associated with the token
	resp, err := client.Get("https://oauth2.googleapis.com/tokeninfo")
	if err != nil {
		return "", fmt.Errorf("failed to call tokeninfo endpoint: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body %d: %s", resp.StatusCode, string(bodyBytes))
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("tokeninfo endpoint returned non-OK status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Unmarshal response body and get `email`
	var responseJSON map[string]any
	err = json.Unmarshal(bodyBytes, &responseJSON)
	if err != nil {

		return "", fmt.Errorf("error parsing JSON: %v", err)
	}

	emailValue, ok := responseJSON["email"]
	if !ok {
		return "", fmt.Errorf("email not found in response: %v", err)
	}
	// service account email used for IAM should trim the suffix
	email := strings.TrimSuffix(emailValue.(string), ".gserviceaccount.com")
	return email, nil
}

func GetIAMOAuthAccessToken(ctx context.Context) (string, error) {
	creds, err := google.FindDefaultCredentials(ctx, "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		return "", fmt.Errorf("failed to find default credentials (run 'gcloud auth application-default login'?): %w", err)
	}

	token, err := creds.TokenSource.Token() // This gets an oauth2.Token
	if err != nil {
		return "", fmt.Errorf("failed to get token from token source: %w", err)
	}

	if !token.Valid() {
		return "", fmt.Errorf("retrieved token is invalid or expired")
	}
	return token.AccessToken, nil
}
