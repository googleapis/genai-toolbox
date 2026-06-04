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

package bigquery

import (
	"strings"

	"google.golang.org/api/option"
)

// normalizeAPIEndpoint returns a value accepted by option.WithEndpoint for both
// the bigquery client and the bigquery/v2 REST service. Empty/whitespace yields
// "" (default Google endpoint). The scheme is preserved so http-only proxies and
// emulators (e.g. http://localhost:9050) keep working; a missing scheme defaults
// to https. A default port is appended when absent (:80 for http, else :443), and
// a trailing slash is removed.
func normalizeAPIEndpoint(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	scheme := "https://"
	for _, p := range []string{"https://", "http://"} {
		if len(s) >= len(p) && strings.EqualFold(s[:len(p)], p) {
			scheme, s = p, s[len(p):]
			break
		}
	}
	host := strings.TrimSuffix(s, "/")
	if !strings.Contains(host, ":") {
		if scheme == "http://" {
			host += ":80"
		} else {
			host += ":443"
		}
	}
	return scheme + host
}

func appendAPIEndpointOption(opts []option.ClientOption, apiEndpoint string) []option.ClientOption {
	if ep := normalizeAPIEndpoint(apiEndpoint); ep != "" {
		return append(opts, option.WithEndpoint(ep))
	}
	return opts
}
