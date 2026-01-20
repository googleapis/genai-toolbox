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
 * Escapes special characters for safe rendering in HTML text contexts.
 *
 * This utility encodes user-controlled values to avoid unintended script
 * execution when rendering content as HTML. It is intended as a defensive
 * measure and does not perform HTML sanitization.
 *
 * @param {*} input The value to escape.
 * @return {string} The escaped string safe for HTML rendering.
 */
const htmlEscapes = {
    '&': '&amp;',
    '<': '&lt;',
    '>': '&gt;',
    '"': '&quot;',
    "'": '&#x27;',
    '`': '&#x60;'
};

const escapeCharsRegex = /[&<>"'`]/g;

export function escapeHtml(input) {
    if (input === null || input === undefined) {
        return '';
    }

    const str = String(input);
    return str.replace(escapeCharsRegex, (char) => htmlEscapes[char]);
}
