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

/**
 * These functions runs after the browser finishes loading and parsing HTML structure.
 * This ensures that elements can be safely accessed.
 */
document.addEventListener('DOMContentLoaded', () => {
    const toolDisplayArea = document.getElementById('tool-display-area');
    const secondaryPanelContent = document.getElementById('secondary-panel-content');

    if (!secondaryPanelContent || !toolDisplayArea) {
        console.error('Required DOM elements not found.');
        return;
    }

    let toolDetailsAbortController = null;

    // fetches tools 
    async function loadTools() {
        secondaryPanelContent.innerHTML = '<p>Fetching tools...</p>';
        try {
            const response = await fetch('/api/toolset');
            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }
            const apiResponse = await response.json();
            renderToolList(apiResponse);
        } catch (error) {
            console.error('Failed to load tools:', error);
            secondaryPanelContent.innerHTML = '<p class="error">Failed to load tools. Please try again later.</p>';
        }
    }

    // renders the fetched tools into the nav bar
    function renderToolList(apiResponse) {
        secondaryPanelContent.innerHTML = '';

        if (!apiResponse || typeof apiResponse.tools !== 'object' || apiResponse.tools === null) {
            console.error('Error: Expected an object with a "tools" property, but received:', apiResponse);
            secondaryPanelContent.textContent = 'Error: Invalid response format from toolset API.';
            return;
        }

        const toolsObject = apiResponse.tools;
        const toolNames = Object.keys(toolsObject);

        if (toolNames.length === 0) {
            secondaryPanelContent.textContent = 'No tools found.';
            return;
        }

        const ul = document.createElement('ul');
        toolNames.forEach(toolName => {
            const li = document.createElement('li');
            const button = document.createElement('button');
            button.textContent = toolName;
            button.dataset.toolname = toolName;
            button.classList.add('tool-button');
            button.addEventListener('click', handleToolClick);
            li.appendChild(button);
            ul.appendChild(li);
        });
        secondaryPanelContent.appendChild(ul);
    }

    // handles selecting a specific tool from the secondary nav bar
    function handleToolClick(event) {
        const toolName = event.target.dataset.toolname;
        if (toolName) {
            const currentActive = secondaryPanelContent.querySelector('.tool-button.active');
            if (currentActive) {
                currentActive.classList.remove('active');
            }
            event.target.classList.add('active');
            fetchToolDetails(toolName);
        }
    }

    // fetches details for specific tool
    async function fetchToolDetails(toolName) {

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
                parameters: (toolObject.parameters || []).map(param => {
                    let inputType = 'text'; 
                    const apiType = param.type ? param.type.toLowerCase() : 'string';
                    let valueType = 'string'; 
                    let label = param.description || param.name;

                    if (apiType === 'integer' || apiType === 'number') {
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

    // Initial load of tools list
    loadTools();
});
