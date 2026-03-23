---
title: "Toolbox with MCP Auth"
type: docs
weight: 4
description: >
  How to set up and configure Toolbox with MCP Authentication.
---

## Overview

Toolbox supports integrating with Model Context Protocol (MCP) clients by enabling OAuth/JWT-based Server-Wide Authentication. This allows an MCP client natively sending Bearer tokens to be verified by Toolbox before executing any queries.

This guide details the specific configuration steps required to deploy Toolbox with MCP Auth enabled.

## Step 1: Configure the `generic` Auth Service

Update your `tools.yaml` file to use a `generic` authentication service with `mcpEnabled` set to `true`. This instructs Toolbox to intercept requests on the `/mcp` routes and validate Bearer tokens using the JWKS (JSON Web Key Set) fetched from your OIDC provider endpoint (`authUrl`).

```yaml
authServices:
  - name: my-mcp-auth
    type: generic
    mcpEnabled: true
    authUrl: "https://your-auth-server.example.com"
    audience: "your-mcp-audience" # Matches the `aud` claim in the JWT
    scopesRequired:
      - "mcp:tools"
```

When `mcpEnabled` is true, Toolbox also provisions the `/.well-known/oauth-protected-resource` Protected Resource Metadata (PRM) endpoint automatically using the `authUrl`.

## Step 2: Deployment

Deploying Toolbox with MCP auth requires defining the `TOOLBOX_URL` that the deployed service will use, as this URL must be included as the `resource` field in the PRM returned to the client.

You can set this either through the `TOOLBOX_URL` environment variable or the `--toolbox-url` command-line flag during deployment.

### Local Deployment

To run Toolbox locally with MCP auth enabled, simply export the `TOOLBOX_URL` referencing your local port before running the binary:

```bash
export TOOLBOX_URL="http://127.0.0.1:5000"
./toolbox --tools-file tools.yaml
```

If you prefer to use the `--toolbox-url` flag explicitly:

```bash
./toolbox --tools-file tools.yaml --toolbox-url "http://127.0.0.1:5000"
```

### Cloud Run Deployment

```bash
# Set your target Cloud Run URL
export TOOLBOX_URL="https://toolbox-service-123456789-uc.a.run.app"
export IMAGE="us-central1-docker.pkg.dev/database-toolbox/toolbox/toolbox:latest"

gcloud run deploy toolbox \
    --image $IMAGE \
    --service-account toolbox-identity \
    --region us-central1 \
    --set-secrets "/app/tools.yaml=tools:latest" \
    --set-env-vars "TOOLBOX_URL=${TOOLBOX_URL}" \
    --args="--tools-file=/app/tools.yaml","--address=0.0.0.0","--port=8080"
```

### Alternative: Manual PRM File Override

If you strictly need to define your own Protected Resource Metadata instead of auto-generating it from the `AuthService` config, you can use the `--mcp-prm-file <path>` flag. 

1. Create a `prm.json` containing the RFC-9207 compliant metadata. Note that the `resource` field must match the `TOOLBOX_URL`:
   ```json
   {
     "resource": "https://toolbox-service-123456789-uc.a.run.app",
     "authorization_servers": ["https://your-auth-server.example.com"],
     "scopes_supported": ["mcp:tools"],
     "bearer_methods_supported": ["header"]
   }
   ```
2. Set the `--mcp-prm-file` flag to the path of the PRM file.

 - If you are using local deployment, you can just provide the path to the file directly:
   ```bash
   ./toolbox --tools-file tools.yaml --mcp-prm-file prm.json
   ```
 - If you are using Cloud Run, upload it to GCP Secret Manager and Attach the secret to the Cloud Run deployment and provide the flag.
    ```bash
    gcloud secrets create prm_file --data-file=prm.json

    gcloud run deploy toolbox \
      # ... previous args
      --set-secrets "/app/tools.yaml=tools:latest,/app/prm.json=prm_file:latest" \
      --args="--tools-file=/app/tools.yaml","--mcp-prm-file=/app/prm.json","--port=8080"
    ```
    

## Step 3: Connecting to the Secure MCP Endpoint

Once the Cloud Run instance is deployed, your MCP client must obtain a valid JWT token from your authentication server (the `authUrl` in `tools.yaml`).

The client should provide this JWT via the standard HTTP `Authorization` header when connecting to the `Streamable HTTP` SSE endpoint (`/mcp`):

```bash
{
  "mcpServers": {
    "toolbox-secure": {
      "type": "http",
      "url": "https://toolbox-service-123456789-uc.a.run.app/mcp",
      "headers": {
        "Authorization": "Bearer eyJhbGc..."
      }
    }
  }
}
```

Toolbox will intercept incoming connections, fetch the latest JWKS from your `authUrl`, and validate that the `aud` (audience), signature, and `scopes` on the JWT match the requirements defined by your `mcpEnabled` auth service.
