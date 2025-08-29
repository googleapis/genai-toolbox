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

import { renderToolInterface } from "./toolDisplay.js";

let toolDetailsAbortController = null;

/**
 * Renders the list of tools as buttons within the provided HTML element.
 * @param {!Array<{name: string, tools: Object<string,*>}>} toolsets The array of toolset objects.
 * @param {!HTMLElement} secondNavContent The HTML element to render the tool list into.
 * @param {!HTMLElement} toolDisplayArea The HTML element for displaying tool details (passed to event handlers).
 */
export function renderToolList(toolsets, secondNavContent, toolDisplayArea) {
    secondNavContent.innerHTML = '';

    if (!toolsets || (Array.isArray(toolsets) && toolsets.length === 0)) {
        secondNavContent.textContent = 'No tools found.';
        return;
    }

    const toolsetsArray = Array.isArray(toolsets) ? toolsets : [toolsets];

    const ul = document.createElement('ul');
    toolsetsArray.forEach((toolset, index) => {
        if (toolset && toolset.tools) {
            const toolNames = Object.keys(toolset.tools);
            toolNames.forEach(toolName => {
                const li = document.createElement('li');
                const button = document.createElement('button');
                button.dataset.toolname = toolName;
                button.classList.add('tool-button');

                const numberIndicator = document.createElement('span');
                numberIndicator.classList.add('number-indicator');
                if (toolset.name !== "") {
                    numberIndicator.textContent = index + 1;
                }
                button.appendChild(numberIndicator);

                const nameSpan = document.createElement('span');
                nameSpan.textContent = toolName;
                button.appendChild(nameSpan);

                button.addEventListener('click', (event) => handleToolClick(event, secondNavContent, toolDisplayArea));
                li.appendChild(button);
                ul.appendChild(li);
            });
        }
    });
    secondNavContent.appendChild(ul);
}

/**
 * Handles the click event on a tool button. 
 * @param {!Event} event The click event object.
 * @param {!HTMLElement} secondNavContent The parent element containing the tool buttons.
 * @param {!HTMLElement} toolDisplayArea The HTML element where tool details will be shown.
 */
function handleToolClick(event, secondNavContent, toolDisplayArea) {
    const toolButton = event.currentTarget;
    const toolName = toolButton.dataset.toolname;
    if (toolName) {
        const currentActive = secondNavContent.querySelector('.tool-button.active');
        if (currentActive) {
            currentActive.classList.remove('active');
        }
        toolButton.classList.add('active');
        fetchToolDetails(toolName, toolDisplayArea);
    }
}

/**
 * Fetches details for a specific tool /api/tool endpoint.
 * It aborts any previous in-flight request for tool details to stop race condition.
 * @param {string} toolName The name of the tool to fetch details for.
 * @param {!HTMLElement} toolDisplayArea The HTML element to display the tool interface in.
 * @returns {!Promise<void>} A promise that resolves when the tool details are fetched and rendered, or rejects on error.
 */
async function fetchToolDetails(toolName, toolDisplayArea) {
    if (toolDetailsAbortController) {
        toolDetailsAbortController.abort();
        console.debug("Aborted previous tool fetch.");
    }

    toolDetailsAbortController = new AbortController();
    const signal = toolDetailsAbortController.signal;

    toolDisplayArea.innerHTML = '<p>Loading tool details...</p>';

    try {
        const response = await fetch(`/api/tool/${encodeURIComponent(toolName)}`, { signal });
        if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
        }
        const apiResponse = await response.json();

        if (!apiResponse.tools || !apiResponse.tools[toolName]) {
            throw new Error(`Tool "${toolName}" data not found in API response.`);
        }
        const toolObject = apiResponse.tools[toolName];
        console.debug("Received tool object: ", toolObject)

        const toolInterfaceData = {
            id: toolName,
            name: toolName,
            description: toolObject.description || "No description provided.",
            authRequired: toolObject.authRequired || [],
            parameters: (toolObject.parameters || []).map(param => {
                let inputType = 'text'; 
                const apiType = param.type ? param.type.toLowerCase() : 'string';
                let valueType = 'string'; 
                let label = param.description || param.name;

                if (apiType === 'integer' || apiType === 'float') {
                    inputType = 'number';
                    valueType = 'number';
                } else if (apiType === 'boolean') {
                    inputType = 'checkbox';
                    valueType = 'boolean';
                } else if (apiType === 'array') {
                    inputType = 'textarea'; 
                    const itemType = param.items && param.items.type ? param.items.type.toLowerCase() : 'string';
                    valueType = `array<${itemType}>`;
                    label += ' (Array)';
                }

                return {
                    name: param.name,
                    type: inputType,    
                    valueType: valueType, 
                    label: label,
                    authServices: param.authSources,
                    required: param.required || false,
                    // defaultValue: param.default, can't do this yet bc tool manifest doesn't have default
                };
            })
        };

        console.debug("Transformed toolInterfaceData:", toolInterfaceData);

        renderToolInterface(toolInterfaceData, toolDisplayArea);
    } catch (error) {
        if (error.name === 'AbortError') {
            console.debug("Previous fetch was aborted, expected behavior.");
        } else {
            console.error(`Failed to load details for tool "${toolName}":`, error);
            toolDisplayArea.innerHTML = `<p class="error">Failed to load details for ${toolName}. ${error.message}</p>`;
        }
    }
}
