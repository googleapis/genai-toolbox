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

document.addEventListener('DOMContentLoaded', () => {
    const sourceTypeSelect = document.getElementById('sourceTypeSelect');
    const yamlOutputTextarea = document.getElementById('yamlOutput');
    let dbConfigData = {};

    const jsonPath = '/ui/data/source_templates.json'; 

    async function loadDbConfigs() {
        try {
            const response = await fetch(jsonPath);
            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status} fetching ${jsonPath}`);
            }
            dbConfigData = await response.json();
            populateDropdown();
        } catch (error) {
            console.error('Could not load database types:', error);
            sourceTypeSelect.disabled = true;
            let errorOption = sourceTypeSelect.querySelector('option[value=""]');
            if (errorOption) {
                errorOption.textContent = 'Failed to load types';
            }
        }
    }

    function populateDropdown() {
        if (!dbConfigData || Object.keys(dbConfigData).length === 0) {
            console.warn('No database types found in JSON data.');
            return;
        }

        Object.keys(dbConfigData).sort().forEach(dbType => {
            const option = document.createElement('option');
            option.value = dbType;
            option.textContent = dbType;
            sourceTypeSelect.appendChild(option);
        });
    }

    sourceTypeSelect.addEventListener('change', () => {
        const selectedType = sourceTypeSelect.value;
        if (!selectedType || !dbConfigData[selectedType]) {
            return;
        }

        const fields = dbConfigData[selectedType];
        if (!Array.isArray(fields)) {
            console.error(`Fields for ${selectedType} is not an array.`);
            return;
        }

        const currentYaml = yamlOutputTextarea.value;
        const lines = currentYaml.split('\n');
        const sourcesIndex = lines.findIndex(line => line.trim() === 'sources:');
        if (sourcesIndex === -1) {
            alert('The line "sources:" was not found in the YAML content. Please add it to insert a source.');
            sourceTypeSelect.value = ""; 
            return;
        }

        const NAME_INDENT = '  ';
        const FIELD_INDENT = '    ';

        let snippetLines = [];
        snippetLines.push(`${NAME_INDENT}YOUR_${selectedType.toUpperCase()}_SOURCE_NAME:`); 
        fields.forEach(field => {
            snippetLines.push(`${FIELD_INDENT}${field}:`);
        });

        lines.splice(sourcesIndex + 1, 0, ...snippetLines);
        yamlOutputTextarea.value = lines.join('\n');
        yamlOutputTextarea.scrollTop = 0;
        sourceTypeSelect.value = "";
    });

    loadDbConfigs();
});
