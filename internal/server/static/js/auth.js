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
 * Renders the Google Sign-In button using the GIS library.
 * @param {string} toolId The ID of the tool.
 * @param {string} clientId The Google OAuth Client ID.
 * @param {string} authProfileName The name of the auth service in tools file.
 */
function renderGoogleSignInButton(toolId, clientId, authProfileName) { 
    const UNIQUE_ID_BASE = `${toolId}-${authProfileName}`;
    const GIS_CONTAINER_ID = `gisContainer-${UNIQUE_ID_BASE}`;
    const gisContainer = document.getElementById(GIS_CONTAINER_ID);
    const setupGisBtn = document.querySelector(`#google-auth-details-${UNIQUE_ID_BASE} .btn--setup-gis`);

    if (!gisContainer) {
        console.error('GIS container not found:', GIS_CONTAINER_ID);
        return;
    }

    if (!clientId) {
        alert('Please enter an OAuth Client ID first.');
        return;
    }

    // hide the setup button and show the container for the GIS button
    if (setupGisBtn) setupGisBtn.style.display = 'none';
    gisContainer.innerHTML = ''; 
    gisContainer.style.display = 'flex'; 
    if (window.google && window.google.accounts && window.google.accounts.id) {
        try {
            const handleResponse = (response) => handleCredentialResponse(response, toolId, authProfileName);
            window.google.accounts.id.initialize({
                client_id: clientId,
                callback: handleResponse,
                auto_select: false
            });
            window.google.accounts.id.renderButton(
                gisContainer,
                { theme: "outline", size: "large", text: "signin_with" }
            );
        } catch (error) {
            console.error("Error initializing Google Sign-In:", error);
            alert("Error initializing Google Sign-In. Check the Client ID and browser console.");
            gisContainer.innerHTML = '<p style="color: red;">Error loading Sign-In button.</p>';
            if (setupGisBtn) setupGisBtn.style.display = ''; 
        }
    } else {
        console.error("GIS library not fully loaded yet.");
        alert("Google Identity Services library not ready. Please try again in a moment.");
        gisContainer.innerHTML = '<p style="color: red;">GIS library not ready.</p>';
        if (setupGisBtn) setupGisBtn.style.display = ''; 
    }
}

/**
 * Handles the credential response from the Google Sign-In library.
 * @param {!CredentialResponse} response The credential response object from GIS.
 * @param {string} toolId The ID of the tool.
 * @param {string} authProfileName The name of the auth service in tools file.
 */
function handleCredentialResponse(response, toolId, authProfileName) { 
    console.debug("handleCredentialResponse called with:", { response, toolId, authProfileName });
    const headersTextarea = document.getElementById(`headers-textarea-${toolId}`);
    if (!headersTextarea) {
        console.error('Headers textarea not found for toolId:', toolId);
        return;
    }

    const UNIQUE_ID_BASE = `${toolId}-${authProfileName}`;
    const setupGisBtn = document.querySelector(`#google-auth-details-${UNIQUE_ID_BASE} .setup-gis-btn`);
    const gisContainer = document.getElementById(`gisContainer-${UNIQUE_ID_BASE}`);

    if (response.credential) {
        const idToken = response.credential;

        try {
            let currentHeaders = {};
            if (headersTextarea.value) {
                currentHeaders = JSON.parse(headersTextarea.value);
            }
            const HEADER_KEY = `${authProfileName}_token`; 
            currentHeaders[HEADER_KEY] = `${idToken}`;
            headersTextarea.value = JSON.stringify(currentHeaders, null, 2);

            if (gisContainer) gisContainer.style.display = 'none';
            if (setupGisBtn) setupGisBtn.style.display = '';

        } catch (e) {
            alert('Headers are not valid JSON. Please correct and try again.');
            console.error("Header JSON parse error:", e);
        }
    } else {
        console.error("Error: No credential in response", response);
        alert('Error: No ID Token received. Check console for details.');
        
        if (gisContainer) gisContainer.style.display = 'none';
        if (setupGisBtn) setupGisBtn.style.display = '';
    }
}

// creates the Google Auth method dropdown
export function createGoogleAuthMethodItem(toolId, authProfileName) { 
    const UNIQUE_ID_BASE = `${toolId}-${authProfileName}`;
    const item = document.createElement('div');

    item.className = 'auth-method-item';
    item.innerHTML = `
        <div class="auth-method-header">
            <span class="auth-method-label">Google ID Token (${authProfileName})</span>
            <button class="toggle-details-tab">Auto Setup</button>
        </div>
        <div class="auth-method-details" id="google-auth-details-${UNIQUE_ID_BASE}" style="display: none;">
            <div class="auth-controls">
                <div class="auth-input-row">
                    <label for="clientIdInput-${UNIQUE_ID_BASE}">OAuth Client ID:</label>
                    <input type="text" id="clientIdInput-${UNIQUE_ID_BASE}" placeholder="Enter Client ID" class="auth-input">
                </div>
                <div class="auth-instructions">
                    <strong>Action Required:</strong> Add this URL (e.g., http://localhost:PORT) to the Client ID's <strong>Authorized JavaScript origins</strong> to avoid a 401 error. If using Google OAuth, 
                    navigate to <a href="https://console.cloud.google.com/apis/credentials" target="_blank">Google Cloud Console</a> for this setting.
                </div>
                <div class="auth-method-actions">
                    <button class="btn btn--setup-gis">Continue</button>
                    <div id="gisContainer-${UNIQUE_ID_BASE}" class="auth-interactive-element gis-container" style="display: none;"></div>
                </div>
            </div>
        </div>
    `;

    const toggleBtn = item.querySelector('.toggle-details-tab');
    const detailsDiv = item.querySelector(`#google-auth-details-${UNIQUE_ID_BASE}`);
    const setupGisBtn = item.querySelector('.btn--setup-gis');
    const clientIdInput = item.querySelector(`#clientIdInput-${UNIQUE_ID_BASE}`);
    const gisContainer = item.querySelector(`#gisContainer-${UNIQUE_ID_BASE}`);

    toggleBtn.addEventListener('click', () => {
        const isVisible = detailsDiv.style.display === 'flex'; 
        detailsDiv.style.display = isVisible ? 'none' : 'flex'; 
        toggleBtn.textContent = isVisible ? 'Auto Setup' : 'Close';
        if (!isVisible) { 
            if (gisContainer) {
                gisContainer.innerHTML = '';
                gisContainer.style.display = 'none';
            }
            if (setupGisBtn) {
                setupGisBtn.style.display = ''; 
            }
        }
    });

    setupGisBtn.addEventListener('click', () => {
        const clientId = clientIdInput.value;
        if (!clientId) {
            alert('Please enter an OAuth Client ID first.');
            return;
        }
        renderGoogleSignInButton(toolId, clientId, authProfileName);
    });

    return item;
}
