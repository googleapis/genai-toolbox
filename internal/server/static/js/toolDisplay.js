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

    const inputCheckboxWrapper = document.createElement('div');
    const inputContainer = document.createElement('div');
    inputCheckboxWrapper.className = 'input-checkbox-wrapper';
    inputContainer.className = 'param-input-element-container';

    // Build parameter's value input box.
    const PLACEHOLDER_LABEL = param.label;
    let inputElement;
    let boolValueLabel = null;

    if (param.type === 'textarea') {
        inputElement = document.createElement('textarea');
        inputElement.rows = 3;
        inputContainer.appendChild(inputElement);
    } else if(param.type === 'checkbox') {
        inputElement = document.createElement('input');
        inputElement.type = 'checkbox';
        inputElement.title = PLACEHOLDER_LABEL;
        inputElement.checked = false;

        // handle true/false label for boolean params
        boolValueLabel = document.createElement('span');
        boolValueLabel.className = 'checkbox-bool-label';
        boolValueLabel.textContent = inputElement.checked ? ' true' : ' false';

        inputContainer.appendChild(inputElement); 
        inputContainer.appendChild(boolValueLabel); 

        inputElement.addEventListener('change', () => {
            boolValueLabel.textContent = inputElement.checked ? ' true' : ' false';
        });
    } else {
        inputElement = document.createElement('input');
        inputElement.type = param.type;
        inputContainer.appendChild(inputElement);
    }

    inputElement.id = INPUT_ID;
    inputElement.name = param.name;
    inputElement.classList.add('param-input-element');

    if (IS_AUTH_PARAM) {
        inputElement.disabled = true;
        inputElement.classList.add('auth-param-input');
        if (param.type !== 'checkbox') {
            inputElement.placeholder = param.authServices;
        }
    } else if (param.type !== 'checkbox') {
        inputElement.placeholder = PLACEHOLDER_LABEL ? PLACEHOLDER_LABEL.trim() : '';
    }
    inputCheckboxWrapper.appendChild(inputContainer);

    // create the "Include Param" checkbox
    const INCLUDE_CHECKBOX_ID = `include-${INPUT_ID}`;
    const includeContainer = document.createElement('div');
    const includeCheckbox = document.createElement('input');

    includeContainer.className = 'include-param-container';
    includeCheckbox.type = 'checkbox';
    includeCheckbox.id = INCLUDE_CHECKBOX_ID;
    includeCheckbox.name = `include-${param.name}`;
    includeCheckbox.title = 'Include this parameter'; // Add a tooltip

    // default to checked, unless it's an optional parameter
    includeCheckbox.checked = param.required;

    includeContainer.appendChild(includeCheckbox);
    inputCheckboxWrapper.appendChild(includeContainer);

    paramItem.appendChild(inputCheckboxWrapper);

    // function to update UI based on checkbox state
    const updateParamIncludedState = () => {
        const isIncluded = includeCheckbox.checked;
        if (isIncluded) {
            paramItem.classList.remove('disabled-param');
            if (!IS_AUTH_PARAM) {
                 inputElement.disabled = false;
            }
            if (boolValueLabel) {
                boolValueLabel.classList.remove('disabled');
            }
        } else {
            paramItem.classList.add('disabled-param');
            inputElement.disabled = true;
            if (boolValueLabel) {
                boolValueLabel.classList.add('disabled');
            }
        }
    };

    // add event listener to the include checkbox
    includeCheckbox.addEventListener('change', updateParamIncludedState);
    updateParamIncludedState(); 

    return paramItem;
}

/**
 * Function to create the header editor popup modal.
 * @param {string} toolId The unique identifier for the tool.
 * @param {!Object<string, string>} currentHeaders The current headers.
 * @param {function(!Object<string, string>): void} saveCallback A function to be
 *     called when the "Save" button is clicked and the headers are successfully
 *     parsed. The function receives the updated headers object as its argument.
 * @return {!HTMLDivElement} The outermost div element of the created modal.
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
    const authTokenDropdown = createAuthTokenInfoDropdown();

    modalActions.className = 'header-modal-actions';
    closeButton.textContent = 'Close';
    closeButton.className = 'btn btn--closeHeaders';
    closeButton.addEventListener('click', () => closeHeaderEditor(toolId));
    saveButton.textContent = 'Save';
    saveButton.className = 'btn btn--saveHeaders';
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
    modalContent.appendChild(authTokenDropdown);
    modal.appendChild(modalContent);

    // Close modal if clicked outside
    window.addEventListener('click', (event) => {
        if (event.target === modal) {
            closeHeaderEditor(toolId);
        }
    });

    return modal;
}

/**
 * Function to open the header popup.
 */
function openHeaderEditor(toolId) {
    const modal = document.getElementById(`header-modal-${toolId}`);
    if (modal) {
        modal.style.display = 'block';
    }
}

/**
 * Function to close the header popup.
 */
function closeHeaderEditor(toolId) {
    const modal = document.getElementById(`header-modal-${toolId}`);
    if (modal) {
        modal.style.display = 'none';
    }
}

/**
 * Creates a dropdown element showing information on how to extract Google auth tokens.
 * @return {HTMLDetailsElement} The details element representing the dropdown.
 */
