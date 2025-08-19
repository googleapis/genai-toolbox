<!-- [START cloud_setup] -->
If you plan to use **Google Cloud’s Vertex AI** with your agent (e.g., using
`vertexai=True` or a Google GenAI model), follow these one-time setup steps for
local development:

1. [Install the Google Cloud CLI](https://cloud.google.com/sdk/docs/install)
1. [Set up Application Default Credentials (ADC)](https://cloud.google.com/docs/authentication/set-up-adc-local-dev-environment)
1. Set your project and enable Vertex AI

    ```bash
    gcloud config set project YOUR_PROJECT_ID
    gcloud services enable aiplatform.googleapis.com
    ```

[install-python]: https://wiki.python.org/moin/BeginnersGuide/Download
[install-pip]: https://pip.pypa.io/en/stable/installation/
[install-venv]: https://packaging.python.org/en/latest/tutorials/installing-packages/#creating-virtual-environments
[install-postgres]: https://www.postgresql.org/download/
<!-- [END cloud_setup] -->