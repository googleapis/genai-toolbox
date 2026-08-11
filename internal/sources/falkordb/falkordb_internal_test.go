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

package falkordb

import "testing"

func TestValidateTLS(t *testing.T) {
	tcs := []struct {
		desc               string
		enabled            bool
		insecureSkipVerify bool
		wantErr            bool
	}{
		{
			desc:               "insecureSkipVerify without tls is rejected",
			enabled:            false,
			insecureSkipVerify: true,
			wantErr:            true,
		},
		{
			desc:               "tls disabled without insecureSkipVerify is accepted",
			enabled:            false,
			insecureSkipVerify: false,
			wantErr:            false,
		},
		{
			desc:               "insecureSkipVerify with tls enabled is accepted",
			enabled:            true,
			insecureSkipVerify: true,
			wantErr:            false,
		},
		{
			desc:               "tls enabled with verification is accepted",
			enabled:            true,
			insecureSkipVerify: false,
			wantErr:            false,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			cfg := Config{
				Name: "my-falkordb-instance",
				TLS: TLSConfig{
					Enabled:            tc.enabled,
					InsecureSkipVerify: tc.insecureSkipVerify,
				},
			}
			err := cfg.validateTLS()
			if (err != nil) != tc.wantErr {
				t.Fatalf("validateTLS() error = %v, wantErr %t", err, tc.wantErr)
			}
		})
	}
}
