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

import { loadTools } from "./loadTools.js";

document.addEventListener('DOMContentLoaded', () => {
    const searchInput = document.getElementById('toolset-search-input');
    const searchButton = document.getElementById('toolset-search-button');
    const secondNavContent = document.getElementById('secondary-panel-content');
    const toolDisplayArea = document.getElementById('tool-display-area');

    if (!searchInput || !searchButton || !secondNavContent || !toolDisplayArea) {
        console.error('Required DOM elements not found.');
        return;
    }

    // Event listener for search button click
    searchButton.addEventListener('click', () => {
        const toolsetName = searchInput.value.trim();
        if (toolsetName) {
            loadTools(secondNavContent, toolDisplayArea, toolsetName)
        } else {
            secondNavContent.innerHTML = '<p>Please enter a toolset name to search.</p>';
        }
    });

    // Event listener for Enter key in search input
    searchInput.addEventListener('keypress', (event) => {
        if (event.key === 'Enter') {
            const toolsetName = searchInput.value.trim();
            if (toolsetName) {
                loadTools(secondNavContent, toolDisplayArea, toolsetName);
            } else {
                secondNavContent.innerHTML = '<p>Please enter a toolset name to search.</p>';
            }
        }
    });
})