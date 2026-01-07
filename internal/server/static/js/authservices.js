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

import { fetchAuthServices, fetchAuthService } from "./loadAuthServices.js";
import { renderAuthServiceDetails } from "./authservicesDisplay.js";

/**
 * These functions run after the browser finishes loading and parsing HTML structure.
 * This ensures that elements can be safely accessed.
 */
document.addEventListener('DOMContentLoaded', () => {
    const authServiceDisplayArea = document.getElementById('authservice-display-area');
    const secondaryPanelContent = document.getElementById('secondary-panel-content');

    if (!secondaryPanelContent || !authServiceDisplayArea) {
        console.error('Required DOM elements not found.');
        return;
    }

    loadAuthServices(secondaryPanelContent, authServiceDisplayArea);
});

/**
 * Fetches the auth services and renders the list.
 * @param {!HTMLElement} secondaryPanelContent The element for the auth service list.
 * @param {!HTMLElement} authServiceDisplayArea The element for showing auth service details.
 * @returns {!Promise<void>}
 */
async function loadAuthServices(secondaryPanelContent, authServiceDisplayArea) {
    secondaryPanelContent.innerHTML = '<p>Fetching auth services...</p>';
    try {
        const services = await fetchAuthServices();
        renderAuthServiceList(services, secondaryPanelContent, authServiceDisplayArea);
    } catch (error) {
        console.error('Failed to load auth services:', error);
        secondaryPanelContent.innerHTML = `<p class="error">Failed to load auth services: <pre><code>${error}</code></pre></p>`;
    }
}

/**
 * Renders the list of auth services as buttons.
 * @param {!Array<{name: string, kind: string}>} services The auth services to render.
 * @param {!HTMLElement} secondaryPanelContent The element for the auth service list.
 * @param {!HTMLElement} authServiceDisplayArea The element for showing auth service details.
 */
function renderAuthServiceList(services, secondaryPanelContent, authServiceDisplayArea) {
    secondaryPanelContent.innerHTML = '';

    if (!Array.isArray(services) || services.length === 0) {
        secondaryPanelContent.textContent = 'No auth services found.';
        return;
    }

    const ul = document.createElement('ul');
    services.forEach(service => {
        const li = document.createElement('li');
        const button = document.createElement('button');
        button.textContent = service.name;
        button.dataset.authservicename = service.name;
        button.classList.add('tool-button');
        button.addEventListener('click', (event) => handleAuthServiceClick(event, secondaryPanelContent, authServiceDisplayArea));
        li.appendChild(button);
        ul.appendChild(li);
    });
    secondaryPanelContent.appendChild(ul);
}

/**
 * Handles the click event on an auth service button.
 * @param {!Event} event The click event object.
 * @param {!HTMLElement} secondaryPanelContent The element containing the auth service list.
 * @param {!HTMLElement} authServiceDisplayArea The element for showing auth service details.
 */
async function handleAuthServiceClick(event, secondaryPanelContent, authServiceDisplayArea) {
    const authServiceName = event.target.dataset.authservicename;
    if (!authServiceName) {
        return;
    }

    const currentActive = secondaryPanelContent.querySelector('.tool-button.active');
    if (currentActive) {
        currentActive.classList.remove('active');
    }
    event.target.classList.add('active');

    authServiceDisplayArea.innerHTML = '<p>Loading auth service details...</p>';
    try {
        const service = await fetchAuthService(authServiceName);
        renderAuthServiceDetails(service, authServiceDisplayArea);
    } catch (error) {
        console.error(`Failed to load details for auth service "${authServiceName}":`, error);
        authServiceDisplayArea.innerHTML = `<p class="error">Failed to load details for ${authServiceName}. ${error.message}</p>`;
    }
}
