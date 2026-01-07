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
 * Renders source details into the main content area.
 * @param {{name: string, kind: string}} source The source to render.
 * @param {!HTMLElement} container The container to render into.
 */
export function renderSourceDetails(source, container) {
    container.innerHTML = '';

    const wrapper = document.createElement('div');
    wrapper.className = 'tool-box';

    const title = document.createElement('h3');
    title.textContent = source.name || 'Unnamed source';

    const kind = document.createElement('p');
    kind.innerHTML = `<strong>Kind:</strong> ${source.kind || 'unknown'}`;

    wrapper.appendChild(title);
    wrapper.appendChild(kind);
    container.appendChild(wrapper);
}
