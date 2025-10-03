#!/bin/bash

set -euxo pipefail

# Generate image tag based on git SHA only (no version tracking)
git_sha=$(echo -n "$COMPARE" | head -c 7)
image_tag="sha-$git_sha"
echo "image_tag=$image_tag"
echo "image_tag=$image_tag" >> $GITHUB_OUTPUT
echo "**image_tag**: \`$image_tag\`" >> $GITHUB_STEP_SUMMARY

# Get changed files to trigger relevant jobs
change_schema="./.github/workflows/utils/change_schema.yml"
# Load yml array into a bash array
# Need to output each entry as a single line
readarray pipelines < <(yq e -o=j -I=0 '.pipelines[]' $change_schema)

for item in "${pipelines[@]}"; do
    # item is a yaml snippet representing a single entry
    name=$(echo "$item" | yq e '.name' -)
    regex=$(echo "$item" | yq e '.regex' -)
    echo "name: $name"
    echo "regex: $regex"

    # Grep with -e
    # This causes the run to terminate immediately when any pipeline exits with a non-zero status.
    # grep returns a 1 when it doesn't find any match, and thus, terminates immediately.
    # We instead run an OR on the grep command, and exit on anything greater than 1 so we don't
    # clobber other error codes.
    exit_code=0
    result=$(git diff $BASE $COMPARE --name-only | grep -P -c "$regex" || exit_code=$?)
    echo $result
    if (( exit_code > 1 )) ; then
        exit $exit_code
    fi

    # cast to string, that GitHub actions can then ingest as a boolean via fromJSON()
    if (( result > 0 )) ; then
      result="true"
    else
      result="false"
    fi

    echo "$name=$result" >> $GITHUB_OUTPUT
done
