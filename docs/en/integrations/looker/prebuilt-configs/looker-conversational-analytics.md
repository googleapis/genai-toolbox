---
title: "Looker Conversational Analytics"
type: docs
description: "Details of the Looker Conversational Analytics prebuilt configuration."
---

## Looker Conversational Analytics

*   `--prebuilt` value: `looker-conversational-analytics`
*   **Environment Variables:**
    *   `LOOKER_BASE_URL`: The URL of your Looker instance.
    *   `LOOKER_CLIENT_ID`: The client ID for the Looker API.
    *   `LOOKER_CLIENT_SECRET`: The client secret for the Looker API.
    *   `LOOKER_VERIFY_SSL`: Whether to verify SSL certificates.
    *   `LOOKER_USE_CLIENT_OAUTH`: Whether to use OAuth for authentication.
    *   `LOOKER_PROJECT`: The GCP Project to use for Conversational Analytics.
    *   `LOOKER_LOCATION`: The GCP Location to use for Conversational Analytics.
*   **Permissions:**
    *   A Looker account with permissions to access the desired models,
        explores, and data is required.
    *   **Looker Instance User** (`roles/looker.instanceUser`): IAM role to
        access Looker.
    *   **Gemini for Google Cloud User** (`roles/cloudaicompanion.user`): IAM
        role to access Conversational Analytics.
    *   **Gemini Data Analytics Stateless Chat User (Beta)**
        (`roles/geminidataanalytics.dataAgentStatelessUser`): IAM role to
        access Conversational Analytics.
*   **Tools:**
    *   `ask_data_insights`: Ask a question of the data.
    *   `get_models`: Retrieves the list of LookML models.
    *   `get_explores`: Retrieves the list of explores in a model.
