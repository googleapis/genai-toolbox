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
	"context"
	"os"
	"os/exec"
	"strings"

	"google.golang.org/api/idtoken"
)

var ServiceAccountEmail = os.Getenv("SERVICE_ACCOUNT_EMAIL")
var ClientId = os.Getenv("CLIENT_ID")

// GetGoogleIdToken retrieve and return the Google ID token
func GetGoogleIdToken(audience string) (string, error) {
	// For local testing - use gcloud command to print personal ID token
	cmd := exec.Command("gcloud", "auth", "print-identity-token")
	output, err := cmd.Output()
	if err == nil {
		return strings.TrimSpace(string(output)), nil
	}
	// For Cloud Build testing - retrieve ID token from GCE metadata server
	ts, err := idtoken.NewTokenSource(context.Background(), ClientId)
	if err != nil {
		return "", err
	}
	token, err := ts.Token()
	if err != nil {
		return "", err
	}
	return token.AccessToken, nil
}
