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

package generic

import (
	"context"
	"net/http"
)

func NewAuthServiceForTest(cfg Config, client *http.Client, issuer string) *AuthService {
	return &AuthService{
		Config: cfg,
		client: client,
		issuer: issuer,
	}
}

func (a *AuthService) ValidateJwtTokenForTest(ctx context.Context, tokenStr string) (map[string]any, error) {
	return a.validateJwtToken(ctx, tokenStr)
}

func (a *AuthService) ValidateOpaqueTokenForTest(ctx context.Context, tokenStr string) (map[string]any, error) {
	return a.validateOpaqueToken(ctx, tokenStr)
}

func NewSecureHTTPClientForTest() *http.Client {
	return newSecureHTTPClient()
}

func IsJWTFormatForTest(token string) bool {
	return isJWTFormat(token)
}
