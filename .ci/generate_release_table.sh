#! /bin/bash


# Check if VERSION has been set
if [ -z "${VERSION}" ]; then
  echo "Error: VERSION env var is not set" >&2  # Print to stderr
  exit 1  # Exit with a non-zero status to indicate an error
fi


FILES=("linux.amd64" "darwin.arm64" "darwin.amd64" "windows.amd64" "windows.arm64")
output_string=""

# Define the descriptions - ensure this array's order matches FILES
DESCRIPTIONS=(
    "For **Linux** systems running on **Intel/AMD 64-bit processors**."
    "For **macOS** systems running on **Apple Silicon** (M1, M2, M3, etc.) processors."
    "For **macOS** systems running on **Intel processors**."
    "For **Windows** systems running on **Intel/AMD 64-bit processors**."
    "For **Windows** systems running on **ARM 64-bit processors**."
)

# Write the table header
ROW_FMT="| %-105s | %-120s | %-67s |\n"
output_string+=$(printf "$ROW_FMT" "**OS/Architecture**" "**Description**" "**SHA256 Hash**")$'\n'
output_string+=$(printf "$ROW_FMT" "$(printf -- '-%0.s' {1..105})" "$(printf -- '-%0.s' {1..120})" "$(printf -- '-%0.s' {1..67})")$'\n'


# Loop through all files matching the pattern "toolbox.*.*"
for i in "${!FILES[@]}"
do
    file_key="${FILES[$i]}" # e.g., "linux.amd64"
    description_text="${DESCRIPTIONS[$i]}"

    # Extract OS and ARCH from the filename
    OS=$(echo "$file_key" | cut -d '.' -f 1)
    ARCH=$(echo "$file_key" | cut -d '.' -f 2)

    # Determine the GCS bucket (default to the official production bucket if not overridden)
    GCS_BUCKET="${BUCKET:-test-release-script}"

    # Get release URL and GCS path
    if [ "$OS" = 'windows' ];
    then
        URL="https://storage.googleapis.com/$GCS_BUCKET/$VERSION/$OS/$ARCH/toolbox.exe"
        GCS_PATH="gs://$GCS_BUCKET/$VERSION/$OS/$ARCH/toolbox.exe"
    else
        URL="https://storage.googleapis.com/$GCS_BUCKET/$VERSION/$OS/$ARCH/toolbox"
        GCS_PATH="gs://$GCS_BUCKET/$VERSION/$OS/$ARCH/toolbox"
    fi

    if ! curl "$URL" --fail --output toolbox 2>/dev/null; then
        echo "Direct download failed (403/private bucket?). Trying gcloud storage cp..." >&2
        gcloud storage cp "$GCS_PATH" toolbox || exit 1
    fi

    # Calculate the SHA256 checksum of the file
    SHA256=$(shasum -a 256 toolbox | awk '{print $1}')

    # Write the table row
    output_string+=$(printf "$ROW_FMT" "[$OS/$ARCH]($URL)" "$description_text" "$SHA256")$'\n'

    rm toolbox
done

printf "$output_string\n"

