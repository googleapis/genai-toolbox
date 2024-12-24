#!/bin/bash

# TODO: Configure image
export IMAGE=us-central1-docker.pkg.dev/database-toolbox/toolbox/toolbox:latest

# Deploy toolbox to cloud run
echo "Deploying toolbox to Cloud Run..."
if ! gcloud run deploy toolbox \
    --image $IMAGE \
    --service-account 107716898620-compute \
    --region us-central1 \
    --set-secrets "/app/tools.yaml=sdk_testing_tools:latest" \
    --args="--tools_file=/app/tools.yaml","--address=0.0.0.0","--port=8080"; then
  echo "ERROR: Failed to deploy toolbox to Cloud Run."
  exit 1
fi
echo "Toolbox deployed successfully."

# Proxy connections to Cloud Run
echo "Proxying connections to Cloud Run..."
if ! gcloud run services proxy toolbox --port=8080 --region=us-central1; then
  echo "ERROR: Failed to proxy connections to Cloud Run."
  exit 1
fi
echo "Proxy established successfully."

# TODO: Update url?
# Check if the endpoint works
echo "Checking if the endpoint works..."
if curl http://127.0.0.1:8080; then
  echo "Endpoint is working."
else
  echo "ERROR: Endpoint is not working."
  exit 1
fi