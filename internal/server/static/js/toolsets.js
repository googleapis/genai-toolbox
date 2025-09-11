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

import { renderToolList } from "./loadTools.js";

document.addEventListener('DOMContentLoaded', () => {
    const searchInput = document.getElementById('toolset-search-input');
    const searchButton = document.getElementById('toolset-search-button');
    const secondNavContent = document.getElementById('secondary-panel-content');
    const toolDisplayArea = document.getElementById('tool-display-area');
    const suggestionsPanel = document.getElementById('suggestions-panel');
    const selectedToolsetsContainer = document.getElementById('selected-toolsets-container');

    if (!searchInput || !searchButton || !secondNavContent || !toolDisplayArea || !suggestionsPanel || !selectedToolsetsContainer) {
        console.error('Required DOM elements not found.');
        return;
    }

    let availableToolsets = [];
    let selectedToolsets = [];
    const loadedToolsets = {};

    // Fetch toolset names from the API
    const fetchToolsetNames = async () => {
        try {
            const response = await fetch('/api/toolset-names');
            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }
            availableToolsets = await response.json();
        } catch (error) {
            console.error('Error fetching toolset names:', error);
        }
    };
    fetchToolsetNames();

    // Render selected toolsets
    const renderSelectedToolsets = () => {
        selectedToolsetsContainer.innerHTML = '';
        selectedToolsets.forEach((name, index) => {
            const pill = document.createElement('div');
            pill.classList.add('toolset-pill');

            const numberIndicator = document.createElement('span');
            numberIndicator.classList.add('number-indicator');
            numberIndicator.textContent = index + 1;
            pill.appendChild(numberIndicator);

            const nameSpan = document.createElement('span');
            nameSpan.textContent = name;
            pill.appendChild(nameSpan);

            const removeBtn = document.createElement('span');
            removeBtn.classList.add('remove-btn');
            removeBtn.textContent = 'x';
            removeBtn.addEventListener('click', () => removeToolset(name));
            pill.appendChild(removeBtn);

            selectedToolsetsContainer.appendChild(pill);
        });
    };

    // Add a toolset to the selected list
    const addToolset = async (name) => {
        if (name && !selectedToolsets.includes(name) && availableToolsets.includes(name)) {
            if (!loadedToolsets[name]) {
                try {
                    const response = await fetch(`/api/toolset/${name}`);
                    if (!response.ok) {
                        throw new Error(`HTTP error! status: ${response.status}`);
                    }
                    const apiResponse = await response.json();
                    loadedToolsets[name] = apiResponse.tools;
                } catch (error) {
                    console.error(`Failed to load toolset ${name}:`, error);
                    return; 
                }
            }
            selectedToolsets.push(name);
            updateDisplayedTools();
            searchInput.value = '';
        }
    };

    // Remove a toolset from the selected list
    const removeToolset = (name) => {
        selectedToolsets = selectedToolsets.filter(t => t !== name);
        updateDisplayedTools();
    };

    // Update the displayed tools based on the selected toolsets
    const updateDisplayedTools = () => {
        const activeToolButton = secondNavContent.querySelector('.tool-button.active');
        const activeToolName = activeToolButton ? activeToolButton.dataset.toolname : null;

        renderSelectedToolsets();
        const toolsetsToRender = selectedToolsets.map((name) => ({
            name: name,
            tools: loadedToolsets[name],
        })).filter(ts => ts.tools);

        if (toolsetsToRender.length > 0) {
            renderToolList(toolsetsToRender, secondNavContent, toolDisplayArea);
        } else {
            secondNavContent.innerHTML = '<p>Please enter a toolset name to see available tools. <br><br>To view the default toolset that consists of all tools, please select the "Tools" tab.</p>';
            toolDisplayArea.innerHTML = '';
        }

        if (activeToolName) {
            const newActiveButton = secondNavContent.querySelector(`[data-toolname="${activeToolName}"]`);
            if (newActiveButton) {
                newActiveButton.classList.add('active');
            }
        }
    };

    // Function to display suggestions
    const showSuggestions = (input) => {
        suggestionsPanel.innerHTML = '';
        if (input.length === 0) {
            suggestionsPanel.style.display = 'none';
            return;
        }

        const filteredNames = availableToolsets.filter(name =>
            name.toLowerCase().includes(input.toLowerCase()) && !selectedToolsets.includes(name)
        );

        if (filteredNames.length > 0) {
            filteredNames.forEach(name => {
                const suggestionItem = document.createElement('div');
                suggestionItem.textContent = name;
                suggestionItem.classList.add('suggestion-item');
                suggestionItem.addEventListener('click', () => {
                    addToolset(name);
                    suggestionsPanel.style.display = 'none';
                });
                suggestionsPanel.appendChild(suggestionItem);
            });
            suggestionsPanel.style.display = 'block';
        } else {
            suggestionsPanel.style.display = 'none';
        }
    };

    // Event listener for search input
    searchInput.addEventListener('input', () => {
        const inputValue = searchInput.value.trim();
        showSuggestions(inputValue);
    });

    // Event listener for search button click
    searchButton.addEventListener('click', () => {
        const toolsetName = searchInput.value.trim();
        addToolset(toolsetName);
        suggestionsPanel.style.display = 'none';
    });

    // Event listener for Enter key in search input
    searchInput.addEventListener('keypress', (event) => {
        if (event.key === 'Enter') {
            const toolsetName = searchInput.value.trim();
            addToolset(toolsetName);
            suggestionsPanel.style.display = 'none';
        }
    });

    // Hide suggestions when clicking outside
    document.addEventListener('click', (event) => {
        if (!suggestionsPanel.contains(event.target) && event.target !== searchInput) {
            suggestionsPanel.style.display = 'none';
        }
    });
});
