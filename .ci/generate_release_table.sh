#! /bin/bash


# Check if VERSION has been set
if [ -z "${VERSION}" ]; then
  echo "Error: VERSION env var is not set" >&2  # Print to stderr
  exit 1  # Exit with a non-zero status to indicate an error
fi


FILES=("linux.amd64" "darwin.arm64" "darwin.amd64" "windows.amd64")
output_string=""

# Write the table header
ROW_FMT="| %-93s | %73s |"
output_string+=$(printf "$ROW_FMT" "**os/arch**" "**sha256**")$'\n'
output_string+=$(printf "$ROW_FMT" $(printf -- '-%0.s' {1..93}) $(printf -- '-%0.s' {1..73}))$'\n'

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
    SHA256=$(shasum -a 256 toolbox | awk '{print $1}')

    # Write the table row
    # output_string+="| [$OS/$ARCH]($URL)   | $SHA256 |\n"
    output_string+=$(printf "$ROW_FMT" "[$OS/$ARCH]($URL)" "$SHA256")$'\n'

    rm toolbox
done
printf "$output_string\n"