function createAuthTokenInfoDropdown() {
    const details = document.createElement('details');
    const summary = document.createElement('summary');
    const content = document.createElement('div');

    details.className = 'auth-token-details';
    details.appendChild(summary);
    summary.textContent = 'How to extract Google OAuth ID Token';
    content.className = 'auth-token-content';

    // auth instruction dropdown
    const tabButtons = document.createElement('div');
    const leftTab = document.createElement('button');
    const rightTab = document.createElement('button');
    
    tabButtons.className = 'auth-tab-group';
    leftTab.className = 'auth-tab-picker active';
    leftTab.textContent = 'With Standard Account';
    leftTab.setAttribute('data-tab', 'standard');
    rightTab.className = 'auth-tab-picker';
    rightTab.textContent = 'With Service Account';
    rightTab.setAttribute('data-tab', 'service');

    tabButtons.appendChild(leftTab);
    tabButtons.appendChild(rightTab);
    content.appendChild(tabButtons);

    const tabContentContainer = document.createElement('div');
    const standardTemplate = document.getElementById('auth-token-standard-template');
    const standardAccount = document.importNode(standardTemplate.content, true).firstElementChild;
    const serviceTemplate = document.getElementById('auth-token-service-template');
    const serviceAccount = document.importNode(serviceTemplate.content, true).firstElementChild;

    tabContentContainer.appendChild(standardAccount);
    tabContentContainer.appendChild(serviceAccount);
    content.appendChild(tabContentContainer);

    // switching tabs logic
    const tabBtns = [leftTab, rightTab];
    tabBtns.forEach(btn => {
        btn.addEventListener('click', () => {
            // deactivate all buttons and contents
            tabBtns.forEach(b => b.classList.remove('active'));
            content.querySelectorAll('.auth-tab-content').forEach(c => c.classList.remove('active'));

            // activate clicked button and corresponding content
            btn.classList.add('active');
            const tabId = btn.getAttribute('data-tab');
            content.querySelector(`#auth-tab-${tabId}`).classList.add('active');
        });
    });

    details.appendChild(content);
    return details;
}

/**
 * Renders the tool display area.
 */
export function renderToolInterface(tool, containerElement) {
    const TOOL_ID = tool.id;
    containerElement.innerHTML = '';

    let lastResults = null;
    let currentHeaders = {
        "Content-Type": "application/json"
    };

    // function to update lastResults so we can toggle json
    const updateLastResults = (newResults) => {
        lastResults = newResults;
    };

    const updateCurrentHeaders = (newHeaders) => {
        currentHeaders = newHeaders;
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

    const DISLCAIMER_INFO = "*Checked parameters are sent with the value from their text field. Empty fields will be sent as an empty string. To exclude a parameter, uncheck it."
    const paramsContainer = document.createElement('div');
    const form = document.createElement('form');
    const paramsHeader = document.createElement('div');
    const disclaimerText = document.createElement('div');

    paramsContainer.className = 'tool-params tool-box';
    paramsContainer.innerHTML = '<h5>Parameters:</h5>';
    paramsHeader.className = 'params-header';
    paramsContainer.appendChild(paramsHeader);
    disclaimerText.textContent = DISLCAIMER_INFO;
    disclaimerText.className = 'params-disclaimer'; 
    paramsContainer.appendChild(disclaimerText);

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

    editHeadersButton.className = 'btn btn--editHeaders';
    editHeadersButton.textContent = 'Edit Headers';
    editHeadersButton.addEventListener('click', () => openHeaderEditor(TOOL_ID));
    runButtonContainer.className = 'run-button-container';
    runButtonContainer.appendChild(editHeadersButton);

    runButton.className = 'btn btn--run';
    runButton.textContent = 'Run Tool';
    runButtonContainer.appendChild(runButton);
    containerElement.appendChild(runButtonContainer);

    // response Area (bottom)
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

    // create and append the header editor modal
    const headerModal = createHeaderEditorModal(TOOL_ID, currentHeaders, updateCurrentHeaders);
    containerElement.appendChild(headerModal);

    prettifyCheckbox.addEventListener('change', () => {
        if (lastResults) {
            displayResults(lastResults, responseArea, prettifyCheckbox.checked);
        }
    });

    runButton.addEventListener('click', (event) => {
        event.preventDefault();
        handleRunTool(TOOL_ID, form, responseArea, tool.parameters, prettifyCheckbox, updateLastResults, currentHeaders);
    });
}

/**
 * Checks if a specific parameter is marked as included for a given tool.
 * @param {string} toolId The ID of the tool.
 * @param {string} paramName The name of the parameter.
 * @return {boolean|null} True if the parameter's include checkbox is checked,
 *                         False if unchecked, Null if the checkbox element is not found.
 */
export function isParamIncluded(toolId, paramName) {
    const inputId = `param-${toolId}-${paramName}`;
    const includeCheckboxId = `include-${inputId}`;
    const includeCheckbox = document.getElementById(includeCheckboxId);

    if (includeCheckbox && includeCheckbox.type === 'checkbox') {
        return includeCheckbox.checked;
    }

    console.warn(`Include checkbox not found for ID: ${includeCheckboxId}`);
    return null;
}