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

package util

import "slices"

const (
	VERSION_20241105 = "2024-11-05"
	VERSION_20250326 = "2025-03-26"
	VERSION_20250618 = "2025-06-18"
	VERSION_20251125 = "2025-11-25"
	VERSION_20260728 = "2026-07-28"
)

// LATEST_STATEFUL_PROTOCOL_VERSION is the latest version of the MCP protocol
// supported that uses the stateful initialize handshake (< 2026).
// Update this value and InitializeResponse when a new stateful version is added.
const LATEST_STATEFUL_PROTOCOL_VERSION = VERSION_20251125

// STATEFUL_ERA_VERSIONS are the legacy MCP protocol versions
// that support the stateful initialize handshake (< 2026).
var STATEFUL_ERA_VERSIONS = []string{
	VERSION_20241105,
	VERSION_20250326,
	VERSION_20250618,
	VERSION_20251125,
}

// STATELESS_ERA_VERSIONS are the modern MCP protocol versions
// that operate statelessly without an initialize handshake (>= 2026).
var STATELESS_ERA_VERSIONS = []string{
	VERSION_20260728,
}

// SUPPORTED_PROTOCOL_VERSIONS is the composite list of all supported MCP protocol versions across both eras.
var SUPPORTED_PROTOCOL_VERSIONS = slices.Concat(STATEFUL_ERA_VERSIONS, STATELESS_ERA_VERSIONS)

var SUPPORTED_PROTOCOL_VERSIONS_NONSTABLE = []string{}

func GetSupportedVersions(enableDraft bool) []string {
	if enableDraft {
		return append(SUPPORTED_PROTOCOL_VERSIONS, SUPPORTED_PROTOCOL_VERSIONS_NONSTABLE...)
	}
	return SUPPORTED_PROTOCOL_VERSIONS
}

func GetStatefulEraVersions() []string {
	return STATEFUL_ERA_VERSIONS
}

func GetLatestStatefulVersion() string {
	return LATEST_STATEFUL_PROTOCOL_VERSION
}
