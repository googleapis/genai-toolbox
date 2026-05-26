#!/usr/bin/env bash

# ==============================================================================
# Script: lint-sample-filters.sh
# Description: Checks markdown files to ensure 'sample_filters' match the
#              allowed tags in filters.yaml.
# ==============================================================================


FILTERS_FILE=".hugo/data/filters.yaml"
DOCS_DIR="docs/en/"
FAILED=0

# Load valid filters from the YAML file (remove dashes and quotes)
VALID_FILTERS=$(grep -Eo "^\s*-\s*.*" "$FILTERS_FILE" | sed -E 's/^\s*-\s*["\047]?//; s/["\047]?\s*$//')

echo "Scanning $DOCS_DIR for invalid sample filters..."

# Find and check each markdown file
while IFS= read -r file; do

    # CASE A: Inline Array (e.g., sample_filters: ["Tag 1", "Tag 2"])
    if grep -q "^sample_filters:\s*\[" "$file"; then
        TAGS=$(grep "^sample_filters:\s*\[" "$file" | grep -Eo '"[^"]+"|\x27[^\x27]+\x27' | sed "s/['\"]//g")

    # CASE B: Vertical List (e.g., - Tag 1 \n - Tag 2)
    elif grep -q "^sample_filters:" "$file"; then
        TAGS=$(awk '/^sample_filters:/ {flag=1; next} /^[a-zA-Z]/ {flag=0} flag && /-/ {sub(/^[ \t]*-[ \t]*["\x27]?/, ""); sub(/["\x27]?\s*$/, ""); print}' "$file")

    # Skip file if no sample_filters exist
    else
        continue
    fi

    # Validate extracted tags against the allowed list
    while IFS= read -r tag; do
        # Skip empty lines
        [[ -z "$tag" ]] && continue

        # Check if the exact tag exists in our valid list
        if ! grep -Fxq "$tag" <<< "$VALID_FILTERS"; then
            echo "Invalid filter found: '$tag' in $file"
            FAILED=1
        fi
    done <<< "$TAGS"

done < <(find "$DOCS_DIR" -name "*.md")

# Final Output
if [[ $FAILED -eq 1 ]]; then
    echo "------------------------------------------------------"
    echo "Build failed: Unapproved sample_filter detected."
    echo "Check your spelling/spaces or add it to $FILTERS_FILE."
    exit 1
else
    echo "All sample filters are valid!"
    exit 0
fi
