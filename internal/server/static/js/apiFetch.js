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

/**
 * Fetches JSON from the provided URL and returns the object found at the given key.
 * @param {string} url
 * @param {string} key
 * @returns {Promise<object>}
 */
export async function fetchJsonObjectByKey(url, key) {
  const response = await fetch(url);
  if (!response.ok) {
    throw new Error(`HTTP error! status: ${response.status}`);
  }
  const apiResponse = await response.json();
  if (
    !apiResponse ||
    typeof apiResponse[key] !== "object" ||
    apiResponse[key] === null
  ) {
    throw new Error(`Invalid response format from API for key "${key}".`);
  }
  return apiResponse[key];
}
