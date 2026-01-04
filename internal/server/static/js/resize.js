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

const STORAGE_KEY = 'toolbox-second-nav-width';
const DEFAULT_WIDTH = 250;
const MIN_WIDTH = 200;
const MAX_WIDTH_PERCENT = 50;

/**
 * Creates and attaches a resize handle to the second navigation panel
 */
export function initializeResize() {
    const secondNav = document.querySelector('.second-nav');
    if (!secondNav) {
        return;
    }

    // Create resize handle
    const resizeHandle = document.createElement('div');
    resizeHandle.className = 'resize-handle';
    resizeHandle.setAttribute('aria-label', 'Resize panel');
    secondNav.appendChild(resizeHandle);

    // Load saved width or use default
    const savedWidth = localStorage.getItem(STORAGE_KEY);
    const initialWidth = savedWidth ? parseInt(savedWidth, 10) : DEFAULT_WIDTH;
    setPanelWidth(secondNav, initialWidth);

    // Setup resize functionality
    let isResizing = false;
    let startX = 0;
    let startWidth = 0;

    resizeHandle.addEventListener('mousedown', (e) => {
        isResizing = true;
        startX = e.clientX;
        startWidth = secondNav.offsetWidth;
        resizeHandle.classList.add('active');
        document.body.style.cursor = 'ew-resize';
        document.body.style.userSelect = 'none';
        e.preventDefault();
    });

    document.addEventListener('mousemove', (e) => {
        if (!isResizing) {
            return;
        }

        const deltaX = e.clientX - startX;
        const newWidth = startWidth + deltaX;
        const maxWidth = (window.innerWidth * MAX_WIDTH_PERCENT) / 100;

        const clampedWidth = Math.max(MIN_WIDTH, Math.min(newWidth, maxWidth));
        setPanelWidth(secondNav, clampedWidth);
    });

    document.addEventListener('mouseup', () => {
        if (isResizing) {
            isResizing = false;
            resizeHandle.classList.remove('active');
            document.body.style.cursor = '';
            document.body.style.userSelect = '';
            
            // Save width to localStorage
            localStorage.setItem(STORAGE_KEY, secondNav.offsetWidth.toString());
        }
    });

    // Handle window resize to enforce max width
    window.addEventListener('resize', () => {
        const currentWidth = secondNav.offsetWidth;
        const maxWidth = (window.innerWidth * MAX_WIDTH_PERCENT) / 100;
        
        if (currentWidth > maxWidth) {
            setPanelWidth(secondNav, maxWidth);
            localStorage.setItem(STORAGE_KEY, maxWidth.toString());
        }
    });
}

/**
 * Sets the width of the panel and updates flex property
 */
function setPanelWidth(panel, width) {
    panel.style.flex = `0 0 ${width}px`;
    panel.style.width = `${width}px`;
}

