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

    const summary = document.createElement('p');
    summary.innerHTML = `<strong>Type:</strong> ${formatSourceType(source.kind)}`;

    const configTitle = document.createElement('h5');
    configTitle.textContent = 'Configuration';

    const configList = document.createElement('ul');
    const configEntries = source.config && typeof source.config === 'object' ? Object.entries(source.config) : [];
    if (configEntries.length === 0) {
        const emptyItem = document.createElement('li');
        emptyItem.textContent = 'No configuration details available.';
        configList.appendChild(emptyItem);
    } else {
        configEntries.forEach(([key, value]) => {
            const item = document.createElement('li');
            item.textContent = `${key}: ${formatConfigValue(value)}`;
            configList.appendChild(item);
        });
    }

    wrapper.appendChild(title);
    wrapper.appendChild(kind);
    wrapper.appendChild(summary);
    wrapper.appendChild(configTitle);
    wrapper.appendChild(configList);
    container.appendChild(wrapper);
}

function formatConfigValue(value) {
    if (value === null || value === undefined) {
        return 'null';
    }
    if (typeof value === 'object') {
        try {
            return JSON.stringify(value);
        } catch (e) {
            return '[object]';
        }
    }
    return String(value);
}

function formatSourceType(kind) {
    if (!kind) {
        return 'Unknown source';
    }
    const normalized = String(kind).replace(/[_-]+/g, ' ').trim().toLowerCase();
    return `${capitalizeWords(normalized)} source`;
}

function capitalizeWords(value) {
    return value.replace(/\b\w/g, char => char.toUpperCase());
}
