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

import { fetchJsonObjectByKey } from "./apiFetch.js";

/**
 * Fetches details for a specific auth service.
 * @param {string} authServiceName The name of the auth service to fetch details for.
 * @returns {!Promise<{name: string, kind: string}>}
 */
export async function fetchAuthService(authServiceName) {
  const authServices = await fetchJsonObjectByKey(
    `/api/authservice/${encodeURIComponent(authServiceName)}`,
    "authServices"
  );
  const service = authServices[authServiceName];
  if (!service) {
    throw new Error(
      `Auth service "${authServiceName}" not found in API response.`
    );
  }

  return {
    name: service.name || authServiceName,
    kind: service.kind || "",
    headerName: service.headerName || "",
    tools: Array.isArray(service.tools) ? service.tools : [],
  };
}

/**
 * Fetches the list of auth services from the API.
 * @returns {!Promise<!Array<{name: string, kind: string}>>}
 */
export async function fetchAuthServices() {
  const authServices = await fetchJsonObjectByKey(
    "/api/authservice",
    "authServices"
  );
  return Object.values(authServices).map((service) => ({
    name: service.name || "",
    kind: service.kind || "",
    headerName: service.headerName || "",
    tools: Array.isArray(service.tools) ? service.tools : [],
  }));
}
