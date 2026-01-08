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
 * Fetches details for a specific source.
 * @param {string} sourceName The name of the source to fetch details for.
 * @returns {!Promise<{name: string, kind: string}>}
 */
export async function fetchSource(sourceName) {
  const sources = await fetchJsonObjectByKey(
    `/api/source/${encodeURIComponent(sourceName)}`,
    "sources"
  );
  const source = sources[sourceName];
  if (!source) {
    throw new Error(`Source "${sourceName}" not found in API response.`);
  }

  return {
    name: source.name || sourceName,
    kind: source.kind || "",
    config: source.config || {},
  };
}

/**
 * Fetches the list of sources from the API.
 * @returns {!Promise<!Array<{name: string, kind: string}>>}
 */
export async function fetchSources() {
  const sources = await fetchJsonObjectByKey("/api/source", "sources");
  return Object.values(sources).map((source) => ({
    name: source.name || "",
    kind: source.kind || "",
  }));
}
