---
title: "Deploy to Cloud Run"
type: docs
weight: 3
description: >
  How to set up and configure Toolbox to run on Cloud Run.
---


## Before you begin

1. [Install](https://cloud.google.com/sdk/docs/install) the Google Cloud CLI.

1. Set the PROJECT_ID environment variable:

    ```bash
    export PROJECT_ID="my-project-id"
    ```

1. Initialize gcloud CLI:

    ```bash
    gcloud init
    gcloud config set project $PROJECT_ID
    ```

1. Make sure you've set up and initialized your database.

1. You must have the following APIs enabled:

    ```bash
    gcloud services enable run.googleapis.com \
                           cloudbuild.googleapis.com \
                           artifactregistry.googleapis.com \
                           iam.googleapis.com \
                           secretmanager.googleapis.com

    ```

1. To create an IAM account, you must have the following IAM permissions (or
   roles):
    - Create Service Account role (roles/iam.serviceAccountCreator)

1. To create a secret, you must have the following roles:
    - Secret Manager Admin role (roles/secretmanager.admin)

1. To deploy to Cloud Run, you must have the following set of roles:
    - Cloud Run Developer (roles/run.developer)
    - Service Account User role (roles/iam.serviceAccountUser)

{{< notice note >}}
If you are under a domain restriction organization policy
[restricting](https://cloud.google.com/run/docs/authenticating/public#domain-restricted-sharing)
unauthenticated invocations for your project, you will need to access your
deployed service as described under [Testing private
services](https://cloud.google.com/run/docs/triggering/https-request#testing-private).
{{< /notice >}}

{{< notice note >}}
If you are using sources that require VPC-access (such as
AlloyDB or Cloud SQL over private IP), make sure your Cloud Run service and the
database are in the same VPC network.
{{< /notice >}}


## Create a service account

1. Create a backend service account if you don't already have one:

    ```bash
    gcloud iam service-accounts create toolbox-identity
    ```

1.  Grant permissions to use secret manager:

    ```bash
    gcloud projects add-iam-policy-binding $PROJECT_ID \
        --member serviceAccount:toolbox-identity@$PROJECT_ID.iam.gserviceaccount.com \
        --role roles/secretmanager.secretAccessor
    ```

1. Grant additional permissions to the service account that are specific to the source, e.g.:
    - [AlloyDB for PostgreSQL](../resources/sources/alloydb-pg.md#iam-permissions)
    - [Cloud SQL for PostgreSQL](../resources/sources/cloud-sql-pg.md#iam-permissions)

## Configure `tools.yaml` file

Create a `tools.yaml` file that contains your configuration for Toolbox. For
details, see the
[configuration](https://github.com/googleapis/genai-toolbox/blob/main/README.md#configuration)
section.

## Deploy to Cloud Run

1. Upload `tools.yaml` as a secret:

    ```bash
    gcloud secrets create tools --data-file=tools.yaml
    ```

    If you already have a secret and want to update the secret version, execute
    the following:

    ```bash
    gcloud secrets versions add tools --data-file=tools.yaml
    ```

1. Set an environment variable to the container image that you want to use for cloud run:

    ```bash
    export IMAGE=us-central1-docker.pkg.dev/database-toolbox/toolbox/toolbox:latest
    ```

1. Deploy Toolbox to Cloud Run using the following command:

    ```bash
    gcloud run deploy toolbox \
        --image $IMAGE \
        --service-account toolbox-identity \
        --region us-central1 \
        --set-secrets "/app/tools.yaml=tools:latest" \
        --args="--tools_file=/app/tools.yaml","--address=0.0.0.0","--port=8080"
        # --allow-unauthenticated # https://cloud.google.com/run/docs/authenticating/public#gcloud
    ```

    If you are using a VPC network, use the command below:

    ```bash
    gcloud run deploy toolbox \
        --image $IMAGE \
        --service-account toolbox-identity \
        --region us-central1 \
        --set-secrets "/app/tools.yaml=tools:latest" \
        --args="--tools_file=/app/tools.yaml","--address=0.0.0.0","--port=8080" \
        # TODO(dev): update the following to match your VPC if necessary
        --network default \
        --subnet default
        # --allow-unauthenticated # https://cloud.google.com/run/docs/authenticating/public#gcloud
    ```

## Connecting to Cloud Run

Next, we will use `gcloud` to authenticate requests to our Cloud Run instance:

1. Run the `run services proxy` to proxy connections to Cloud Run:

    ```bash
    gcloud run services proxy toolbox --port=8080 --region=us-central1
    ```

    If you are prompted to install the proxy, reply *Y* to install.

1. Finally, use `curl` to verify the endpoint works:

    ```bash
    curl http://127.0.0.1:8080
    ```

## Connecting with Toolbox Client SDK

Next, we will use Toolbox with client SDK:

1. Run the following to retrieve a non-deterministic URL for the cloud run service:

    ```bash
    gcloud run services describe toolbox --format 'value(status.url)'
    ```

1. Import and initialize the toolbox client with the URL retrieved above:

   {{< tabpane persist=header >}}
{{< tab header="LangChain" lang="Python" >}}
from toolbox_langchain import ToolboxClient

# Replace with the cloud run service URL generated above
toolbox = ToolboxClient("http://$YOUR_URL")
{{< /tab >}}
{{< tab header="Llamaindex" lang="Python" >}}
from toolbox_llamaindex import ToolboxClient

# Replace with the cloud run service URL generated above
toolbox = ToolboxClient("http://$YOUR_URL")
{{< /tab >}}
{{< /tabpane >}}

