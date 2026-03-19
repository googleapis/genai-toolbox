#!/bin/bash
set -e

python3 - << 'EOF'
"""
MCP TOOLBOX: TOOL PAGE LINTER
=============================
This script enforces a standardized structure for individual Tool pages 
and their parent directory wrappers. It ensures LLM agents can parse 
tool capabilities and parameter definitions reliably.

MAINTENANCE GUIDE:
------------------
1. TO ADD A NEW HEADING: 
   Add the exact heading text to the 'ALLOWED_ORDER' list in the desired 
   sequence.

2. TO MAKE A HEADING MANDATORY/OPTIONAL: 
   Add or remove the heading text in the 'REQUIRED' set.

3. TO UPDATE SHORTCODE LOGIC:
   If the shortcode name changes, update the 'SHORTCODE_PATTERN' variable.

4. SCOPE & BEHAVIOR:
   This script targets all .md files in docs/en/integrations/**/tools/.
   - For `_index.md` files: It only validates the frontmatter (requiring 
     `title: "Tools"` and `weight: 2`) and ignores the body.
   - For regular tool files: It validates H1/H2 hierarchy, checks for 
     required headings ("About", "Example"), and enforces that the 
     `{{< compatible-sources >}}` shortcode is paired with the 
     "## Compatible Sources" heading.
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
tools_pages_found = 0

# Specifically target the tools directories
for filepath in integration_dir.rglob("tools/*.md"):
    tools_pages_found += 1
    with open(filepath, "r", encoding="utf-8") as f:
        content = f.read()

    # Separate YAML frontmatter from the markdown body
    match = re.match(r'^\s*---\s*\n(.*?)\n---\s*(.*)', content, re.DOTALL)
    if match:
        frontmatter = match.group(1)
        body = match.group(2)
    else:
        print(f"[{filepath}] Error: Missing or invalid YAML frontmatter.")
        has_errors = True
        continue

    file_errors = False

    # --- SPECIAL VALIDATION FOR tools/_index.md ---
    if filepath.name == "_index.md":
        title_match = re.search(r"^title:\s*[\"']?(.*?)[\"']?\s*$", frontmatter, re.MULTILINE)
        if not title_match or title_match.group(1).strip() != "Tools":
            print(f"[{filepath}] Error: tools/_index.md must have exactly title: \"Tools\"")
            file_errors = True

        weight_match = re.search(r"^weight:\s*(\d+)\s*$", frontmatter, re.MULTILINE)
        if not weight_match or weight_match.group(1).strip() != "2":
            print(f"[{filepath}] Error: tools/_index.md must have exactly weight: 2")
            file_errors = True

        if file_errors:
            has_errors = True
        continue # Skip the rest of the body linting for this structural file

    # --- VALIDATION FOR REGULAR TOOL PAGES ---
    # If the file has no markdown content (metadata placeholder only), skip it entirely
    if not body.strip():
        continue

    # 1. Check Shortcode Placement
    sources_section_match = re.search(r"^##\s+Compatible Sources\s*(.*?)(?=^##\s|\Z)", body, re.MULTILINE | re.DOTALL)
    if sources_section_match:
        if not re.search(SHORTCODE_PATTERN, sources_section_match.group(1)):
            print(f"[{filepath}] Error: The compatible-sources shortcode must be placed under '## Compatible Sources'.")
            file_errors = True
    elif re.search(SHORTCODE_PATTERN, body):
        print(f"[{filepath}] Error: Shortcode found, but '## Compatible Sources' heading is missing.")
        file_errors = True

    # 2. Strip code blocks from body to avoid linting example markdown headings
    clean_body = re.sub(r"```.*?```", "", body, flags=re.DOTALL)

    # 3. Check H1 Headings
    if re.search(r"^#\s+\w+", clean_body, re.MULTILINE):
        print(f"[{filepath}] Error: H1 headings (#) are forbidden in the body.")
        file_errors = True

    # 4. Check H2 Headings
    h2s = re.findall(r"^##\s+(.*)", clean_body, re.MULTILINE)
    h2s = [h2.strip() for h2 in h2s]

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

if tools_pages_found == 0:
    print("Info: No tool directories found. Passing gracefully.")
    sys.exit(0)
elif has_errors:
    print("Linting failed for Tool pages. Please fix the structure errors above.")
    sys.exit(1)
else:
    print(f"Success: All {tools_pages_found} Tool page(s) passed structure validation.")
EOF
