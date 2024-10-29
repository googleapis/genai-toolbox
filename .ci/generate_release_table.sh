#! /bin/bash

FILES=("linux.amd64" "darwin.arm64" "darwin.amd64" "windows.amd64")
output_string=""

# Write the table header
output_string+="| **os/arch** | **sha256**    |\n"
output_string+="|------|---------------|\n"

# Loop through all files matching the pattern "toolbox.*.*"
for file in "${FILES[@]}"
do
    # Extract OS and ARCH from the filename
    OS=$(echo "$file" | cut -d '.' -f 1)
    ARCH=$(echo "$file" | cut -d '.' -f 2)

    # Get release URL
    URL=https://storage.googleapis.com/genai-toolbox/$VERSION/$OS/$ARCH/toolbox

    curl "$URL" --fail --output toolbox || exit 1

    # Calculate the SHA256 checksum of the file
    # Running from mac
    SHA256=$(shasum -a 256 toolbox | awk '{print $1}')
    # Running from non-mac
    # SHA256=$(sha256sum toolbox | awk '{print $1}')

    # Write the table row
    output_string+="| [$OS/$ARCH]($URL)   | $SHA256 |\n"

    rm toolbox
done
echo "$output_string"

