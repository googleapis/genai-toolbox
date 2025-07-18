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
 * Runs a specific tool using the /api/tools/toolName/invoke endpoint
 * @param {string} toolId The unique identifier for the tool.
 * @param {!HTMLFormElement} form The form element containing parameter inputs.
 * @param {!HTMLTextAreaElement} responseArea The textarea to display results or errors.
 * @param {!Array<!Object>} parameters An array of parameter definition objects
 * @param {!HTMLInputElement} prettifyCheckbox The checkbox to control JSON formatting.
 * @param {function(?Object): void} updateLastResults Callback to store the last results.
 */
export async function handleRunTool(toolId, form, responseArea, parameters, prettifyCheckbox, updateLastResults) {
    responseArea.value = 'Running tool...';
    updateLastResults(null);
    const formData = new FormData(form);
    const typedParams = {};

    for (const param of parameters) {
        const NAME = param.name;
        const VALUE_TYPE = param.valueType;
        const RAW_VALUE = formData.get(NAME);

        try {
            if (VALUE_TYPE === 'boolean') {
                typedParams[NAME] = RAW_VALUE !== null;
                console.debug(`Parameter ${NAME} (boolean) set to: ${typedParams[NAME]}`);
                continue; 
            }

            // handle missing values for non-boolean types
            if (RAW_VALUE === null || RAW_VALUE === undefined || RAW_VALUE === '') {
                if (param.required) {
                    const errorMessage = `Error: Required parameter "${NAME}" is missing.`;
                    console.warn(errorMessage);
                    responseArea.value = errorMessage;
                    return; 
                }
                console.debug(`Optional parameter ${NAME} is missing, skipping.`);
                continue;
            }

            // process remaining non-boolean types
            if (VALUE_TYPE && VALUE_TYPE.startsWith('array<')) {
                const ELEMENT_TYPE = VALUE_TYPE.substring(6, VALUE_TYPE.length - 1);
                let parsedArray;
                try {
                    parsedArray = JSON.parse(RAW_VALUE);
                } catch (e) {
                    throw new Error(`Invalid JSON format for ${NAME}. Expected a array. ${e.message}`);
                }

                if (!Array.isArray(parsedArray)) {
                    throw new Error(`Input for ${NAME} must be a JSON array (e.g., ["a", "b"]).`);
                }

                if (ELEMENT_TYPE === 'number') {
                    typedParams[NAME] = parsedArray.map((item, index) => {
                        const num = Number(item);
                        if (isNaN(num)) {
                            throw new Error(`Invalid number "${item}" found in array for ${NAME} at index ${index}.`);
                        }
                        return num;
                    });
                } else if (ELEMENT_TYPE === 'boolean') {
                    typedParams[NAME] = parsedArray.map(item => item === true || String(item).toLowerCase() === 'true');
                } else {
                    typedParams[NAME] = parsedArray;
                }
            } else {
                switch (VALUE_TYPE) {
                    case 'number':
                        const num = Number(RAW_VALUE);
                        if (isNaN(num)) {
                            throw new Error(`Invalid number input for ${NAME}: ${RAW_VALUE}`);
                        }
                        typedParams[NAME] = num;
                        break;
                    case 'string':
                    default:
                        typedParams[NAME] = RAW_VALUE;
                        break;
                }
            }
        } catch (error) {
            console.error('Error processing parameter:', NAME, error);
            responseArea.value = `Error for ${NAME}: ${error.message}`;
            return; 
        }
    }

    console.debug('Running tool:', toolId, 'with typed params:', typedParams);
    try {
        const response = await fetch(`/api/tool/${toolId}/invoke`, {
            method: 'POST',
            headers: {'Content-Type': 'application/json'},
            body: JSON.stringify(typedParams)
        });
        if (!response.ok) {
            const errorBody = await response.text();
            throw new Error(`HTTP error ${response.status}: ${errorBody}`);
        }
        const results = await response.json();
        updateLastResults(results);
        displayResults(results, responseArea, prettifyCheckbox.checked);
    } catch (error) {
        console.error('Error running tool:', error);
        responseArea.value = `Error: ${error.message}`;
        updateLastResults(null);
    }
}

/**
 * Displays the results from the tool run in the response area.
 */
export function displayResults(results, responseArea, prettify) {
    if (results === null || results === undefined) {
        return;
    }
    try {
        const resultJson = JSON.parse(results.result);
        if (prettify) {
            responseArea.value = JSON.stringify(resultJson, null, 2);
        } else {
            responseArea.value = JSON.stringify(resultJson);
        }
    } catch (error) {
        console.error("Error parsing or stringifying results:", error);
        if (typeof results.result === 'string') {
            responseArea.value = results.result;
        } else {
            responseArea.value = "Error displaying results. Invalid format.";
        }
    }
}
