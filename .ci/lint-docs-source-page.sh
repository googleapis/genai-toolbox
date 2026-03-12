#!/bin/bash
set -e

python3 - << 'EOF'
"""
MCP TOOLBOX: SOURCE PAGE LINTER
===============================
This script enforces a standardized structure for integration Source pages 
(_index.md files). It ensures users can predictably find 
information across all database integrations.

MAINTENANCE GUIDE:
------------------
1. TO ADD A NEW HEADING: 
   Add the exact heading text to the 'ALLOWED_ORDER' list in the desired 
   sequence.

2. TO MAKE A HEADING MANDATORY: 
   Add the heading text to the 'REQUIRED' set.

3. TO IGNORE NEW CONTENT TYPES:
   Update the regex in the 'clean_body' variable (Step 3) to strip out 
   Markdown before linting.

4. SCOPE:
   This script ignores top-level directory files and only targets 
   integrations/{provider}/_index.md.
"""

import os
import re
import sys
from pathlib import Path

# --- CONFIGURATION ---
ALLOWED_ORDER = [
    "About",
    "Available Tools",
    "Requirements",
    "Example",
    "Reference",
    "Advanced Usage",
    "Troubleshooting",
    "Additional Resources"
]
REQUIRED = {"About", "Example", "Reference"}
SHORTCODE_PATTERN = r"\{\{<\s*list-tools.*?>\}\}"
# ---------------------

integration_dir = Path("./docs/en/integrations")
if not integration_dir.exists():
    print("Info: Directory './docs/en/integrations' not found. Skipping linting.")
    sys.exit(0)

has_errors = False

for filepath in integration_dir.rglob("_index.md"):
    if filepath.parent == integration_dir:
        continue

    with open(filepath, "r", encoding="utf-8") as f:
        content = f.read()

    match = re.match(r'^\s*---\s*\n(.*?)\n---\s*(.*)', content, re.DOTALL)
    if match:
        frontmatter, body = match.group(1), match.group(2)
    else:
        frontmatter, body = "", content

    if not body.strip():
        continue

    file_errors = False

    # 1. Frontmatter Title Check
    title_match = re.search(r"^title:\s*[\"']?(.*?)[\"']?\s*$", frontmatter if frontmatter else content, re.MULTILINE)
    if not title_match or not title_match.group(1).strip().endswith("Source"):
        print(f"[{filepath}] Error: Title must end with 'Source'.")
        file_errors = True

    # 2. Shortcode Placement Check
    tools_section = re.search(r"^##\s+Available Tools\s*(.*?)(?=^##\s|\Z)", body, re.MULTILINE | re.DOTALL)
    if tools_section:
        if not re.search(SHORTCODE_PATTERN, tools_section.group(1)):
            print(f"[{filepath}] Error: {{< list-tools >}} must be under '## Available Tools'.")
            file_errors = True
    elif re.search(SHORTCODE_PATTERN, body):
        print(f"[{filepath}] Error: {{< list-tools >}} found, but '## Available Tools' heading is missing.")
        file_errors = True

    # 3. Heading Linting (Stripping code blocks first)
    clean_body = re.sub(r"```.*?```", "", body, flags=re.DOTALL)

    if re.search(r"^#\s+\w+", clean_body, re.MULTILINE):
        print(f"[{filepath}] Error: H1 (#) headings are forbidden in the body.")
        file_errors = True

    h2s = [h.strip() for h in re.findall(r"^##\s+(.*)", clean_body, re.MULTILINE)]

    # 4. Required & Unauthorized Check
    if missing := (REQUIRED - set(h2s)):
        print(f"[{filepath}] Error: Missing required H2s: {missing}")
        file_errors = True

    if unauthorized := (set(h2s) - set(ALLOWED_ORDER)):
        print(f"[{filepath}] Error: Unauthorized H2s found: {unauthorized}")
        file_errors = True

    # 5. Order Check
    if [h for h in h2s if h in ALLOWED_ORDER] != [h for h in ALLOWED_ORDER if h in h2s]:
        print(f"[{filepath}] Error: Headings out of order. Reference: {ALLOWED_ORDER}")
        file_errors = True

    if file_errors: has_errors = True

if has_errors:
    print("Linting failed. Fix structure errors above.")
    sys.exit(1)
print("Success: Source pages validated.")
sys.exit(0)
EOF
