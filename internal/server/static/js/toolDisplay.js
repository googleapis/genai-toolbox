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
 * Renders the tool display area.
 */
export function renderToolInterface(tool, containerElement) {
    const TOOL_ID = tool.id;
    containerElement.innerHTML = '';

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
    const responseContainer = document.createElement('div');
    const responseHeader = document.createElement('h5');
    const responseArea = document.createElement('textarea');

    responseHeader.textContent = 'Response:';
    responseContainer.className = 'tool-response tool-box';
    responseContainer.appendChild(responseHeader);

    responseArea.id = RESPONSE_AREA_ID;
    responseArea.readOnly = true;
    responseArea.placeholder = 'Results will appear here...';
    responseArea.rows = 10;
    responseContainer.appendChild(responseArea);

    containerElement.appendChild(responseContainer);
}
