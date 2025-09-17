#! /bin/bash


# Check if VERSION has been set
if [ -z "${VERSION}" ]; then
  echo "Error: VERSION env var is not set" >&2  # Print to stderr
  exit 1  # Exit with a non-zero status to indicate an error
fi


FILES=("linux.amd64" "darwin.arm64" "darwin.amd64" "windows.amd64")
output_string=""

# Define the descriptions - ensure this array's order matches FILES
DESCRIPTIONS=(
    "For **Linux** systems running on **Intel/AMD 64-bit processors**."
    "For **macOS** systems running on **Apple Silicon** (M1, M2, M3, etc.) processors."
    "For **macOS** systems running on **Intel processors**."
    "For **Windows** systems running on **Intel/AMD 64-bit processors**."
)

# Write the table header
ROW_FMT="| %-105s | %-120s | %-67s | %-108s |\n"
output_string+=$(printf "$ROW_FMT" "**OS/Architecture**" "**Description**" "**SHA256 Hash**" "**Signature**")$'\n'
output_string+=$(printf "$ROW_FMT" "$(printf -- '-%0.s' {1..105})" "$(printf -- '-%0.s' {1..120})" "$(printf -- '-%0.s' {1..67})" "$(printf -- '-%0.s' {1..67})")$'\n'


# Loop through all files matching the pattern "toolbox.*.*"
for i in "${!FILES[@]}"
do
    file_key="${FILES[$i]}" # e.g., "linux.amd64"
    description_text="${DESCRIPTIONS[$i]}"

    # Extract OS and ARCH from the filename
    OS=$(echo "$file_key" | cut -d '.' -f 1)
    ARCH=$(echo "$file_key" | cut -d '.' -f 2)

    # Get release URL
    if [ "$OS" = 'windows' ];
    then
        URL="https://storage.googleapis.com/genai-toolbox/$VERSION/$OS/$ARCH/toolbox.exe"
    else
        URL="https://storage.googleapis.com/genai-toolbox/$VERSION/$OS/$ARCH/toolbox"
    fi

    # Generate the signature URL and link
    SIG_URL="${URL}.sig"
    SIG_LINK="[.sig]($SIG_URL)"

    curl "$URL" --fail --output toolbox || exit 1

    # Calculate the SHA256 checksum of the file
    SHA256=$(shasum -a 256 toolbox | awk '{print $1}')

    # Write the table row
    output_string+=$(printf "$ROW_FMT" "[$OS/$ARCH]($URL)" "$description_text" "$SHA256" "$SIG_LINK")$'\n'

    rm toolbox
done

printf "$output_string\n"
