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

/**
 * Fetches the list of sources from the API.
 * @returns {!Promise<!Array<{name: string, kind: string}>>}
 */
export async function fetchSources() {
    const response = await fetch('/api/source');
    if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
    }
    const apiResponse = await response.json();
    if (!apiResponse || typeof apiResponse.sources !== 'object' || apiResponse.sources === null) {
        throw new Error('Invalid response format from source API.');
    }

    return Object.values(apiResponse.sources).map(source => ({
        name: source.name || '',
        kind: source.kind || '',
    }));
}

/**
 * Fetches details for a specific source.
 * @param {string} sourceName The name of the source to fetch details for.
 * @returns {!Promise<{name: string, kind: string}>}
 */
export async function fetchSource(sourceName) {
    const response = await fetch(`/api/source/${encodeURIComponent(sourceName)}`);
    if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
    }
    const apiResponse = await response.json();
    if (!apiResponse || typeof apiResponse.sources !== 'object' || apiResponse.sources === null) {
        throw new Error('Invalid response format from source API.');
    }
    const source = apiResponse.sources[sourceName];
    if (!source) {
        throw new Error(`Source "${sourceName}" not found in API response.`);
    }

    return {
        name: source.name || sourceName,
        kind: source.kind || '',
        config: source.config || {},
    };
}
