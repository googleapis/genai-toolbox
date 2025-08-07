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
function renderMainContent(containerId, idString, instructionContent) {
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
            ${instructionContent}
        </main>
    </div>
    `;

    mainContentContainer.innerHTML = contentHTML;
}

function getHomepageInstructions() {
    return `
      <div class="resource-instructions">
        <h1 class="resource-title">Welcome to Toolbox UI</h1>
        <p class="resource-intro">Toolbox UI is a built-in web interface that allows users to visually inspect and test out configured resources such as tools and toolsets. To get started, select a resource from the navigation tab to the left.</p>
        <a href="https://googleapis.github.io/genai-toolbox/how-to/use-toolbox-ui/" class="btn btn--externalDocs" target="_blank" rel="noopener noreferrer">Toolbox UI Documentation</a>
      </div>
    `;
}

function getSourceInstructions() {
    return `
      <div class="resource-instructions">
        <h1 class="resource-title">Sources</h1>
        <p class="resource-intro">To inspect a specific source, please click on one of your sources to the left.</p>

        <h2 class="resource-subtitle">What are Sources?</h2>
        <p class="resource-description">
          Sources represent your different data sources that a tool can interact with. You can define Sources as a map in the <code>sources</code> section of your <code>tools.yaml</code> file.
          Typically, a source configuration will contain any information needed to connect with and interact with the database.
        </p>

        <a href="https://googleapis.github.io/genai-toolbox/resources/sources/" class="btn btn--externalDocs" target="_blank" rel="noopener noreferrer">Sources Documentation</a>
      </div>
    `;
}

function getAuthServiceInstructions() {
    return `
      <div class="resource-instructions">
        <h1 class="resource-title">Auth Services</h1>
        <p class="resource-intro">To inspect Auth Services, please click on one of your services to the left.</p>

        <h2 class="resource-subtitle">What are Auth Services?</h2>
        <p class="resource-description">
          AuthServices represent services that handle authentication and authorization. It can primarily be used by Tools in two different ways:
                <ul>
                    <li><strong>Authorized Invocation:</strong> A tool is validated by the auth service before the call can be invoked. Toolbox will reject any calls that fail to validate.</li>
                    <li><strong>Authenticated Parameters:</strong> A parameter thats value must be replaced with a field from an OIDC claim.</li>
                </ul> 
        </p>
        <p class="resource-description">You can define Auth Services as a map in the <code>authServices</code> section of your <code>tools.yaml</code> file. <br><br></p>

        <a href="https://googleapis.github.io/genai-toolbox/resources/authservices/" class="btn btn--externalDocs" target="_blank" rel="noopener noreferrer">AuthServices Documentation</a>
      </div>
    `;
}

function getToolInstructions() {
    return `
      <div class="resource-instructions">
        <h1 class="resource-title">Tools</h1>
        <p class="resource-intro">To inspect and test a tool, please click on one of your tools to the left.</p>
        <h2 class="resource-subtitle">What are Tools?</h2>
        <p class="resource-description">
          Tools define actions an agent can take, such as running a SQL statement or interacting with a source. 
          You can define Tools as a map in the <code>tools</code> section of your <code>tools.yaml</code> file. <br><br>
          Some tools also use <strong>parameters</strong>. Parameters for each Tool will define what inputs the agent will need to provide to invoke them. 
        </p>
        <a href="https://googleapis.github.io/genai-toolbox/resources/tools/" class="btn btn--externalDocs" target="_blank" rel="noopener noreferrer">Tools Documentation</a>
      </div>
    `;
}

function getToolsetInstructions() {
    return `
      <div class="resource-instructions">
        <h1 class="resource-title">Toolsets</h1>
        <p class="resource-intro">To inspect a specific toolset, please enter the name of a toolset and press search.</p>
        <h2 class="resource-subtitle">What are Toolsets?</h2>
        <p class="resource-description">
          Toolsets define groups of tools an agent can access. You can define Toolsets as a map in the <code>toolsets</code> section of your <code>tools.yaml</code> file. Toolsets may
          only include valid tools that are also defined in your <code>tools.yaml</code> file.
        </p>
        <a href="https://googleapis.github.io/genai-toolbox/getting-started/configure/#toolsets" class="btn btn--externalDocs" target="_blank" rel="noopener noreferrer">Toolsets Documentation</a>
      </div>
    `;
}