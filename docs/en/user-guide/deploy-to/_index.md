---
title: "Deploy Toolbox"
type: docs
weight: 5
description: >
  Learn how to deploy the MCP Toolbox server to production environments.
---

Once you have tested your MCP Toolbox configuration locally, you can deploy the server to a highly available, production-ready environment.

Choose your preferred deployment platform below to get started:

*   **[Docker](./docker/)**: Run the official Toolbox container image on any Docker-compatible host.
*   **[Google Cloud Run](./cloud-run/)**: Deploy a fully managed, scalable, and secure cloud run instance.
*   **[Kubernetes](./kubernetes/)**: Deploy the Toolbox as a microservice using GKE.

{{< notice tip >}}
**Production Security:** When moving to production, never hardcode passwords or API keys directly into your `tools.yaml`. Always use environment variable substitution and inject those values securely through your deployment platform's secret manager.
{{< /notice >}}
