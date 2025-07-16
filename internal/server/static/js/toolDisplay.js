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

import { handleRunTool, displayResults } from './runTool.js';

/**
 * Helper function to create form inputs for parameters.
 */
function createParamInput(param, toolId) {
    const paramItem = document.createElement('div');
    paramItem.className = 'param-item';

    const label = document.createElement('label');
    const INPUT_ID = `param-${toolId}-${param.name}`;
    const NAME_TEXT = document.createTextNode(param.name);
    label.setAttribute('for', INPUT_ID);
    label.appendChild(NAME_TEXT);

    const IS_AUTH_PARAM = param.authServices && param.authServices.length > 0;
    let additionalLabelText = '';
    if (IS_AUTH_PARAM) {
        additionalLabelText += ' (auth)';
    }
    if (!param.required) {
        additionalLabelText += ' (optional)';
    }

    if (additionalLabelText) {
        const additionalSpan = document.createElement('span');
        additionalSpan.textContent = additionalLabelText;
        additionalSpan.classList.add('param-label-extras');
        label.appendChild(additionalSpan);
    }
    paramItem.appendChild(label);

    // Build parameter's value input box.
    const PLACEHOLDER_LABEL = param.label;
    let inputElement;
    if (param.type === 'textarea') {
        inputElement = document.createElement('textarea');
        inputElement.rows = 3;
    } else if(param.type === 'checkbox') {
        inputElement = document.createElement('input');
        inputElement.type = 'checkbox';
        inputElement.title = PLACEHOLDER_LABEL;
    } else {
        inputElement = document.createElement('input');
        inputElement.type = param.type;
    }
    
    inputElement.id = INPUT_ID;
    inputElement.name = param.name;
    if (IS_AUTH_PARAM) {
        inputElement.disabled = true;
        inputElement.classList.add('auth-param-input');
        if (param.type !== 'checkbox') {
            inputElement.placeholder = param.authServices;
        }
    } else if (param.type !== 'checkbox') {
        inputElement.placeholder = PLACEHOLDER_LABEL.trim();
    }
    paramItem.appendChild(inputElement);
    return paramItem;
}

/**
 * Function to create the header editor popup modal.
 */
function createHeaderEditorModal(toolId, currentHeaders, saveCallback) {
    const MODAL_ID = `header-modal-${toolId}`;
    let modal = document.getElementById(MODAL_ID);

    if (modal) {
        modal.remove(); 
    }

    modal = document.createElement('div');
    modal.id = MODAL_ID;
    modal.className = 'header-modal';

    const modalContent = document.createElement('div');
    const modalHeader = document.createElement('h5');
    const headersTextarea = document.createElement('textarea');

    modalContent.className = 'header-modal-content';
    modalHeader.textContent = 'Edit Request Headers';
    headersTextarea.id = `headers-textarea-${toolId}`;
    headersTextarea.className = 'headers-textarea';
    headersTextarea.rows = 10;
    headersTextarea.value = JSON.stringify(currentHeaders, null, 2);

    modalContent.appendChild(modalHeader);
    modalContent.appendChild(headersTextarea);

    const modalActions = document.createElement('div');
    const closeButton = document.createElement('button');
    const saveButton = document.createElement('button');

    modalActions.className = 'header-modal-actions';
    closeButton.textContent = 'Close';
    closeButton.className = 'close-headers-btn';
    closeButton.addEventListener('click', () => closeHeaderEditor(toolId));
    saveButton.textContent = 'Save';
    saveButton.className = 'save-headers-btn';
    saveButton.addEventListener('click', () => {
        try {
            const updatedHeaders = JSON.parse(headersTextarea.value);
            saveCallback(updatedHeaders);
            closeHeaderEditor(toolId);
        } catch (e) {
            alert('Invalid JSON format for headers.');
            console.error("Header JSON parse error:", e);
        }
    });

    modalActions.appendChild(closeButton);
    modalActions.appendChild(saveButton);
    modalContent.appendChild(modalActions);
    modal.appendChild(modalContent);

    // Close modal if clicked outside
    window.addEventListener('click', (event) => {
        if (event.target === modal) {
            closeHeaderEditor(toolId);
        }
    });

    return modal;
}

function openHeaderEditor(toolId) {
    const modal = document.getElementById(`header-modal-${toolId}`);
    if (modal) {
        const textarea = modal.querySelector('.headers-textarea');
        modal.style.display = 'block';
    }
}

function closeHeaderEditor(toolId) {
    const modal = document.getElementById(`header-modal-${toolId}`);
    if (modal) {
        modal.style.display = 'none';
    }
}

/**
 * Renders the tool display area.
 */
