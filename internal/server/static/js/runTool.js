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

import { isParamIncluded } from "./toolDisplay.js";

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
    const formData = new FormData(form);
    const typedParams = {};
    responseArea.value = 'Running tool...';
    updateLastResults(null);

    for (const param of parameters) {
        const NAME = param.name;
        const VALUE_TYPE = param.valueType;
        const RAW_VALUE = formData.get(NAME);
        const INCLUDE_CHECKED = isParamIncluded(toolId, NAME)

        try {
            if (!INCLUDE_CHECKED) {
                console.debug(`Param ${NAME} was intentionally skipped.`)
                // if param was purposely unchecked, don't include it in body
                continue;
            }

            if (VALUE_TYPE === 'boolean') {
                typedParams[NAME] = RAW_VALUE !== null;
                console.debug(`Parameter ${NAME} (boolean) set to: ${typedParams[NAME]}`);
                continue; 
            }

            // process remaining types
            if (VALUE_TYPE && VALUE_TYPE.startsWith('array<')) {
                typedParams[NAME] = parseArrayParameter(RAW_VALUE, VALUE_TYPE, NAME);
            } else {
                switch (VALUE_TYPE) {
                    case 'number':
                        if (RAW_VALUE === "") {
                            console.debug(`Param ${NAME} was empty, setting to empty string.`)
                            typedParams[NAME] = "";
                        } else {
                            const num = Number(RAW_VALUE);
                            if (isNaN(num)) {
                                throw new Error(`Invalid number input for ${NAME}: ${RAW_VALUE}`);
                            }
                            typedParams[NAME] = num;
                        }
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
 * Parses and validates a single array parameter from a raw string value.
 * @param {string} rawValue The raw string value from FormData.
 * @param {string} valueType The full array type string (e.g., "array<number>").
 * @param {string} paramName The name of the parameter for error messaging.
 * @return {!Array<*>} The parsed array.
 * @throws {Error} If parsing or type validation fails.
 */
function parseArrayParameter(rawValue, valueType, paramName) {
    const ELEMENT_TYPE = valueType.substring(6, valueType.length - 1);
    let parsedArray;
    try {
        parsedArray = JSON.parse(rawValue);
    } catch (e) {
        throw new Error(`Invalid JSON format for ${paramName}. Expected an array. ${e.message}`);
    }

    if (!Array.isArray(parsedArray)) {
        throw new Error(`Input for ${paramName} must be a JSON array (e.g., ["a", "b"]).`);
    }

    return parsedArray.map((item, index) => {
        switch (ELEMENT_TYPE) {
            case 'number':
                const NUM = Number(item);
                if (isNaN(NUM)) {
                    throw new Error(`Invalid number "${item}" found in array for ${paramName} at index ${index}.`);
                }
                return NUM;
            case 'boolean':
                return item === true || String(item).toLowerCase() === 'true';
            case 'string':
            default:
                return item;
        }
    });
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
