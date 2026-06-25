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

// State Management
let resources = [];
let currentTab = 'sources';
let editingIndex = null; // null if creating a new resource, index if editing

// Default Template Resources to initialize the builder
const DEFAULT_RESOURCES = [
  {
    kind: 'source',
    name: 'test-sqlite',
    type: 'sqlite',
    database: 'tests/conformance/test.db'
  },
  {
    kind: 'tool',
    name: 'test_simple_text',
    type: 'sqlite-sql',
    source: 'test-sqlite',
    description: 'Test simple text',
    statement: "SELECT 'This is a simple text response for testing.' AS text;"
  },
  {
    kind: 'prompt',
    name: 'test_prompt_with_arguments',
    description: 'Test prompt with arguments',
    messages: [
      {
        role: 'user',
        content: "Prompt with arguments: arg1='{{.arg1}}', arg2='{{.arg2}}'"
      }
    ],
    arguments: [
      {
        name: 'arg1',
        description: 'First test argument',
        required: true
      },
      {
        name: 'arg2',
        description: 'Second test argument',
        required: false
      }
    ]
  }
];

// Initialize application
document.addEventListener('DOMContentLoaded', () => {
  resources = [...DEFAULT_RESOURCES];
  setupEventListeners();
  switchTab('sources');
  renderYaml();
});

