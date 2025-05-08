---
title: "Deploy to Kubernetes (Helm)"
type: docs
weight: 2
description: >
  How to set up and configure Toolbox to deploy on any Kubernetes cluster using Helm.
---

# Overview

This chart installs the [MCP Toolbox for Databases](https://googleapis.github.io/genai-toolbox/getting-started/introduction/) (formerly "Gen AI Toolbox for Databases") on [Kubernetes](https://kubernetes.io) via the [Helm](https://helm.sh) package manager.

- [Before you begin](#before-you-begin)
- [Deploy to Kubernetes](#deploy-to-kubernetes)
- [Configuration](#configuration)

## Before you begin

Assuming you have a running Kubernetes cluster,

1. Verify if you have `kubectl` installed (version: 1.25+):
    ```bash
    kubectl version --client
    ```

2. Verify if you have `helm` installed (version: 3.1+):
    ```bash
    helm version --client
    ```

## Deploy to Kubernetes

1. Add the Helm repository:

    ```bash
    helm repo add genai-toolbox https://googleapis.github.io/genai-toolbox
    helm repo update
    ```

2. Review and Customize Values

    Download the default values file and edit as needed:

    ```bash
    helm show values genai-toolbox/genai-toolbox > values.yaml
    # Edit values.yaml to configure your sources, tools, auth, etc.
    ```

3. Install the Chart

    ```bash
    helm install toolbox genai-toolbox/genai-toolbox -f values.yaml --namespace toolbox --create-namespace
    ```

4. Check that your pods are running:

    ```bash
    kubectl get pods -n toolbox
    ```

5. To remove the deployment:

    ```bash
    helm uninstall toolbox -n toolbox
    ```

## Configuration

See the GitHub [values.yaml](https://github.com/googleapis/genai-toolbox/blob/main/helm/genai-toolbox/values.yaml) for a list of configurable parameters and their default values.