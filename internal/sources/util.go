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
	"fmt"
	"strings"

	"cloud.google.com/go/cloudsqlconn"
)

// GetCloudSQLDialOpts retrieve dial options with the right ip type and user agent for cloud sql
// databases.
func GetCloudSQLOpts(ipType, userAgent string) ([]cloudsqlconn.Option, error) {
	opts := []cloudsqlconn.Option{cloudsqlconn.WithUserAgent(userAgent)}
	switch strings.ToLower(ipType) {
	case "private":
		opts = append(opts, cloudsqlconn.WithDefaultDialOptions(cloudsqlconn.WithPrivateIP()))
	case "public":
		opts = append(opts, cloudsqlconn.WithDefaultDialOptions(cloudsqlconn.WithPublicIP()))
	default:
		return nil, fmt.Errorf("invalid ipType %s", ipType)
	}
	return opts, nil
}