// Setup Event Listeners
function setupEventListeners() {
  // Tab Switching
  document.querySelectorAll('.tab-btn').forEach(btn => {
    btn.addEventListener('click', (e) => {
      const tab = e.target.getAttribute('data-tab');
      switchTab(tab);
    });
  });

  // Add Resource Button
  document.getElementById('btn-add-resource').addEventListener('click', () => {
    openCreateForm();
  });

  // Form Cancel Button
  document.getElementById('btn-cancel-edit').addEventListener('click', () => {
    closeForm();
  });

  // Form Delete Active Button
  document.getElementById('btn-delete-active').addEventListener('click', () => {
    deleteActiveResource();
  });

  // Form Submit
  document.getElementById('editor-form').addEventListener('submit', (e) => {
    e.preventDefault();
    saveActiveResource();
  });

  // Reset / Clear All
  document.getElementById('btn-clear-all').addEventListener('click', () => {
    if (confirm('Are you sure you want to clear all resources and reset?')) {
      resources = [];
      closeForm();
      renderResourceList();
      renderYaml();
    }
  });

  // Copy YAML
  document.getElementById('btn-copy-yaml').addEventListener('click', () => {
    const yamlCode = document.getElementById('yaml-code').innerText;
    navigator.clipboard.writeText(yamlCode).then(() => {
      const btn = document.getElementById('btn-copy-yaml');
      btn.innerText = 'Copied!';
      btn.classList.add('bg-zinc-100', 'text-zinc-900');
      setTimeout(() => {
        btn.innerText = 'Copy';
        btn.classList.remove('bg-zinc-100', 'text-zinc-900');
      }, 1500);
    });
  });

  // Download YAML
  document.getElementById('btn-download-yaml').addEventListener('click', () => {
    const yamlCode = document.getElementById('yaml-code').innerText;
    const blob = new Blob([yamlCode], { type: 'text/yaml;charset=utf-8;' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.setAttribute('download', 'tools.yaml');
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
  });

  // Import Modal Controls
  const importModal = document.getElementById('import-modal');
  document.getElementById('btn-import-modal').addEventListener('click', () => {
    document.getElementById('text-import').value = '';
    document.getElementById('selected-filename').innerText = 'No file chosen';
    document.getElementById('import-error').classList.add('hidden');
    importModal.classList.remove('hidden');
  });

  document.getElementById('btn-close-import').addEventListener('click', () => {
    importModal.classList.add('hidden');
  });

  document.getElementById('btn-cancel-import').addEventListener('click', () => {
    importModal.classList.add('hidden');
  });

  // File Selector for Import
  const fileImport = document.getElementById('file-import');
  document.getElementById('btn-file-select').addEventListener('click', () => {
    fileImport.click();
  });

  fileImport.addEventListener('change', (e) => {
    const file = e.target.files[0];
    if (file) {
      document.getElementById('selected-filename').innerText = file.name;
      const reader = new FileReader();
      reader.onload = (evt) => {
        document.getElementById('text-import').value = evt.target.result;
      };
      reader.readAsText(file);
    }
  });

  // Confirm Import
  document.getElementById('btn-confirm-import').addEventListener('click', () => {
    const yamlContent = document.getElementById('text-import').value.trim();
    if (!yamlContent) {
      document.getElementById('import-error').innerText = 'Content is empty';
      document.getElementById('import-error').classList.remove('hidden');
      return;
    }

    try {
      // parse using jsyaml
      const documents = jsyaml.loadAll(yamlContent);
      
      // Validation & Mapping
      const imported = [];
      documents.forEach((doc, idx) => {
        if (!doc) return;
        if (!doc.kind) {
          throw new Error(`Document #${idx + 1} is missing a "kind" property.`);
        }
        if (!doc.name) {
          throw new Error(`Document #${idx + 1} is missing a "name" property.`);
        }
        // Normalize kinds to match tab state mapping (camelCase plural)
        imported.push(doc);
      });

      resources = imported;
      importModal.classList.add('hidden');
      closeForm();
      switchTab(currentTab);
      renderYaml();
    } catch (err) {
      const errElem = document.getElementById('import-error');
      errElem.innerText = `Error: ${err.message}`;
      errElem.classList.remove('hidden');
    }
  });
}

// Switch resource type tabs
function switchTab(tab) {
  currentTab = tab;
  
  // Update nav buttons style
  document.querySelectorAll('.tab-btn').forEach(btn => {
    const active = btn.getAttribute('data-tab') === tab;
    if (active) {
      btn.className = "tab-btn px-3 py-2 text-left font-mono font-medium rounded-md border border-zinc-200 bg-[#E1F3FE] text-[#1F6C9F] transition-all cursor-pointer";
    } else {
      btn.className = "tab-btn px-3 py-2 text-left font-mono font-medium rounded-md border border-transparent bg-zinc-50 text-zinc-600 hover:bg-zinc-100 hover:text-zinc-900 active:scale-[0.98] transition-all cursor-pointer";
    }
  });

  // Update List Title
  const titles = {
    sources: 'Sources',
    authServices: 'Auth Services',
    tools: 'Tools',
    toolsets: 'Toolsets',
    embeddingModels: 'Embedding Models',
    prompts: 'Prompts'
  };
  document.getElementById('list-title').innerText = titles[tab] || tab;

  renderResourceList();
}

// Render left column resource items
function renderResourceList() {
  const listContainer = document.getElementById('resource-list');
  listContainer.innerHTML = '';

  const filtered = resources
    .map((res, index) => ({ res, originalIndex: index }))
    .filter(item => getPluralKind(item.res.kind) === currentTab);

  if (filtered.length === 0) {
    listContainer.innerHTML = `
      <div class="text-center py-8 text-zinc-400 font-mono text-[11px]">
        No ${currentTab} defined.
      </div>
    `;
    return;
  }

  filtered.forEach(item => {
    const wrapper = document.createElement('div');
    const isActive = editingIndex === item.originalIndex;
    
    wrapper.className = `group flex items-center justify-between p-2.5 rounded-md border border-transparent transition-all cursor-pointer ${
      isActive 
        ? 'bg-white border-zinc-200 shadow-sm' 
        : 'hover:bg-zinc-50 hover:border-zinc-200/40'
    }`;

    // Item details
    const textDiv = document.createElement('div');
    textDiv.className = 'flex-1 min-w-0 pr-2';
    textDiv.addEventListener('click', () => {
      loadResourceIntoForm(item.originalIndex);
    });

    const name = document.createElement('div');
    name.className = 'text-xs font-mono font-medium text-zinc-800 truncate';
    name.innerText = item.res.name;

    const sub = document.createElement('div');
    sub.className = 'text-[10px] text-zinc-400 font-mono mt-0.5';
    sub.innerText = item.res.type || item.res.kind;

    textDiv.appendChild(name);
    textDiv.appendChild(sub);

    // Delete spot action
    const delBtn = document.createElement('button');
    delBtn.className = 'text-zinc-300 hover:text-red-700 hover:bg-[#FDEBEC] p-1 rounded opacity-0 group-hover:opacity-100 focus:opacity-100 transition-all cursor-pointer';
    delBtn.setAttribute('title', 'Delete item');
    delBtn.innerHTML = `
      <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
        <path stroke-linecap="round" stroke-linejoin="round" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
      </svg>
    `;
    delBtn.addEventListener('click', (e) => {
      e.stopPropagation();
      if (confirm(`Delete ${item.res.name}?`)) {
        resources.splice(item.originalIndex, 1);
        if (editingIndex === item.originalIndex) {
          closeForm();
        } else if (editingIndex > item.originalIndex) {
          editingIndex--;
        }
        renderResourceList();
        renderYaml();
      }
    });

    wrapper.appendChild(textDiv);
    wrapper.appendChild(delBtn);
    listContainer.appendChild(wrapper);
  });
}

// Convert model kinds to plural form tab naming
function getPluralKind(kind) {
  const mapping = {
    source: 'sources',
    authService: 'authServices',
    tool: 'tools',
    toolset: 'toolsets',
    embeddingModel: 'embeddingModels',
    prompt: 'prompts'
  };
  return mapping[kind] || kind;
}

// Convert plurals back to singular kind
function getSingularKind(tab) {
  const mapping = {
    sources: 'source',
    authServices: 'authService',
    tools: 'tool',
    toolsets: 'toolset',
    embeddingModels: 'embeddingModel',
    prompts: 'prompt'
  };
  return mapping[tab] || tab;
}

// Open Form to Create New Resource
function openCreateForm() {
  editingIndex = null;
  const kind = getSingularKind(currentTab);
  
  // Set header
  document.getElementById('form-badge-kind').innerText = kind.toUpperCase();
  document.getElementById('form-badge-kind').className = `px-2 py-0.5 text-[9px] font-mono font-semibold uppercase tracking-wider rounded ${getKindColorClass(kind)}`;
  document.getElementById('form-title-context').innerText = 'new_resource';
  document.getElementById('form-header-title').innerText = `Add New ${formatKindName(kind)}`;
  
  // Hide delete button for new records
  document.getElementById('btn-delete-active').classList.add('hidden');

  buildFormFields(kind, {});
  
  document.getElementById('welcome-view').classList.add('hidden');
  document.getElementById('editor-form').classList.remove('hidden');
  
  // Highlight nothing in list
  renderResourceList();
}

// Open Form to Edit Existing Resource
function loadResourceIntoForm(index) {
  editingIndex = index;
  const res = resources[index];
  const kind = res.kind;

  // Set header
  document.getElementById('form-badge-kind').innerText = kind.toUpperCase();
  document.getElementById('form-badge-kind').className = `px-2 py-0.5 text-[9px] font-mono font-semibold uppercase tracking-wider rounded ${getKindColorClass(kind)}`;
  document.getElementById('form-title-context').innerText = res.name;
  document.getElementById('form-header-title').innerText = `Edit ${formatKindName(kind)}`;

  // Show delete button
  document.getElementById('btn-delete-active').classList.remove('hidden');

  buildFormFields(kind, res);

  document.getElementById('welcome-view').classList.add('hidden');
  document.getElementById('editor-form').classList.remove('hidden');

  // Highlight list item
  renderResourceList();
}

// Close form and return to welcome view
function closeForm() {
  editingIndex = null;
  document.getElementById('welcome-view').classList.remove('hidden');
  document.getElementById('editor-form').classList.add('hidden');
  renderResourceList();
}

// Delete Active Resource
function deleteActiveResource() {
  if (editingIndex !== null) {
    if (confirm(`Are you sure you want to delete this resource?`)) {
      resources.splice(editingIndex, 1);
      closeForm();
      renderYaml();
    }
  }
}

// Color badges for different kinds
function getKindColorClass(kind) {
  const classes = {
    source: 'bg-[#E1F3FE] text-[#1F6C9F]',
    authService: 'bg-[#FBF3DB] text-[#956400]',
    tool: 'bg-[#EDF3EC] text-[#346538]',
    toolset: 'bg-zinc-100 text-zinc-700',
    embeddingModel: 'bg-purple-50 text-purple-700 border-purple-200',
    prompt: 'bg-red-50 text-red-700 border-red-200'
  };
  return classes[kind] || 'bg-zinc-100 text-zinc-700';
}

function formatKindName(kind) {
  const names = {
    source: 'Source',
    authService: 'Auth Service',
    tool: 'Tool',
    toolset: 'Toolset',
    embeddingModel: 'Embedding Model',
    prompt: 'Prompt'
  };
  return names[kind] || kind;
}

// Save/Apply active form edits
function saveActiveResource() {
  const kind = getSingularKind(currentTab);
  const name = document.getElementById('field-name').value.trim();

  if (!name) {
    alert('Name is required.');
    return;
  }

  // Build resource object from form values
  const obj = { kind, name };

  if (kind === 'source') {
    obj.type = document.getElementById('field-source-type').value;
    const optionalFields = [
      'host', 'port', 'database', 'user', 'password', 'queryParams',
      'project', 'region', 'cluster', 'instance', 'ipType', 'defaultProject'
    ];
    optionalFields.forEach(f => {
      const elem = document.getElementById(`field-${f}`);
      if (elem && elem.parentElement.style.display !== 'none') {
        const val = elem.value.trim();
        if (val) {
          if (f === 'port') {
            obj[f] = parseInt(val, 10);
          } else {
            obj[f] = val;
          }
        }
      }
    });
  } 
  
  else if (kind === 'authService') {
    obj.type = document.getElementById('field-auth-type').value;
    
    // scopesRequired
    const scopes = [];
    document.querySelectorAll('.scope-input-row').forEach(row => {
      const val = row.querySelector('input').value.trim();
      if (val) scopes.push(val);
    });
    if (scopes.length > 0) {
      obj.scopesRequired = scopes;
    }

    obj.mcpEnabled = document.getElementById('field-mcp-enabled').checked;

    // clientOAuth
    if (document.getElementById('chk-enable-oauth').checked) {
      obj.clientOAuth = {
        clientId: document.getElementById('field-oauth-client-id').value.trim(),
        clientSecret: document.getElementById('field-oauth-client-secret').value.trim(),
        authorizationUri: document.getElementById('field-oauth-auth-uri').value.trim(),
        tokenUri: document.getElementById('field-oauth-token-uri').value.trim(),
        redirectUri: document.getElementById('field-oauth-redirect-uri').value.trim()
      };
      // Clean up empty fields in nested object
      Object.keys(obj.clientOAuth).forEach(k => {
        if (!obj.clientOAuth[k]) delete obj.clientOAuth[k];
      });
      if (Object.keys(obj.clientOAuth).length === 0) {
        delete obj.clientOAuth;
      }
    }
  } 
  
  else if (kind === 'tool') {
    obj.type = document.getElementById('field-tool-type').value.trim();
    obj.source = document.getElementById('field-tool-source').value;
    obj.description = document.getElementById('field-tool-desc').value.trim();
    
    const statement = document.getElementById('field-tool-statement').value.trim();
    if (statement) {
      obj.statement = statement;
    }

    // Parameters
    const params = getParametersFromForm('.param-row');
    if (params.length > 0) obj.parameters = params;

    // Template Parameters
    const tparams = getParametersFromForm('.tparam-row');
    if (tparams.length > 0) obj.templateParameters = tparams;
  } 
  
  else if (kind === 'toolset') {
    const selectedTools = [];
    document.querySelectorAll('.toolset-checkbox:checked').forEach(cb => {
      selectedTools.push(cb.value);
    });
    obj.tools = selectedTools;
  } 
  
  else if (kind === 'embeddingModel') {
    obj.type = document.getElementById('field-embed-type').value;
    const modelId = document.getElementById('field-embed-model-id').value.trim();
    if (modelId) obj.modelId = modelId;
    
    const project = document.getElementById('field-embed-project').value.trim();
    if (project) obj.project = project;

    const region = document.getElementById('field-embed-region').value.trim();
    if (region) obj.region = region;
  } 
  
  else if (kind === 'prompt') {
    obj.description = document.getElementById('field-prompt-desc').value.trim();

    // Messages
    const messages = [];
    document.querySelectorAll('.message-row').forEach(row => {
      const role = row.querySelector('.msg-role').value;
      const content = row.querySelector('.msg-content').value.trim();
      if (content) {
        messages.push({ role, content });
      }
    });
    if (messages.length > 0) obj.messages = messages;

    // Arguments
    const args = [];
    document.querySelectorAll('.arg-row').forEach(row => {
      const argName = row.querySelector('.arg-name').value.trim();
      const argDesc = row.querySelector('.arg-desc').value.trim();
      const argReq = row.querySelector('.arg-req').checked;
      if (argName) {
        args.push({
          name: argName,
          description: argDesc || undefined,
          required: argReq || undefined
        });
      }
    });
    if (args.length > 0) obj.arguments = args;
  }

  // Update resources array
  if (editingIndex !== null) {
    resources[editingIndex] = obj;
  } else {
    resources.push(obj);
    editingIndex = resources.length - 1;
  }

  renderResourceList();
  renderYaml();
  
  // Re-load form to reflect clean changes
  loadResourceIntoForm(editingIndex);
}

// Helper to gather parameter forms
function getParametersFromForm(selector) {
  const params = [];
  document.querySelectorAll(selector).forEach(row => {
    const name = row.querySelector('.param-name').value.trim();
    const type = row.querySelector('.param-type').value;
    const desc = row.querySelector('.param-desc').value.trim();
    const def = row.querySelector('.param-default').value.trim();
    const req = row.querySelector('.param-req').checked;

    if (name) {
      const p = {
        name,
        type,
        description: desc || undefined,
        required: req || undefined
      };

      if (def) {
        // cast if number/boolean
        if (type === 'integer') p.default = parseInt(def, 10);
        else if (type === 'number') p.default = parseFloat(def);
        else if (type === 'boolean') p.default = (def.toLowerCase() === 'true');
        else p.default = def;
      }

      // items if array
      if (type === 'array') {
        const itemType = row.querySelector('.param-item-type').value;
        const itemDesc = row.querySelector('.param-item-desc').value.trim();
        p.items = {
          type: itemType,
          description: itemDesc || undefined
        };
      }

      // authServices
      const authRows = row.querySelectorAll('.param-auth-row');
      const authSvcs = [];
      authRows.forEach(ar => {
        const sName = ar.querySelector('.param-auth-svc').value;
        const sField = ar.querySelector('.param-auth-field').value.trim();
        if (sName && sField) {
          authSvcs.push({ name: sName, field: sField });
        }
      });
      if (authSvcs.length > 0) {
        p.authServices = authSvcs;
      }

      params.push(p);
    }
  });
  return params;
}

// Generate the forms in HTML dynamically
function buildFormFields(kind, data) {
  const container = document.getElementById('form-fields');
  container.innerHTML = '';

  // Every kind has a name field
  const nameGroup = createFormGroup('Name', 'field-name', 'text', data.name || '', true, 'Unique identifier for this resource.');
  container.appendChild(nameGroup);

  if (kind === 'source') {
    // Source Type selector
    const sourceTypes = [
      'sqlite', 'alloydb-postgres', 'postgres', 'mysql', 'cloud-monitoring',
      'alloydb-admin', 'spanner', 'bigquery', 'clickhouse', 'firestore',
      'cloud-storage', 'mssql', 'neo4j', 'oceanbase', 'oracledb', 'snowflake'
    ];
    const typeSelect = createSelectGroup('Source Type', 'field-source-type', sourceTypes, data.type || 'postgres');
    container.appendChild(typeSelect);

    // Create container for connection fields to show/hide based on source type
    const connFieldsContainer = document.createElement('div');
    connFieldsContainer.className = 'space-y-4 pt-2 border-t border-zinc-100';
    connFieldsContainer.id = 'conn-fields-container';
    
    // Add fields
    connFieldsContainer.appendChild(createFormGroup('Host', 'field-host', 'text', data.host || '', false));
    connFieldsContainer.appendChild(createFormGroup('Port', 'field-port', 'number', data.port || '', false));
    connFieldsContainer.appendChild(createFormGroup('Database / Connection String', 'field-database', 'text', data.database || '', false));
    connFieldsContainer.appendChild(createFormGroup('User', 'field-user', 'text', data.user || '', false));
    connFieldsContainer.appendChild(createFormGroup('Password', 'field-password', 'password', data.password || '', false));
    connFieldsContainer.appendChild(createFormGroup('Query Parameters', 'field-queryParams', 'text', data.queryParams || '', false, 'E.g. sslmode=disable'));
    
    // GCP fields
    connFieldsContainer.appendChild(createFormGroup('GCP Project ID', 'field-project', 'text', data.project || '', false));
    connFieldsContainer.appendChild(createFormGroup('Region', 'field-region', 'text', data.region || '', false));
    connFieldsContainer.appendChild(createFormGroup('Cluster ID', 'field-cluster', 'text', data.cluster || '', false));
    connFieldsContainer.appendChild(createFormGroup('Instance ID', 'field-instance', 'text', data.instance || '', false));
    connFieldsContainer.appendChild(createSelectGroup('IP Connection Type', 'field-ipType', ['public', 'private'], data.ipType || 'public'));
    
    // Admin fields
    connFieldsContainer.appendChild(createFormGroup('Default Project', 'field-defaultProject', 'text', data.defaultProject || '', false));

    container.appendChild(connFieldsContainer);

    // Connect source type event listener
    const selectEl = typeSelect.querySelector('select');
    selectEl.addEventListener('change', () => toggleSourceFields(selectEl.value));
    toggleSourceFields(data.type || 'postgres');
  } 
  
  else if (kind === 'authService') {
    // Auth Type
    const authTypes = ['google', 'generic'];
    const typeSelect = createSelectGroup('Auth Service Type', 'field-auth-type', authTypes, data.type || 'google');
    container.appendChild(typeSelect);

    // Scopes Required dynamic list
    const scopeContainer = document.createElement('div');
    scopeContainer.className = 'space-y-2';
    scopeContainer.innerHTML = `
      <label class="text-[10px] tracking-wider font-mono font-semibold uppercase text-zinc-500 mb-1 block">Scopes Required</label>
      <div id="scopes-list" class="space-y-1.5"></div>
      <button type="button" id="btn-add-scope" class="mt-1 px-2.5 py-1 text-[10px] font-mono border border-zinc-200 bg-white hover:bg-zinc-50 rounded active:scale-[0.97] transition-all cursor-pointer">
        + Add Scope
      </button>
    `;
    container.appendChild(scopeContainer);

    const scopesList = scopeContainer.querySelector('#scopes-list');
    scopeContainer.querySelector('#btn-add-scope').addEventListener('click', () => {
      addScopeRow(scopesList, '');
    });

    if (data.scopesRequired && Array.isArray(data.scopesRequired)) {
      data.scopesRequired.forEach(s => addScopeRow(scopesList, s));
    } else {
      addScopeRow(scopesList, '');
    }

    // mcpEnabled
    const mcpGroup = createCheckboxGroup('MCP Enabled', 'field-mcp-enabled', data.mcpEnabled !== false);
    container.appendChild(mcpGroup);

    // OAuth Expandable Section
    const oauthSection = document.createElement('div');
    oauthSection.className = 'pt-4 border-t border-zinc-100 space-y-4';
    
    const chkEnableOAuth = createCheckboxGroup('Enable Client OAuth Configuration', 'chk-enable-oauth', !!data.clientOAuth);
    oauthSection.appendChild(chkEnableOAuth);

    const oauthFields = document.createElement('div');
    oauthFields.id = 'oauth-fields';
    oauthFields.className = 'space-y-4 pl-4 border-l border-zinc-200/60 hidden';
    
    const oauthData = data.clientOAuth || {};
    oauthFields.appendChild(createFormGroup('Client ID', 'field-oauth-client-id', 'text', oauthData.clientId || '', false));
    oauthFields.appendChild(createFormGroup('Client Secret', 'field-oauth-client-secret', 'text', oauthData.clientSecret || '', false));
    oauthFields.appendChild(createFormGroup('Authorization URI', 'field-oauth-auth-uri', 'text', oauthData.authorizationUri || '', false));
    oauthFields.appendChild(createFormGroup('Token URI', 'field-oauth-token-uri', 'text', oauthData.tokenUri || '', false));
    oauthFields.appendChild(createFormGroup('Redirect URI', 'field-oauth-redirect-uri', 'text', oauthData.redirectUri || '', false));

    oauthSection.appendChild(oauthFields);
    container.appendChild(oauthSection);

    const oauthToggle = chkEnableOAuth.querySelector('input');
    oauthToggle.addEventListener('change', () => {
      oauthFields.style.display = oauthToggle.checked ? 'block' : 'none';
    });
    oauthFields.style.display = data.clientOAuth ? 'block' : 'none';
  } 
  
  else if (kind === 'tool') {
    // Tool type (SQL query or specific driver call)
    const toolType = createFormGroup('Tool Type / Exec Kind', 'field-tool-type', 'text', data.type || 'postgres-sql', true, 'E.g. postgres-sql, sqlite-sql, postgres-list-tables');
    container.appendChild(toolType);

    // Source dropdown
    const definedSources = resources.filter(r => r.kind === 'source').map(r => r.name);
    const sourceSelect = createSelectGroup('Associated Source', 'field-tool-source', definedSources, data.source || '');
    container.appendChild(sourceSelect);

    if (definedSources.length === 0) {
      const alert = document.createElement('div');
      alert.className = 'p-2.5 rounded bg-[#FBF3DB] text-[#956400] text-[11px] font-mono';
      alert.innerText = 'Warning: No database sources configured. Please configure a Source first so this tool has a target.';
      container.appendChild(alert);
    }

    // Description
    const descGroup = createTextareaGroup('Description', 'field-tool-desc', data.description || '', 'Explain what the LLM can achieve with this tool.');
    container.appendChild(descGroup);

    // Statement / Query text
    const statementGroup = createTextareaGroup('Statement / SQL Code', 'field-tool-statement', data.statement || '', 'The query statement or command template to run on the database.', true);
    statementGroup.querySelector('textarea').className = 'w-full h-32 font-mono text-xs p-3 border border-zinc-200 rounded-md focus:outline-none focus:ring-1 focus:ring-zinc-400 focus:border-zinc-400 placeholder-zinc-300 bg-zinc-50/50';
    container.appendChild(statementGroup);

    // Parameters Container
    const paramSection = document.createElement('div');
    paramSection.className = 'pt-4 border-t border-zinc-100 space-y-4';
    paramSection.innerHTML = `
      <div class="flex items-center justify-between">
        <label class="text-[10px] tracking-wider font-mono font-semibold uppercase text-zinc-500">Parameters</label>
        <button type="button" id="btn-add-param" class="px-2.5 py-1 text-[10px] font-mono border border-zinc-200 bg-white hover:bg-zinc-50 rounded active:scale-[0.97] transition-all cursor-pointer">
          + Add Parameter
        </button>
      </div>
      <div id="params-list" class="space-y-4"></div>
    `;
    container.appendChild(paramSection);

    const paramsList = paramSection.querySelector('#params-list');
    paramSection.querySelector('#btn-add-param').addEventListener('click', () => {
      addParameterRow(paramsList, 'param-row', {});
    });

    if (data.parameters && Array.isArray(data.parameters)) {
      data.parameters.forEach(p => addParameterRow(paramsList, 'param-row', p));
    }

    // Template Parameters Container
    const tparamSection = document.createElement('div');
    tparamSection.className = 'pt-4 border-t border-zinc-100 space-y-4';
    tparamSection.innerHTML = `
      <div class="flex items-center justify-between">
        <label class="text-[10px] tracking-wider font-mono font-semibold uppercase text-zinc-500">Template Parameters (e.g. {{.name}})</label>
        <button type="button" id="btn-add-tparam" class="px-2.5 py-1 text-[10px] font-mono border border-zinc-200 bg-white hover:bg-zinc-50 rounded active:scale-[0.97] transition-all cursor-pointer">
          + Add Template Param
        </button>
      </div>
      <div id="tparams-list" class="space-y-4"></div>
    `;
    container.appendChild(tparamSection);

    const tparamsList = tparamSection.querySelector('#tparams-list');
    tparamSection.querySelector('#btn-add-tparam').addEventListener('click', () => {
      addParameterRow(tparamsList, 'tparam-row', {});
    });

    if (data.templateParameters && Array.isArray(data.templateParameters)) {
      data.templateParameters.forEach(tp => addParameterRow(tparamsList, 'tparam-row', tp));
    }
  } 
  
  else if (kind === 'toolset') {
    // List tools as checkboxes
    const toolsLabel = document.createElement('label');
    toolsLabel.className = 'text-[10px] tracking-wider font-mono font-semibold uppercase text-zinc-500 mb-2 block';
    toolsLabel.innerText = 'Include Tools';
    container.appendChild(toolsLabel);

    const toolCheckboxes = document.createElement('div');
    toolCheckboxes.className = 'border border-zinc-200 rounded-md p-3 max-h-60 overflow-y-auto space-y-2 bg-zinc-50/20';

    const definedTools = resources.filter(r => r.kind === 'tool');

    if (definedTools.length === 0) {
      toolCheckboxes.innerHTML = `<span class="text-xs text-zinc-400 font-mono italic">No tools defined yet</span>`;
    } else {
      definedTools.forEach(t => {
        const item = document.createElement('label');
        item.className = 'flex items-center gap-2.5 text-xs text-zinc-700 cursor-pointer p-1 hover:bg-white rounded transition-colors';
        
        const cb = document.createElement('input');
        cb.type = 'checkbox';
        cb.value = t.name;
        cb.className = 'toolset-checkbox rounded border-zinc-300 text-zinc-900 focus:ring-zinc-400';
        if (data.tools && data.tools.includes(t.name)) {
          cb.checked = true;
        }

        const span = document.createElement('span');
        span.className = 'font-mono';
        span.innerText = t.name;

        item.appendChild(cb);
        item.appendChild(span);
        toolCheckboxes.appendChild(item);
      });
    }
    container.appendChild(toolCheckboxes);
  } 
  
  else if (kind === 'embeddingModel') {
    // Type Select
    const modelTypes = ['gemini', 'custom'];
    const typeSelect = createSelectGroup('Model Service Type', 'field-embed-type', modelTypes, data.type || 'gemini');
    container.appendChild(typeSelect);

    container.appendChild(createFormGroup('Model ID', 'field-embed-model-id', 'text', data.modelId || 'text-embedding-004', false));
    container.appendChild(createFormGroup('GCP Project ID', 'field-embed-project', 'text', data.project || '', false));
    container.appendChild(createFormGroup('Region', 'field-embed-region', 'text', data.region || 'us-central1', false));
  } 
  
  else if (kind === 'prompt') {
    // Description
    const descGroup = createTextareaGroup('Description', 'field-prompt-desc', data.description || '', 'Purpose of the prompt template.');
    container.appendChild(descGroup);

    // Messages
    const msgSection = document.createElement('div');
    msgSection.className = 'pt-4 border-t border-zinc-100 space-y-4';
    msgSection.innerHTML = `
      <div class="flex items-center justify-between">
        <label class="text-[10px] tracking-wider font-mono font-semibold uppercase text-zinc-500">Messages</label>
        <button type="button" id="btn-add-message" class="px-2.5 py-1 text-[10px] font-mono border border-zinc-200 bg-white hover:bg-zinc-50 rounded active:scale-[0.97] transition-all cursor-pointer">
          + Add Message
        </button>
      </div>
      <div id="messages-list" class="space-y-4"></div>
    `;
    container.appendChild(msgSection);

    const messagesList = msgSection.querySelector('#messages-list');
    msgSection.querySelector('#btn-add-message').addEventListener('click', () => {
      addMessageRow(messagesList, {});
    });

    if (data.messages && Array.isArray(data.messages)) {
      data.messages.forEach(m => addMessageRow(messagesList, m));
    } else {
      addMessageRow(messagesList, { role: 'user', content: '' });
    }

    // Arguments
    const argSection = document.createElement('div');
    argSection.className = 'pt-4 border-t border-zinc-100 space-y-4';
    argSection.innerHTML = `
      <div class="flex items-center justify-between">
        <label class="text-[10px] tracking-wider font-mono font-semibold uppercase text-zinc-500">Arguments</label>
        <button type="button" id="btn-add-arg" class="px-2.5 py-1 text-[10px] font-mono border border-zinc-200 bg-white hover:bg-zinc-50 rounded active:scale-[0.97] transition-all cursor-pointer">
          + Add Argument
        </button>
      </div>
      <div id="args-list" class="space-y-4"></div>
    `;
    container.appendChild(argSection);

    const argsList = argSection.querySelector('#args-list');
    argSection.querySelector('#btn-add-arg').addEventListener('click', () => {
      addArgumentRow(argsList, {});
    });

    if (data.arguments && Array.isArray(data.arguments)) {
      data.arguments.forEach(arg => addArgumentRow(argsList, arg));
    }
  }
}

// Show/Hide fields depending on Source Type
function toggleSourceFields(type) {
  const container = document.getElementById('conn-fields-container');
  if (!container) return;

  const show = (fieldId) => {
    const el = document.getElementById(fieldId);
    if (el) el.parentElement.style.display = 'block';
  };
  
  const hide = (fieldId) => {
    const el = document.getElementById(fieldId);
    if (el) el.parentElement.style.display = 'none';
  };

  // Hide all initially
  const fields = [
    'field-host', 'field-port', 'field-database', 'field-user', 'field-password',
    'field-queryParams', 'field-project', 'field-region', 'field-cluster',
    'field-instance', 'field-ipType', 'field-defaultProject'
  ];
  fields.forEach(hide);

  if (type === 'sqlite') {
    show('field-database');
  } 
  else if (type === 'alloydb-postgres') {
    show('field-project');
    show('field-region');
    show('field-cluster');
    show('field-instance');
    show('field-database');
    show('field-user');
    show('field-password');
    show('field-ipType');
  } 
  else if (type === 'postgres' || type === 'mysql' || type === 'mssql' || type === 'oracledb' || type === 'clickhouse' || type === 'snowflake' || type === 'oceanbase' || type === 'neo4j') {
    show('field-host');
    show('field-port');
    show('field-database');
    show('field-user');
    show('field-password');
    show('field-queryParams');
  } 
  else if (type === 'alloydb-admin') {
    show('field-defaultProject');
  } 
  else if (type === 'spanner') {
    show('field-project');
    show('field-instance');
    show('field-database');
  }
  else if (type === 'bigquery') {
    show('field-project');
    show('field-database');
  }
  else if (type === 'firestore' || type === 'cloud-storage') {
    show('field-project');
  }
  else if (type === 'cloud-monitoring') {
    // No fields required
  }
}

// Add Scope row in AuthService
function addScopeRow(container, val) {
  const row = document.createElement('div');
  row.className = 'scope-input-row flex items-center gap-2';
  row.innerHTML = `
    <input type="text" value="${val}" placeholder="e.g. https://www.googleapis.com/auth/cloud-platform" class="flex-1 bg-white border border-zinc-200 rounded-md px-3 py-1.5 text-xs text-zinc-800 focus:outline-none focus:ring-1 focus:ring-zinc-400 focus:border-zinc-400" />
    <button type="button" class="btn-remove-scope p-1.5 text-zinc-400 hover:text-red-700 hover:bg-[#FDEBEC] rounded transition-all cursor-pointer">
      <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" /></svg>
    </button>
  `;
  row.querySelector('.btn-remove-scope').addEventListener('click', () => {
    row.remove();
  });
  container.appendChild(row);
}

// Add Message row in Prompts
function addMessageRow(container, data) {
  const row = document.createElement('div');
  row.className = 'message-row p-3 border border-zinc-200/60 rounded-lg space-y-3 bg-zinc-50/20';
  row.innerHTML = `
    <div class="flex items-center justify-between">
      <select class="msg-role bg-white border border-zinc-200 rounded px-2 py-1 text-xs text-zinc-800 focus:outline-none focus:ring-1 focus:ring-zinc-400 focus:border-zinc-400">
        <option value="user" ${data.role === 'user' ? 'selected' : ''}>user</option>
        <option value="system" ${data.role === 'system' ? 'selected' : ''}>system</option>
        <option value="assistant" ${data.role === 'assistant' ? 'selected' : ''}>assistant</option>
      </select>
      <button type="button" class="btn-remove-message p-1 text-zinc-400 hover:text-red-700 hover:bg-[#FDEBEC] rounded transition-all cursor-pointer">
        <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" /></svg>
      </button>
    </div>
    <textarea placeholder="Message content..." class="msg-content w-full h-20 text-xs p-2.5 border border-zinc-200 rounded-md focus:outline-none focus:ring-1 focus:ring-zinc-400 focus:border-zinc-400 bg-white">${data.content || ''}</textarea>
  `;
  row.querySelector('.btn-remove-message').addEventListener('click', () => {
    row.remove();
  });
  container.appendChild(row);
}

// Add Argument row in Prompts
function addArgumentRow(container, data) {
  const row = document.createElement('div');
  row.className = 'arg-row p-3 border border-zinc-200/60 rounded-lg space-y-3 bg-zinc-50/20';
  row.innerHTML = `
    <div class="flex items-center justify-between">
      <span class="text-[10px] font-mono font-medium text-zinc-500">Argument Config</span>
      <button type="button" class="btn-remove-arg p-1 text-zinc-400 hover:text-red-700 hover:bg-[#FDEBEC] rounded transition-all cursor-pointer">
        <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" /></svg>
      </button>
    </div>
    <div class="grid grid-cols-2 gap-3">
      <div>
        <label class="text-[9px] font-mono text-zinc-400 uppercase">Argument Name</label>
        <input type="text" value="${data.name || ''}" placeholder="arg_name" class="arg-name w-full bg-white border border-zinc-200 rounded px-2.5 py-1.5 text-xs text-zinc-800 focus:outline-none focus:ring-1 focus:ring-zinc-400 focus:border-zinc-400" />
      </div>
      <div class="flex items-end pb-2">
        <label class="flex items-center gap-2 text-xs text-zinc-600 cursor-pointer">
          <input type="checkbox" class="arg-req rounded border-zinc-300 text-zinc-900 focus:ring-zinc-400" ${data.required ? 'checked' : ''} />
          Required argument
        </label>
      </div>
    </div>
    <div>
      <label class="text-[9px] font-mono text-zinc-400 uppercase">Description</label>
      <input type="text" value="${data.description || ''}" placeholder="Explain the role of this argument" class="arg-desc w-full bg-white border border-zinc-200 rounded px-2.5 py-1.5 text-xs text-zinc-800 focus:outline-none focus:ring-1 focus:ring-zinc-400 focus:border-zinc-400" />
    </div>
  `;
  row.querySelector('.btn-remove-arg').addEventListener('click', () => {
    row.remove();
  });
  container.appendChild(row);
}

// Add Parameter card in Tool
function addParameterRow(container, className, data) {
  const card = document.createElement('div');
  card.className = `${className} p-4 border border-zinc-200/80 rounded-lg space-y-4 bg-zinc-50/20`;

  const definedAuthServices = resources.filter(r => r.kind === 'authService').map(r => r.name);

  card.innerHTML = `
    <div class="flex items-center justify-between border-b border-zinc-200/50 pb-2">
      <span class="text-[10px] font-mono font-medium text-zinc-500">Parameter Configuration</span>
      <button type="button" class="btn-remove-param p-1 text-zinc-400 hover:text-red-700 hover:bg-[#FDEBEC] rounded transition-all cursor-pointer">
        <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" /></svg>
      </button>
    </div>
    
    <div class="grid grid-cols-2 gap-3">
      <div>
        <label class="text-[9px] font-mono text-zinc-400 uppercase">Param Name</label>
        <input type="text" value="${data.name || ''}" class="param-name w-full bg-white border border-zinc-200 rounded px-2.5 py-1.5 text-xs text-zinc-800 focus:outline-none focus:ring-1 focus:ring-zinc-400" />
      </div>
      <div>
        <label class="text-[9px] font-mono text-zinc-400 uppercase">Type</label>
        <select class="param-type w-full bg-white border border-zinc-200 rounded px-2.5 py-1.5 text-xs text-zinc-800 focus:outline-none focus:ring-1 focus:ring-zinc-400">
          <option value="string" ${data.type === 'string' ? 'selected' : ''}>string</option>
          <option value="integer" ${data.type === 'integer' ? 'selected' : ''}>integer</option>
          <option value="number" ${data.type === 'number' ? 'selected' : ''}>number</option>
          <option value="boolean" ${data.type === 'boolean' ? 'selected' : ''}>boolean</option>
          <option value="array" ${data.type === 'array' ? 'selected' : ''}>array</option>
          <option value="object" ${data.type === 'object' ? 'selected' : ''}>object</option>
        </select>
      </div>
    </div>

    <div class="grid grid-cols-2 gap-3">
      <div>
        <label class="text-[9px] font-mono text-zinc-400 uppercase">Default Value</label>
        <input type="text" value="${data.default !== undefined ? data.default : ''}" placeholder="Optional" class="param-default w-full bg-white border border-zinc-200 rounded px-2.5 py-1.5 text-xs text-zinc-800 focus:outline-none focus:ring-1" />
      </div>
      <div class="flex items-end pb-2">
        <label class="flex items-center gap-2 text-xs text-zinc-600 cursor-pointer">
          <input type="checkbox" class="param-req rounded border-zinc-300 text-zinc-900 focus:ring-zinc-400" ${data.required ? 'checked' : ''} />
          Required parameter
        </label>
      </div>
    </div>

    <div>
      <label class="text-[9px] font-mono text-zinc-400 uppercase">Description</label>
      <input type="text" value="${data.description || ''}" placeholder="Help the LLM understand this parameter" class="param-desc w-full bg-white border border-zinc-200 rounded px-2.5 py-1.5 text-xs text-zinc-800 focus:outline-none focus:ring-1" />
    </div>

    <!-- Array items config (Shows only if type === 'array') -->
    <div class="param-array-config border-t border-dashed border-zinc-200 pt-3 hidden space-y-3">
      <span class="text-[9px] font-mono font-medium text-zinc-400 uppercase block">Array Item Specification</span>
      <div class="grid grid-cols-2 gap-3">
        <div>
          <label class="text-[9px] font-mono text-zinc-400 uppercase">Item Type</label>
          <select class="param-item-type w-full bg-white border border-zinc-200 rounded px-2.5 py-1.5 text-xs text-zinc-800 focus:outline-none">
            <option value="string" ${data.items && data.items.type === 'string' ? 'selected' : ''}>string</option>
            <option value="integer" ${data.items && data.items.type === 'integer' ? 'selected' : ''}>integer</option>
            <option value="number" ${data.items && data.items.type === 'number' ? 'selected' : ''}>number</option>
            <option value="boolean" ${data.items && data.items.type === 'boolean' ? 'selected' : ''}>boolean</option>
          </select>
        </div>
        <div>
          <label class="text-[9px] font-mono text-zinc-400 uppercase">Item Description</label>
          <input type="text" value="${data.items && data.items.description ? data.items.description : ''}" placeholder="Optional" class="param-item-desc w-full bg-white border border-zinc-200 rounded px-2.5 py-1.5 text-xs text-zinc-800 focus:outline-none" />
        </div>
      </div>
    </div>

    <!-- Auth Service mapping -->
    <div class="border-t border-dashed border-zinc-200 pt-3 space-y-2">
      <div class="flex items-center justify-between">
        <span class="text-[9px] font-mono font-medium text-zinc-400 uppercase">Linked Auth Services</span>
        <button type="button" class="btn-add-param-auth px-2 py-0.5 text-[9px] font-mono border border-zinc-200 bg-white hover:bg-zinc-50 rounded cursor-pointer">
          + Add Auth Link
        </button>
      </div>
      <div class="param-auth-list space-y-1.5"></div>
    </div>
  `;

  // Attach controls
  const typeSelect = card.querySelector('.param-type');
  const arrayConfig = card.querySelector('.param-array-config');
  typeSelect.addEventListener('change', () => {
    arrayConfig.style.display = typeSelect.value === 'array' ? 'block' : 'none';
  });
  arrayConfig.style.display = data.type === 'array' ? 'block' : 'none';

  card.querySelector('.btn-remove-param').addEventListener('click', () => {
    card.remove();
  });

  // Auth links populate
  const authList = card.querySelector('.param-auth-list');
  card.querySelector('.btn-add-param-auth').addEventListener('click', () => {
    addParamAuthRow(authList, definedAuthServices, {});
  });

  if (data.authServices && Array.isArray(data.authServices)) {
    data.authServices.forEach(as => addParamAuthRow(authList, definedAuthServices, as));
  }

  container.appendChild(card);
}

// Add Auth Service link within parameter card
function addParamAuthRow(container, services, data) {
  const row = document.createElement('div');
  row.className = 'param-auth-row flex items-center gap-2 bg-zinc-50 p-1.5 border border-zinc-200/50 rounded';
  
  let options = services.map(s => `<option value="${s}" ${data.name === s ? 'selected' : ''}>${s}</option>`).join('');
  if (services.length === 0) {
    options = `<option value="">No auth services</option>`;
  }

  row.innerHTML = `
    <select class="param-auth-svc bg-white border border-zinc-200 rounded px-1.5 py-1 text-[11px] text-zinc-800 focus:outline-none">
      ${options}
    </select>
    <input type="text" value="${data.field || ''}" placeholder="e.g. sub" class="param-auth-field flex-1 bg-white border border-zinc-200 rounded px-1.5 py-1 text-[11px] text-zinc-800 focus:outline-none" />
    <button type="button" class="btn-remove-param-auth p-1 text-zinc-400 hover:text-red-700 hover:bg-[#FDEBEC] rounded transition-all cursor-pointer">
      <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" /></svg>
    </button>
  `;

  row.querySelector('.btn-remove-param-auth').addEventListener('click', () => {
    row.remove();
  });
  container.appendChild(row);
}

// Form Component Builders (Inputs / Textareas / Checkboxes)
function createFormGroup(label, id, type, value, required = false, helperText = '') {
  const div = document.createElement('div');
  div.className = 'space-y-1';
  div.innerHTML = `
    <label for="${id}" class="text-[10px] tracking-wider font-mono font-semibold uppercase text-zinc-500 block">
      ${label} ${required ? '<span class="text-red-500">*</span>' : ''}
    </label>
    <input type="${type}" id="${id}" value="${value}" ${required ? 'required' : ''} class="w-full bg-white border border-zinc-200 rounded-md px-3 py-1.5 text-xs text-zinc-800 placeholder-zinc-300 focus:outline-none focus:ring-1 focus:ring-zinc-400 focus:border-zinc-400 transition-all" />
    ${helperText ? `<p class="text-[10px] font-mono text-zinc-400">${helperText}</p>` : ''}
  `;
  return div;
}

function createSelectGroup(label, id, options, selectedValue) {
  const div = document.createElement('div');
  div.className = 'space-y-1';
  
  const optionsHtml = options.map(opt => `<option value="${opt}" ${opt === selectedValue ? 'selected' : ''}>${opt}</option>`).join('');
  
  div.innerHTML = `
    <label for="${id}" class="text-[10px] tracking-wider font-mono font-semibold uppercase text-zinc-500 block">${label}</label>
    <select id="${id}" class="w-full bg-white border border-zinc-200 rounded-md px-3 py-1.5 text-xs text-zinc-800 focus:outline-none focus:ring-1 focus:ring-zinc-400 focus:border-zinc-400 transition-all">
      ${optionsHtml}
    </select>
  `;
  return div;
}

function createTextareaGroup(label, id, value, placeholder = '', required = false) {
  const div = document.createElement('div');
  div.className = 'space-y-1';
  div.innerHTML = `
    <label for="${id}" class="text-[10px] tracking-wider font-mono font-semibold uppercase text-zinc-500 block">
      ${label} ${required ? '<span class="text-red-500">*</span>' : ''}
    </label>
    <textarea id="${id}" placeholder="${placeholder}" ${required ? 'required' : ''} class="w-full h-20 text-xs p-2.5 border border-zinc-200 rounded-md focus:outline-none focus:ring-1 focus:ring-zinc-400 focus:border-zinc-400 placeholder-zinc-300 bg-white">${value}</textarea>
  `;
  return div;
}

function createCheckboxGroup(label, id, checked) {
  const div = document.createElement('div');
  div.className = 'flex items-center gap-2.5';
  div.innerHTML = `
    <input type="checkbox" id="${id}" ${checked ? 'checked' : ''} class="rounded border-zinc-300 text-zinc-900 focus:ring-zinc-400 cursor-pointer" />
    <label for="${id}" class="text-xs text-zinc-700 cursor-pointer font-medium select-none">${label}</label>
  `;
  return div;
}

// Generate Live YAML
function renderYaml() {
  const codeContainer = document.getElementById('yaml-code');
  if (resources.length === 0) {
    codeContainer.innerText = '# Config is empty. Create or import resources.';
    document.getElementById('yaml-char-count').innerText = '0 bytes';
    return;
  }

  try {
    // Convert resources state to YAML documents separated by ---
    const documents = resources.map(res => {
      // Create a clean copy without extra UI variables or empty arrays/objects
      const cleanObj = JSON.parse(JSON.stringify(res));
      return jsyaml.dump(cleanObj, {
        noRefs: true,
        lineWidth: 120,
        quotingType: '"',
        forceQuotes: false
      });
    });

    const finalYaml = documents.join('---\n').trim();
    codeContainer.innerText = finalYaml;
    
    // update byte count
    const byteLen = new Blob([finalYaml]).size;
    document.getElementById('yaml-char-count').innerText = `${byteLen} bytes`;
  } catch (err) {
    codeContainer.innerText = `# Serialization Error: ${err.message}`;
  }
}
