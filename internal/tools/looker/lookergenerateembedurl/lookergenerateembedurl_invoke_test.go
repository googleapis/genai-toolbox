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

package lookergenerateembedurl

import (
	"context"
	"net/http"
	"testing"

	"github.com/googleapis/mcp-toolbox/internal/sources"
	"github.com/googleapis/mcp-toolbox/internal/testutils"
	"github.com/googleapis/mcp-toolbox/internal/tools"
	"github.com/googleapis/mcp-toolbox/internal/util"
	"github.com/googleapis/mcp-toolbox/internal/util/parameters"
	"github.com/looker-open-source/sdk-codegen/go/rtl"
	v4 "github.com/looker-open-source/sdk-codegen/go/sdk/v4"
)

type validationTestSource struct{}

func (validationTestSource) SourceType() string { return "validation-test" }

func (validationTestSource) ToConfig() sources.SourceConfig { return nil }

func (validationTestSource) IsReadOnly() bool { return true }

func (validationTestSource) UseClientAuthorization() bool { return false }

func (validationTestSource) GetAuthTokenHeaderName() string { return "Authorization" }

func (validationTestSource) LookerApiSettings() *rtl.ApiSettings { return nil }

func (validationTestSource) GetLookerSDK(context.Context, string) (*v4.LookerSDK, error) {
	panic("GetLookerSDK should not be called for invalid parameters")
}

func (validationTestSource) LookerSessionLength() int64 { return 0 }

func (validationTestSource) GetHostURL(context.Context, *v4.LookerSDK) (string, error) {
	panic("GetHostURL should not be called for invalid parameters")
}

var _ sources.Source = validationTestSource{}

func TestInvokeRejectsEmptyEmbedParameters(t *testing.T) {
	ctx, err := testutils.ContextWithNewLogger()
	if err != nil {
		t.Fatalf("ContextWithNewLogger() error = %v", err)
	}

	tool := Tool{}
	tests := []struct {
		name   string
		params parameters.ParamValues
		want   string
	}{
		{
			name: "empty type",
			params: parameters.ParamValues{
				{Name: "type", Value: ""},
				{Name: "id", Value: "dashboard-id"},
			},
			want: "parameter 'type' cannot be empty",
		},
		{
			name: "empty id",
			params: parameters.ParamValues{
				{Name: "type", Value: "dashboards"},
				{Name: "id", Value: ""},
			},
			want: "parameter 'id' cannot be empty",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, toolboxErr := tool.Invoke(ctx, validationTestSource{}, tc.params, tools.AccessToken(""))
			if toolboxErr == nil {
				t.Fatal("Invoke() error = nil, want parameter validation error")
			}
			if got := toolboxErr.Error(); got != tc.want {
				t.Errorf("Invoke() error = %q, want %q", got, tc.want)
			}
			serverErr, ok := toolboxErr.(*util.ClientServerError)
			if !ok {
				t.Fatalf("Invoke() error type = %T, want *util.ClientServerError", toolboxErr)
			}
			if serverErr.Code != http.StatusBadRequest {
				t.Errorf("Invoke() status code = %d, want %d", serverErr.Code, http.StatusBadRequest)
			}
		})
	}
}
