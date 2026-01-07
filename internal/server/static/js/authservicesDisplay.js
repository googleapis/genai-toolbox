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
 * Renders auth service details into the main content area.
 * @param {{name: string, kind: string}} service The auth service to render.
 * @param {!HTMLElement} container The container to render into.
 */
export function renderAuthServiceDetails(service, container) {
    container.innerHTML = '';

    const wrapper = document.createElement('div');
    wrapper.className = 'tool-box';

    const title = document.createElement('h3');
    title.textContent = service.name || 'Unnamed auth service';

    const kind = document.createElement('p');
    kind.innerHTML = `<strong>Kind:</strong> ${service.kind || 'unknown'}`;

    const headerName = document.createElement('p');
    headerName.innerHTML = `<strong>Header:</strong> ${service.headerName || (service.name ? `${service.name}_token` : 'unknown')}`;

    const toolsTitle = document.createElement('h5');
    toolsTitle.textContent = 'Used by tools';

    const toolsList = document.createElement('ul');
    const tools = Array.isArray(service.tools) ? service.tools : [];
    if (tools.length === 0) {
        const emptyItem = document.createElement('li');
        emptyItem.textContent = 'No tools reference this auth service.';
        toolsList.appendChild(emptyItem);
    } else {
        tools.forEach(toolName => {
            const item = document.createElement('li');
            item.textContent = toolName;
            toolsList.appendChild(item);
        });
    }

    wrapper.appendChild(title);
    wrapper.appendChild(kind);
    wrapper.appendChild(headerName);
    wrapper.appendChild(toolsTitle);
    wrapper.appendChild(toolsList);
    container.appendChild(wrapper);
}
