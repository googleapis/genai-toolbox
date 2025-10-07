<!-- This file has been used in local_quickstart.md, local_quickstart_go.md & local_quickstart_js.md -->
<!-- [START cloud_setup] -->
If you plan to use **Google Cloud’s Vertex AI** with your agent (e.g., using
`vertexai=True` or a Google GenAI model), follow these one-time setup steps for
local development:

1. [Install the Google Cloud CLI](https://cloud.google.com/sdk/docs/install)
1. [Set up Application Default Credentials
   (ADC)](https://cloud.google.com/docs/authentication/set-up-adc-local-dev-environment)
1. Set your project and enable Vertex AI

    ```bash
    gcloud config set project YOUR_PROJECT_ID
    gcloud services enable aiplatform.googleapis.com
    ```

<!-- [END cloud_setup] -->