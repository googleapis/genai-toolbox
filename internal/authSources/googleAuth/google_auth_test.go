// Copyright 2024 Google LLC
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

package googleAuth_test

import (
	"os/exec"
	"testing"

	"github.com/googleapis/genai-toolbox/internal/authSources/googleAuth"
)

func TestGoogleAuthVerification(t *testing.T) {

	authSource := googleAuth.AuthSource{
		Name: "my-google-auth",
		Kind: googleAuth.AuthSourceKind,
	}

	cmd := exec.Command("gcloud", "auth", "print-identity-token")
	out, _ := cmd.Output()
	claims, err := authSource.Verify(string(out))

	if err != nil {
		t.Fatalf("unable to unmarshal: %s", err)
	}

	_, ok := claims["sub"].(string)
	if !ok {
		t.Fatalf("invalid claims: %s", err)
	}
}
