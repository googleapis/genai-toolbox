#!/bin/bash
set -e

python3 - << 'EOF'
"""
MCP TOOLBOX: TOOL PAGE LINTER
=============================
This script enforces a standardized structure for individual Tool pages 
(e.g., integrations/postgres/postgres-sql.md). It ensures that LLM agents 
can parse tool capabilities and parameter definitions reliably.

MAINTENANCE GUIDE:
------------------
1. TO ADD A NEW HEADING: 
   Add the exact heading text to the 'ALLOWED_ORDER' list in the desired 
   sequence.

2. TO MAKE A HEADING MANDATORY: 
   Add the heading text to the 'REQUIRED' set.

3. TO UPDATE SHORTCODE LOGIC:
   If the shortcode name changes, update 'SHORTCODE_PATTERN'.

4. SCOPE:
   This script targets all .md files in integrations/ EXCEPT _index.md files.
"""

import os
import re
import sys
from pathlib import Path

# --- CONFIGURATION ---
ALLOWED_ORDER = [
    "About",
    "Compatible Sources",
    "Requirements",
    "Parameters",
    "Example",
    "Output Format",
    "Reference",
    "Advanced Usage",
    "Troubleshooting",
    "Additional Resources"
]
REQUIRED = {"About", "Example"}
SHORTCODE_PATTERN = r"\{\{<\s*compatible-sources.*?>\}\}"
# ---------------------

integration_dir = Path("./docs/en/integrations")
if not integration_dir.exists():
    print("Info: Directory './docs/en/integrations' not found. Skipping linting.")
    sys.exit(0)

has_errors = False

# Find all .md files, excluding _index.md (which are Source pages)
for filepath in integration_dir.rglob("*.md"):
    if filepath.name == "_index.md":
        continue

    with open(filepath, "r", encoding="utf-8") as f:
        content = f.read()

    # Separate YAML frontmatter from the markdown body
    match = re.match(r'^\s*---\s*\n(.*?)\n---\s*(.*)', content, re.DOTALL)
    if match:
        frontmatter = match.group(1)
        body = match.group(2)
    else:
        frontmatter = ""
        body = content

    if not body.strip():
        continue

    file_errors = False

    # 1. Check Frontmatter Title
    title_source = frontmatter if frontmatter else content
    title_match = re.search(r"^title:\s*[\"']?(.*?)[\"']?\s*$", title_source, re.MULTILINE)
    if not title_match or not title_match.group(1).strip().endswith("Tool"):
        found_title = title_match.group(1) if title_match else "None"
        print(f"[{filepath}] Error: Frontmatter title must end with 'Tool'. Found: '{found_title}'")
        file_errors = True

    # 2. Check Shortcode Placement
    sources_section_match = re.search(r"^##\s+Compatible Sources\s*(.*?)(?=^##\s|\Z)", body, re.MULTILINE | re.DOTALL)
    if sources_section_match:
        if not re.search(SHORTCODE_PATTERN, sources_section_match.group(1)):
            print(f"[{filepath}] Error: The compatible-sources shortcode must be placed under '## Compatible Sources'.")
            file_errors = True
    elif re.search(SHORTCODE_PATTERN, body):
        print(f"[{filepath}] Error: Shortcode found, but '## Compatible Sources' heading is missing.")
        file_errors = True

    # 3. Strip code blocks
    clean_body = re.sub(r"^(?:```|~~~).*?^(?:```|~~~)", "", body, flags=re.DOTALL | re.MULTILINE)

    # 4. Check H1 Headings
    if re.search(r"^#\s+\w+", clean_body, re.MULTILINE):
        print(f"[{filepath}] Error: H1 headings (#) are forbidden in the body.")
        file_errors = True

    # 5. Check H2 Headings
    h2s = [h.strip() for h in re.findall(r"^##\s+(.*)", clean_body, re.MULTILINE)]

    # Missing Required
    if missing := (REQUIRED - set(h2s)):
        print(f"[{filepath}] Error: Missing required H2 headings: {missing}")
        file_errors = True

    # Unauthorized Headings
    if unauthorized := (set(h2s) - set(ALLOWED_ORDER)):
        print(f"[{filepath}] Error: Unauthorized H2 headings found: {unauthorized}")
        file_errors = True

    # Strict Ordering
    filtered_h2s = [h for h in h2s if h in ALLOWED_ORDER]
    expected_order = [h for h in ALLOWED_ORDER if h in h2s]
    if filtered_h2s != expected_order:
        print(f"[{filepath}] Error: Headings are out of order.")
        print(f"  Expected: {expected_order}")
        print(f"  Found:    {filtered_h2s}")
        file_errors = True

    if file_errors:
        has_errors = True

if has_errors:
    print("Linting failed for Tool pages. Please fix the structure errors above.")
    sys.exit(1)
else:
    print("Success: All Tool pages passed structure validation.")
EOF