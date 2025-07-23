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

package trino

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestBuildTrinoDSN(t *testing.T) {
	tests := []struct {
		name            string
		host            string
		port            string
		user            string
		password        string
		catalog         string
		schema          string
		queryTimeout    string
		accessToken     string
		kerberosEnabled bool
		sslEnabled      bool
		want            string
		wantErr         bool
	}{
		{
			name:     "basic configuration",
			host:     "localhost",
			port:     "8080",
			user:     "testuser",
			catalog:  "hive",
			schema:   "default",
			want:     "http://testuser@localhost:8080?catalog=hive&schema=default",
			wantErr:  false,
		},
		{
			name:     "with password",
			host:     "localhost",
			port:     "8080",
			user:     "testuser",
			password: "testpass",
			catalog:  "hive",
			schema:   "default",
			want:     "http://testuser:testpass@localhost:8080?catalog=hive&schema=default",
			wantErr:  false,
		},
		{
			name:         "with SSL",
			host:         "localhost",
			port:         "8443",
			user:         "testuser",
			catalog:      "hive",
			schema:       "default",
			sslEnabled:   true,
			want:         "https://testuser@localhost:8443?catalog=hive&schema=default",
			wantErr:      false,
		},
		{
			name:            "with access token",
			host:            "localhost",
			port:            "8080",
			user:            "testuser",
			catalog:         "hive",
			schema:          "default",
			accessToken:     "jwt-token-here",
			want:            "http://testuser@localhost:8080?accessToken=jwt-token-here&catalog=hive&schema=default",
			wantErr:         false,
		},
		{
			name:            "with kerberos",
			host:            "localhost",
			port:            "8080",
			user:            "testuser",
			catalog:         "hive",
			schema:          "default",
			kerberosEnabled: true,
			want:            "http://testuser@localhost:8080?KerberosEnabled=true&catalog=hive&schema=default",
			wantErr:         false,
		},
		{
			name:         "with query timeout",
			host:         "localhost",
			port:         "8080",
			user:         "testuser",
			catalog:      "hive",
			schema:       "default",
			queryTimeout: "30m",
			want:         "http://testuser@localhost:8080?catalog=hive&queryTimeout=30m&schema=default",
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildTrinoDSN(tt.host, tt.port, tt.user, tt.password, tt.catalog, tt.schema, tt.queryTimeout, tt.accessToken, tt.kerberosEnabled, tt.sslEnabled)
			if (err != nil) != tt.wantErr {
				t.Errorf("buildTrinoDSN() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("buildTrinoDSN() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
