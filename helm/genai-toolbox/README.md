# MCP Toolbox for Databases Helm Chart

This chart installs the [MCP Toolbox for Databases](https://googleapis.github.io/genai-toolbox/getting-started/introduction/) on [Kubernetes](https://kubernetes.io) via the [Helm](https://helm.sh) package manager.

- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [Configuration](#configuration)

## Prerequisites

- Kubernetes 1.22+
- kubectl 1.22+ 
- Helm 3.8.0+ 

## Installation

# Add the Helm repository

```shell
helm repo add mcp-toolbox https://googleapis.github.io/genai-toolbox
helm repo update
```

# Install the chart

```shell
helm upgrade --install toolbox genai-toolbox/genai-toolbox -f values.yaml --namespace toolbox
```
# (optional) Customize the chart values

```shell
helm show values genai-toolbox/genai-toolbox > values.yaml
```

# Uninstall the chart
helm uninstall toolbox --namespace toolbox
```

## Configuration

The following table lists the configurable parameters and their default values.

Specify each parameter using the `--set key=value[,key=value]` argument to `helm install`.

| Parameter | Description | Default |
|---|---|---|
| `autoscaling.enabled` | Enable Horizontal Pod Autoscaler | false |
| `autoscaling.maxReplicas` | Maximum number of pods for autoscaling | 5 |
| `autoscaling.minReplicas` | Minimum number of pods for autoscaling | 1 |
| `autoscaling.targetCPUUtilizationPercentage` | Target CPU utilization for autoscaling | 80 |
| `autoscaling.targetMemoryUtilizationPercentage` | Target memory utilization for autoscaling | (unset) |
| `config` | Toolbox YAML config for auth, sources, tools, toolsets | (see values.yaml) |
| `image.name` | Image name | toolbox |
| `image.pullPolicy` | Image pull policy | IfNotPresent |
| `image.repository` | Container image repository | us-central1-docker.pkg.dev/database-toolbox/toolbox/ |
| `image.tag` | Image tag | latest |
| `imagePullSecrets` | List of image pull secrets | [] |
| `livenessProbe` | Liveness probe configuration for the container | `{ httpGet: { path: /, port: 5000 }, initialDelaySeconds: 10, periodSeconds: 60 }` |
| `namespace` | Kubernetes namespace to deploy into | toolbox |
| `options.address` | Address to bind the app | 0.0.0.0 |
| `options.log_level` | Log level | INFO |
| `options.logging_format` | Logging format | standard |
| `options.port` | App port | 5000 |
| `options.telemetry_gcp` | Enable GCP telemetry | false |
| `options.telemetry_otlp` | OTLP endpoint URL | "" |
| `options.telemetry_service_name` | Service name for telemetry | toolbox |
| `readinessProbe` | Readiness probe configuration for the container | `{ httpGet: { path: /, port: 5000 }, initialDelaySeconds: 5, periodSeconds: 5 }` |
| `replicas` | Number of replicas to deploy | 1 |
| `resources` | CPU/memory resource requests and limits | `{ requests: { cpu: 100m, memory: 128Mi }, limits: { cpu: 1, memory: 512Mi } }` |
| `securityContext` | Container-level security context | `{ runAsNonRoot: true, runAsUser: 1000, allowPrivilegeEscalation: false, capabilities: { drop: [ALL] } }` |
| `serviceAccount.create` | Whether to create a ServiceAccount | true |
| `serviceAccount.name` | Name of the ServiceAccount | genai-toolbox |