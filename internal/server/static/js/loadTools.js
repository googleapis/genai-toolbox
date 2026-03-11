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
import { escapeHtml } from "./sanitize.js";

let toolDetailsAbortController = null;

/**
 * Fetches a toolset from the /api/toolset endpoint and initiates creating the tool list.
 * @param {!HTMLElement} secondNavContent The HTML element where the tool list will be rendered.
 * @param {!HTMLElement} toolDisplayArea The HTML element where the details of a selected tool will be displayed.
 * @param {string} toolsetName The name of the toolset to load (empty string loads all tools).
 * @returns {!Promise<void>} A promise that resolves when the tools are loaded and rendered, or rejects on error.
 */
export async function loadTools(secondNavContent, toolDisplayArea, toolsetName) {
    secondNavContent.innerHTML = '<p>Fetching tools...</p>';
    try {
        const rpcPayload = {
            jsonrpc: "2.0",
            id: crypto.randomUUID(),
            method: "tools/list",
            params: {}
        };

        const response = await fetch('/mcp', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(rpcPayload)
        });

        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }
        const apiResponse = await response.json();
        
        if (apiResponse.error) {
            throw new Error(`MCP error ${apiResponse.error.code}: ${apiResponse.error.message}`);
        }

        renderToolList(apiResponse.result, secondNavContent, toolDisplayArea);
    } catch (error) {
        console.error('Failed to load tools:', error);
        secondNavContent.innerHTML = `<p class="error">Failed to load tools: <pre><code>${escapeHtml(String(error))}</code></pre></p>`;
    }
}

/**
 * Renders the list of tools as buttons within the provided HTML element.
 * @param {?{tools: ?Object<string,*>} } apiResponse The API response object containing the tools.
 * @param {!HTMLElement} secondNavContent The HTML element to render the tool list into.
 * @param {!HTMLElement} toolDisplayArea The HTML element for displaying tool details (passed to event handlers).
 */
function renderToolList(apiResponse, secondNavContent, toolDisplayArea) {
    secondNavContent.innerHTML = '';

    if (!apiResponse || !Array.isArray(apiResponse.tools)) {
        console.error('Error: Expected a "tools" array, but received:', apiResponse);
        secondNavContent.textContent = 'Error: Invalid response format from tools API.';
        return;
    }

    const toolsArray = apiResponse.tools;
    const toolNames = toolsArray.map(t => t.name);

    if (toolNames.length === 0) {
        secondNavContent.textContent = 'No tools found.';
        return;
    }

    const ul = document.createElement('ul');
    toolNames.forEach(toolName => {
        const li = document.createElement('li');
        const button = document.createElement('button');
        button.textContent = toolName;
        button.dataset.toolname = toolName;
        button.classList.add('tool-button');
        button.addEventListener('click', (event) => handleToolClick(event, secondNavContent, toolDisplayArea));
        li.appendChild(button);
        ul.appendChild(li);
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
    const toolName = event.target.dataset.toolname;
    if (toolName) {
        const currentActive = secondNavContent.querySelector('.tool-button.active');
        if (currentActive) {
            currentActive.classList.remove('active');
        }
        event.target.classList.add('active');
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
        const rpcPayload = {
            jsonrpc: "2.0",
            id: crypto.randomUUID(),
            method: "tools/list",
            params: {}
        };

        const response = await fetch('/mcp', { 
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(rpcPayload),
            signal 
        });

        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }

        const apiResponse = await response.json();
        if (apiResponse.error) {
            throw new Error(`MCP error ${apiResponse.error.code}: ${apiResponse.error.message}`);
        }

        const toolsArray = apiResponse.result?.tools || [];
        const toolObject = toolsArray.find(t => t.name === toolName);

        if (!toolObject) {
            throw new Error(`Tool "${toolName}" data not found in API response.`);
        }
        console.debug("Received tool object: ", toolObject)

        // Parse MCP JSON Schema parameters back into internal UI format
        let parameters = [];
        if (toolObject.inputSchema && toolObject.inputSchema.properties) {
             const props = toolObject.inputSchema.properties;
             const requiredCols = toolObject.inputSchema.required || [];
             parameters = Object.keys(props).map(paramName => {
                 const paramSchema = props[paramName];
                 let inputType = 'text'; 
                 const apiType = paramSchema.type ? paramSchema.type.toLowerCase() : 'string';
                 let valueType = 'string'; 
                 let label = paramSchema.description || paramName;

                 if (apiType === 'integer' || apiType === 'number') {
                     inputType = 'number';
                     valueType = 'number';
                 } else if (apiType === 'boolean') {
                     inputType = 'checkbox';
                     valueType = 'boolean';
                 } else if (apiType === 'array') {
                     inputType = 'textarea'; 
                     const itemType = paramSchema.items && paramSchema.items.type ? paramSchema.items.type.toLowerCase() : 'string';
                     valueType = `array<${itemType}>`;
                     label += ' (Array)';
                 }

                 return {
                     name: paramName,
                     type: inputType,    
                     valueType: valueType, 
                     label: label,
                     required: requiredCols.includes(paramName)
                 };
             });
        }

        const toolInterfaceData = {
             id: toolObject.name,
             name: toolObject.name,
             description: toolObject.description || "No description provided.",
             parameters: parameters
        };

        console.debug("Transformed toolInterfaceData:", toolInterfaceData);
        renderToolInterface(toolInterfaceData, toolDisplayArea);
    } catch (error) {
        if (error.name === 'AbortError') {
            console.debug("Previous fetch was aborted, expected behavior.");
        } else {
            console.error(`Failed to load details for tool "${toolName}":`, error);
            toolDisplayArea.innerHTML = `<p class="error">Failed to load details for ${escapeHtml(toolName)}. ${escapeHtml(error.message)}</p>`;
        }
    }
}