export function renderToolInterface(tool, containerElement) {
    const TOOL_ID = tool.id;
    containerElement.innerHTML = '';

    let lastResults = null;
    let currentHeaders = {
        "Content-Type": "application/json",
        "Accept": "application/json"
    };

    // function to update lastResults so we can toggle json
    const updateLastResults = (newResults) => {
        lastResults = newResults;
    };

    const updateCurrentHeaders = (newHeaders) => {
        currentHeaders = newHeaders;
        // Recreate modal with updated headers to reflect change if reopened
        const newModal = createHeaderEditorModal(TOOL_ID, currentHeaders, updateCurrentHeaders);
        containerElement.appendChild(newModal);
    };

    const gridContainer = document.createElement('div');
    gridContainer.className = 'tool-details-grid';

    const toolInfoContainer = document.createElement('div');
    const nameBox = document.createElement('div');
    const descBox = document.createElement('div');

    nameBox.className = 'tool-box tool-name';
    nameBox.innerHTML = `<h5>Name:</h5><p>${tool.name}</p>`;
    descBox.className = 'tool-box tool-description';
    descBox.innerHTML = `<h5>Description:</h5><p>${tool.description}</p>`;

    toolInfoContainer.className = 'tool-info';
    toolInfoContainer.appendChild(nameBox);
    toolInfoContainer.appendChild(descBox);
    gridContainer.appendChild(toolInfoContainer);

    const paramsContainer = document.createElement('div');
    const form = document.createElement('form');
    paramsContainer.className = 'tool-params tool-box';
    paramsContainer.innerHTML = '<h5>Parameters:</h5>';
    form.id = `tool-params-form-${TOOL_ID}`;

    tool.parameters.forEach(param => {
        form.appendChild(createParamInput(param, TOOL_ID));
    });
    paramsContainer.appendChild(form);
    gridContainer.appendChild(paramsContainer);

    containerElement.appendChild(gridContainer);

    const RESPONSE_AREA_ID = `tool-response-area-${TOOL_ID}`;
    const runButtonContainer = document.createElement('div');
    const editHeadersButton = document.createElement('button');
    const runButton = document.createElement('button');

    editHeadersButton.className = 'edit-headers-btn';
    editHeadersButton.textContent = 'Edit Headers';
    editHeadersButton.addEventListener('click', () => openHeaderEditor(TOOL_ID));
    runButtonContainer.className = 'run-button-container';
    runButtonContainer.appendChild(editHeadersButton);

    runButton.className = 'run-tool-btn';
    runButton.textContent = 'Run Tool';
    runButtonContainer.className = 'run-button-container';
    runButtonContainer.appendChild(runButton);
    containerElement.appendChild(runButtonContainer);

    // Response Area (bottom)
    const responseContainer = document.createElement('div');
    const responseHeaderControls = document.createElement('div');
    const responseHeader = document.createElement('h5');
    const responseArea = document.createElement('textarea');

    responseContainer.className = 'tool-response tool-box';
    responseHeaderControls.className = 'response-header-controls';
    responseHeader.textContent = 'Response:';
    responseHeaderControls.appendChild(responseHeader);

    // prettify box
    const PRETTIFY_ID = `prettify-${TOOL_ID}`;
    const prettifyDiv = document.createElement('div');
    const prettifyLabel = document.createElement('label');
    const prettifyCheckbox = document.createElement('input');

    prettifyDiv.className = 'prettify-container';
    prettifyLabel.setAttribute('for', PRETTIFY_ID);
    prettifyLabel.textContent = 'Prettify JSON';
    prettifyLabel.className = 'prettify-label';

    prettifyCheckbox.type = 'checkbox';
    prettifyCheckbox.id = PRETTIFY_ID;
    prettifyCheckbox.checked = true;
    prettifyCheckbox.className = 'prettify-checkbox';

    prettifyDiv.appendChild(prettifyLabel);
    prettifyDiv.appendChild(prettifyCheckbox);

    responseHeaderControls.appendChild(prettifyDiv);
    responseContainer.appendChild(responseHeaderControls);

    responseArea.id = RESPONSE_AREA_ID;
    responseArea.readOnly = true;
    responseArea.placeholder = 'Results will appear here...';
    responseArea.className = 'tool-response-area';
    responseArea.rows = 10;
    responseContainer.appendChild(responseArea);

    containerElement.appendChild(responseContainer);

    // Create and append the header editor modal
    const headerModal = createHeaderEditorModal(TOOL_ID, currentHeaders, updateCurrentHeaders);
    containerElement.appendChild(headerModal);

    prettifyCheckbox.addEventListener('change', () => {
        if (lastResults) {
            displayResults(lastResults, responseArea, prettifyCheckbox.checked);
        }
    });

    runButton.addEventListener('click', (event) => {
        event.preventDefault();
        // Pass currentHeaders to handleRunTool
        handleRunTool(TOOL_ID, form, responseArea, tool.parameters, prettifyCheckbox, updateLastResults, currentHeaders);
    });
}
