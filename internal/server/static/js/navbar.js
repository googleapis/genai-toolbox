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
 * Renders the navigation bar HTML content into the specified container element.
 * @param {string} containerId The ID of the DOM element to inject the navbar into.
 * @param {string | null} activePath The active tab from the navbar.
 */
function renderNavbar(containerId, activePath) {
    const navbarContainer = document.getElementById(containerId);
    if (!navbarContainer) {
        console.error(`Navbar container with ID "${containerId}" not found.`);
        return;
    }

    const navbarHTML = `
        <nav class="left-nav">
            <div class="nav-logo">
                <img src="/ui/assets/mcptoolboxlogo.png" alt="App Logo">
            </div>
            <ul>
                <!--<li><a href="/ui/sources">Sources</a></li>-->
                <!--<li><a href="/ui/authservices">Auth Services</a></li>-->
                <li><a href="/ui/tools">Tools</a></li>
                <li><a href="/ui/toolsets">Toolsets</a></li>
            </ul>
        </nav>
    `;

    navbarContainer.innerHTML = navbarHTML;
    if (activePath) {
        const navLinks = navbarContainer.querySelectorAll('.left-nav ul li a');
        navLinks.forEach(link => {
            const linkPath = new URL(link.href).pathname;
            if (linkPath === activePath) {
                link.classList.add('active');
            } else {
                link.classList.remove('active');
            }
        });
    }
}
