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
 * Renders the main content area into the HTML.
 * @param {string} containerId The ID of the DOM element to inject the content into.
 * @param {string} idString The id of the item inside the main content area.
 */
function renderMainContent(containerId, idString) {
    const mainContentContainer = document.getElementById(containerId);
    if (!mainContentContainer) {
        console.error(`Content container with ID "${containerId}" not found.`);
        return;
    }

    const idAttribute = idString ? `id="${idString}"` : '';
    const contentHTML = `
        <div class="main-content-area">
        <div class="top-bar">
        </div>
        <main class="content" ${idAttribute}">
            <h1>Welcome to MCP Toolbox UI</h1>
            <p>This is the main content area. Click a tab on the left to navigate.</p>
        </main>
    </div>
    `;

    mainContentContainer.innerHTML = contentHTML;
}
