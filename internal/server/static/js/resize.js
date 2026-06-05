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
    let initialWidth = DEFAULT_WIDTH;
    try {
        const savedWidth = localStorage.getItem(STORAGE_KEY);
        if (savedWidth) {
            const parsed = parseInt(savedWidth, 10);
            if (!isNaN(parsed) && parsed >= MIN_WIDTH) {
                initialWidth = parsed;
            }
        }
    } catch (e) {
        // localStorage may be unavailable in private browsing mode
        console.warn('Failed to load saved panel width:', e);
    }
    setPanelWidth(secondNav, initialWidth);

    // Setup resize functionality
    let startX = 0;
    let startWidth = 0;

    const onMouseMove = (e) => {
        const deltaX = e.clientX - startX;
        const newWidth = startWidth + deltaX;
        const maxWidth = (window.innerWidth * MAX_WIDTH_PERCENT) / 100;

        const clampedWidth = Math.max(MIN_WIDTH, Math.min(newWidth, maxWidth));
        setPanelWidth(secondNav, clampedWidth);
    };

    const onMouseUp = () => {
        document.removeEventListener('mousemove', onMouseMove);
        document.removeEventListener('mouseup', onMouseUp);

        resizeHandle.classList.remove('active');
        document.body.style.cursor = '';
        document.body.style.userSelect = '';

        // Save width to localStorage
        try {
            localStorage.setItem(STORAGE_KEY, secondNav.offsetWidth.toString());
        } catch (e) {
            // localStorage may be unavailable in private browsing mode
            console.warn('Failed to save panel width:', e);
        }
    };

    resizeHandle.addEventListener('mousedown', (e) => {
        startX = e.clientX;
        startWidth = secondNav.offsetWidth;
        resizeHandle.classList.add('active');
        document.body.style.cursor = 'ew-resize';
        document.body.style.userSelect = 'none';
        e.preventDefault();

        document.addEventListener('mousemove', onMouseMove);
        document.addEventListener('mouseup', onMouseUp);
    });

    // Handle window resize to enforce max width
    window.addEventListener('resize', () => {
        const currentWidth = secondNav.offsetWidth;
        const maxWidth = (window.innerWidth * MAX_WIDTH_PERCENT) / 100;
        
        if (currentWidth > maxWidth) {
            setPanelWidth(secondNav, maxWidth);
            try {
                localStorage.setItem(STORAGE_KEY, maxWidth.toString());
            } catch (e) {
                // localStorage may be unavailable in private browsing mode
                console.warn('Failed to save panel width:', e);
            }
        }
    });
}

/**
 * Sets the width of the panel and updates flex property
 */
function setPanelWidth(panel, width) {
    panel.style.flex = `0 0 ${width}px`;
}

