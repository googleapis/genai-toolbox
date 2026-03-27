#!/bin/bash
set -e


python3 - << 'EOF'
"""
MCP TOOLBOX: SOURCE PAGE LINTER
===============================
This script enforces a standardized structure for integration Source pages 
(source.md files). It ensures users can predictably find connection details
and configurations across all database integrations.

Note: The structural _index.md folder wrappers are intentionally ignored 
by this script as they should only contain YAML frontmatter.

MAINTENANCE GUIDE:
------------------
1. TO ADD A NEW HEADING: 
   Add the exact heading text to the 'ALLOWED_ORDER' list in the desired 
   sequence.

2. TO MAKE A HEADING MANDATORY/OPTIONAL: 
   Add or remove the heading text in the 'REQUIRED' set. 

3. TO IGNORE NEW CONTENT TYPES:
   Update the regex in the 'clean_body' variable to strip out 
   Markdown before linting.

4. SCOPE:
   This script only targets docs/en/integrations/**/source.md.
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

# Regex to catch any variation of the list-tools shortcode
SHORTCODE_PATTERN = r"\{\{<\s*list-tools.*?>\}\}"
# ---------------------

integration_dir = Path("./docs/en/integrations")
if not integration_dir.exists():
    print("Info: Directory './docs/en/integrations' not found. Skipping linting.")
    sys.exit(0)

has_errors = False
source_pages_found = 0

# ONLY scan files specifically named "source.md"
for filepath in integration_dir.rglob("source.md"):
    source_pages_found += 1
    file_errors = False

    if filepath.parent.parent != integration_dir:
        continue

    with open(filepath, "r", encoding="utf-8") as f:
        content = f.read()

    match = re.match(r'^\s*---\s*\n(.*?)\n---\s*(.*)', content, re.DOTALL)
    if match:
        frontmatter, body = match.group(1), match.group(2)
    else:
        print(f"[{filepath}] Error: Missing or invalid YAML frontmatter.")
        has_errors = True
        continue

    # 1. Check for linkTitle: "Source" in frontmatter
    link_title_match = re.search(r"^linkTitle:\s*[\"']?(.*?)[\"']?\s*$", frontmatter, re.MULTILINE)
    if not link_title_match or link_title_match.group(1).strip() != "Source":
        print(f"[{filepath}] Error: Frontmatter must contain exactly linkTitle: \"Source\".")
        file_errors = True

    # 2. Check for weight: 1 in frontmatter
    weight_match = re.search(r"^weight:\s*[\"']?(\d+)[\"']?\s*$", frontmatter, re.MULTILINE)
    if not weight_match or weight_match.group(1).strip() != "1":
        print(f"[{filepath}] Error: Frontmatter must contain exactly weight: 1.")
        file_errors = True

    # 3. Check Shortcode Placement & Available Tools Section (Only if present)
    tools_section_match = re.search(r"^##\s+Available Tools\s*(.*?)(?=^##\s|\Z)", body, re.MULTILINE | re.DOTALL)
    if tools_section_match:
        if not re.search(SHORTCODE_PATTERN, tools_section_match.group(1)):
            print(f"[{filepath}] Error: The list-tools shortcode must be placed under the '## Available Tools' heading.")
            file_errors = True

    # Strip code blocks from body to avoid linting example markdown headings
    clean_body = re.sub(r"```.*?```", "", body, flags=re.DOTALL)

    if re.search(r"^#\s+\w+", clean_body, re.MULTILINE):
        print(f"[{filepath}] Error: H1 (#) headings are forbidden in the body.")
        file_errors = True

    h2s = [h.strip() for h in re.findall(r"^##\s+(.*)", clean_body, re.MULTILINE)]

    # Missing Required Headings
    missing = REQUIRED - set(h2s)
    if missing:
        print(f"[{filepath}] Error: Missing required H2 headings: {missing}")
        file_errors = True

    if unauthorized := (set(h2s) - set(ALLOWED_ORDER)):
        print(f"[{filepath}] Error: Unauthorized H2s found: {unauthorized}")
        file_errors = True

    # 5. Order Check
    if [h for h in h2s if h in ALLOWED_ORDER] != [h for h in ALLOWED_ORDER if h in h2s]:
        print(f"[{filepath}] Error: Headings out of order. Reference: {ALLOWED_ORDER}")
        file_errors = True

    if file_errors: has_errors = True

# Handle final output based on what was found
if source_pages_found == 0:
    print("Info: No 'source.md' files found in integrations. Passing gracefully.")
    sys.exit(0)
elif has_errors:
    print(f"\nLinting failed. Please fix the structure errors in the {source_pages_found} 'source.md' file(s) above.")
    sys.exit(1)
else:
    print(f"Success: {source_pages_found} 'source.md' file(s) passed structure validation.")
EOF
