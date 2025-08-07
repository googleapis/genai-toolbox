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

document.addEventListener('DOMContentLoaded', () => {
    const SESSION_STORAGE_KEY = 'yamlUploaderSessionData';
    const yamlFileInput = document.getElementById('yamlFileInput');
    const fileError = document.getElementById('fileError');
    const yamlOutput = document.getElementById('yamlOutput'); 
    const clearSessionButton = document.getElementById('clearSessionButton');
    const loadedFileNameDisplay = document.getElementById('loadedFileName');
    const downloadYamlButton = document.getElementById('downloadYamlButton');

    function debounce(func, delay) {
        let timeoutId;
        return (...args) => {
            clearTimeout(timeoutId);
            timeoutId = setTimeout(() => {
                func.apply(this, args);
            }, delay);
        };
    }

    function saveToSessionStorage(filename, yamlContent) {
        try {
            const data = { filename: filename, yamlContent: yamlContent };
            sessionStorage.setItem(SESSION_STORAGE_KEY, JSON.stringify(data));
            console.debug('Saved to sessionStorage');
        } catch (e) {
            console.error("Error saving to sessionStorage:", e);
            fileError.textContent = "Failed to save data to session storage.";
        }
    }

    function loadFromSessionStorage() {
        try {
            const storedData = sessionStorage.getItem(SESSION_STORAGE_KEY);
            if (storedData) {
                const data = JSON.parse(storedData);
                if (data.yamlContent) {
                    yamlOutput.value = data.yamlContent;
                }
                if (data.filename) {
                    loadedFileNameDisplay.textContent = `Current file: ${data.filename}`;
                } else {
                    loadedFileNameDisplay.textContent = "";
                }
                console.debug('Loaded from sessionStorage');
            } else {
                console.debug('No data in sessionStorage');
            }
        } catch (e) {
            console.error("Error loading from sessionStorage:", e);
            fileError.textContent = "Failed to load data from session storage.";
            sessionStorage.removeItem(SESSION_STORAGE_KEY); 
        }
    }

    function clearSessionStorage() {
        sessionStorage.removeItem(SESSION_STORAGE_KEY);
        yamlOutput.value = '';
        fileError.textContent = '';
        yamlFileInput.value = '';
        loadedFileNameDisplay.textContent = '';
        console.log("Cleared stored YAML from sessionStorage.");
    }

    if (yamlFileInput) {
        yamlFileInput.addEventListener('change', (event) => {
            const file = event.target.files[0];
            if (!file) {
                console.log('No file selected.');
                return;
            }

            const fileName = file.name;
            if (!fileName.toLowerCase().endsWith('.yaml') && !fileName.toLowerCase().endsWith('.yml')) {
                fileError.textContent = 'Invalid file type. Please upload a .yaml or .yml file.';
                yamlFileInput.value = ''; 
                return;
            }

            fileError.textContent = ''; 
            const reader = new FileReader();

            reader.onload = (e) => {
                const content = e.target.result;
                try {
                    if (typeof jsyaml === 'undefined') {
                        fileError.textContent = 'Error: js-yaml library not loaded.';
                        return;
                    }
                    const parsedYaml = jsyaml.load(content);
                    const prettyYaml = jsyaml.dump(parsedYaml, { indent: 2, lineWidth: 120, noRefs: true });

                    yamlOutput.value = prettyYaml;
                    loadedFileNameDisplay.textContent = `Current file: ${fileName}`;
                    saveToSessionStorage(fileName, prettyYaml);

                } catch (err) {
                    fileError.textContent = 'Error parsing YAML: ' + err.message;
                    console.error('YAML Parsing Error:', err);
                    yamlOutput.value = '';
                    loadedFileNameDisplay.textContent = '';
                    sessionStorage.removeItem(SESSION_STORAGE_KEY); 
                }
            };
            reader.onerror = () => {
                fileError.textContent = 'Error reading file.';
            };
            reader.readAsText(file);
        });
    }

    if (yamlOutput) {
        yamlOutput.addEventListener('input', debounce(() => {
            const currentFilename = getDownloadFilename();
            saveToSessionStorage(currentFilename, yamlOutput.value);
            console.debug("YAML content changes saved to sessionStorage.");
        }, 500));
    }

    if (clearSessionButton) {
        clearSessionButton.addEventListener('click', () => {
            if (confirm("Clear all content from the YAML editor? This cannot be undone.")) {
                clearSessionStorage();
                console.debug("User confirmed clearing session.");
            } else {
                console.debug("User cancelled clearing session.");
            }
        });
    }

    function getDownloadFilename() {
        const loadedFileName = loadedFileNameDisplay.textContent;
        let baseName = 'tools';
        if (loadedFileName && loadedFileName.startsWith('Current file: ')) {
            const originalFileName = loadedFileName.substring('Current file: '.length);
            baseName = originalFileName.replace(/\.ya?ml$/i, '');
        }
        return `${baseName}_MODIFIED.yaml`;
    }

    if (downloadYamlButton) {
        downloadYamlButton.addEventListener('click', () => {
            const yamlContent = yamlOutput.value;
            if (!yamlContent) {
                alert("Text area is empty. Nothing to download.");
                return;
            }

            const fileName = getDownloadFilename();

            const blob = new Blob([yamlContent], { type: 'application/octet-stream' });
            const url = URL.createObjectURL(blob);

            const a = document.createElement('a');
            a.style.display = 'none';
            a.href = url;
            a.download = fileName;

            document.body.appendChild(a);
            a.click();

            setTimeout(() => {
                document.body.removeChild(a);
                URL.revokeObjectURL(url);
                console.debug('Download link cleaned up');
            }, 100);
        });
    }

    loadFromSessionStorage();
});