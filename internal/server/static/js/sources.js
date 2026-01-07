// Copyright 2026 Google LLC
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

import { fetchSources, fetchSource } from "./loadSources.js";
import { renderSourceDetails } from "./sourcesDisplay.js";

/**
 * These functions run after the browser finishes loading and parsing HTML structure.
 * This ensures that elements can be safely accessed.
 */
document.addEventListener("DOMContentLoaded", () => {
  const sourceDisplayArea = document.getElementById("source-display-area");
  const secondaryPanelContent = document.getElementById(
    "secondary-panel-content"
  );

  if (!secondaryPanelContent || !sourceDisplayArea) {
    console.error("Required DOM elements not found.");
    return;
  }

  loadSources(secondaryPanelContent, sourceDisplayArea);
});

/**
 * Fetches the sources and renders the list.
 * @param {!HTMLElement} secondaryPanelContent The element for the source list.
 * @param {!HTMLElement} sourceDisplayArea The element for showing source details.
 * @returns {!Promise<void>}
 */
async function loadSources(secondaryPanelContent, sourceDisplayArea) {
  secondaryPanelContent.innerHTML = "<p>Fetching sources...</p>";
  try {
    const sources = await fetchSources();
    renderSourceList(sources, secondaryPanelContent, sourceDisplayArea);
  } catch (error) {
    console.error("Failed to load sources:", error);
    secondaryPanelContent.innerHTML = `<p class="error">Failed to load sources: <pre><code>${error}</code></pre></p>`;
  }
}

/**
 * Renders the list of sources as buttons.
 * @param {!Array<{name: string, kind: string}>} sources The sources to render.
 * @param {!HTMLElement} secondaryPanelContent The element for the source list.
 * @param {!HTMLElement} sourceDisplayArea The element for showing source details.
 */
function renderSourceList(sources, secondaryPanelContent, sourceDisplayArea) {
  secondaryPanelContent.innerHTML = "";

  if (!Array.isArray(sources) || sources.length === 0) {
    secondaryPanelContent.textContent = "No sources found.";
    return;
  }

  const ul = document.createElement("ul");
  sources.forEach((source) => {
    const li = document.createElement("li");
    const button = document.createElement("button");
    button.textContent = source.name;
    button.dataset.sourcename = source.name;
    button.classList.add("tool-button");
    button.addEventListener("click", (event) =>
      handleSourceClick(event, secondaryPanelContent, sourceDisplayArea)
    );
    li.appendChild(button);
    ul.appendChild(li);
  });
  secondaryPanelContent.appendChild(ul);
}

/**
 * Handles the click event on a source button.
 * @param {!Event} event The click event object.
 * @param {!HTMLElement} secondaryPanelContent The element containing the source list.
 * @param {!HTMLElement} sourceDisplayArea The element for showing source details.
 */
async function handleSourceClick(
  event,
  secondaryPanelContent,
  sourceDisplayArea
) {
  const sourceName = event.target.dataset.sourcename;
  if (!sourceName) {
    return;
  }

  const currentActive = secondaryPanelContent.querySelector(
    ".tool-button.active"
  );
  if (currentActive) {
    currentActive.classList.remove("active");
  }
  event.target.classList.add("active");

  sourceDisplayArea.innerHTML = "<p>Loading source details...</p>";
  try {
    const source = await fetchSource(sourceName);
    renderSourceDetails(source, sourceDisplayArea);
  } catch (error) {
    console.error(`Failed to load details for source "${sourceName}":`, error);
    sourceDisplayArea.innerHTML = `<p class="error">Failed to load details for ${sourceName}. ${error.message}</p>`;
  }
}
