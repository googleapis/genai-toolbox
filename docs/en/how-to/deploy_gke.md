---
title: "Deploy to GKE"
type: docs
weight: 2
description: >
  How to set up and configure Toolbox to deploy on Google Kubernetes Engine (GKE).
---


## Before you begin

1. [Install the `gcloud` CLI](https://cloud.google.com/sdk/docs/install).

1. Set the PROJECT_ID environment variable:

    ```bash
    export PROJECT_ID="my-project-id"
    ```

1. Initialize gcloud CLI:

    ```bash
    gcloud init
    gcloud config set project $PROJECT_ID
    ```

1. You must have the following APIs enabled:

    ```bash
    gcloud services enable artifactregistry.googleapis.com \
                           cloudbuild.googleapis.com \
                           container.googleapis.com \
                           iam.googleapis.com
    ```

1. `kubectl` is used to manage Kubernetes, the cluster orchestration system used
   by GKE. Verify if you have `kubectl` installed:

    ```bash
    kubectl version --client
    ```
   

1. If needed, install `kubectl` component using the Google Cloud CLI:

   ```bash
   gcloud components install kubectl
   ```

## Create a service account

1. Specify a name for your service account with an environment variable:

    ```bash
    export SA_NAME=toolbox
    ```

1. Create a backend service account:

    ```bash
    gcloud iam service-accounts create $sa_name
    ```

1.  Grant any IAM roles necessary to the IAM service account. Each source have a
    list of necessary IAM permissions listed on it's page. The example below is
    for cloud sql postgres source:

    ```bash
    gcloud projects add-iam-policy-binding $PROJECT_ID \
        --member serviceAccount:$sa_name@$PROJECT_ID.iam.gserviceaccount.com \
        --role roles/cloudsql.client
    ```

    - [AlloyDB IAM Identity](../sources/alloydb-pg.md#iam-identity)
    - [CloudSQL IAM Identity](../sources/cloud-sql-pg.md#iam-identity)
    - [Spanner IAM Identity](../sources/spanner.md#iam-identity)

## Deploying to GKE

1. Set environment variables:

    ```bash
    export cluster_name=toolbox-cluster
    export deployment_name=toolbox
    export service_name=toolbox-service
    export region=us-central1
    export namespace=toolbox-namespace
    export secret_name=toolbox-config
    export ksa_name=toolbox-service-account
    ```

1. Create a [GKE cluster](https://cloud.google.com/kubernetes-engine/docs/concepts/cluster-architecture).

    ```bash
    gcloud container clusters create-auto $cluster_name \
        --location=us-central1 
    ```

1. Get authentication credentials to interact with the cluster. This also
   configures `kubectl` to use the cluster.

    ```bash
    gcloud container clusters get-credentials $cluster_name \
        --region=$region \
        --project=$PROJECT_ID
    ```

1. View the current context for `kubectl`.

    ```bash
    kubectl config current-context
    ```

1. Create namespace for the deployment.

    ```bash
    kubectl create namespace $namespace
    ```

1. Create a Kubernetes Service Account (KSA).

    ```bash
    kubectl create serviceaccount $ksa_name --namespace $namespace
    ```

1. Enable the IAM binding between Google Service Account (GSA) and Kubernetes
   Service Account (KSA).

    ```bash
    gcloud iam service-accounts add-iam-policy-binding \
        --role="roles/iam.workloadIdentityUser" \
        --member="serviceAccount:$PROJECT_ID.svc.id.goog[$namespace/$ksa_name]" \
        $SA_NAME@$PROJECT_ID.iam.gserviceaccount.com
    ```

1. Add annotation to KSA to complete binding:

    ```bash
    kubectl annotate serviceaccount \
        $ksa_name \
        iam.gke.io/gcp-service-account=$SA_NAME@$PROJECT_ID.iam.gserviceaccount.com \
        --namespace $namespace
    ```

1. Prepare the kubernetes Secrets (`tools.yaml` file).

    ```bash
    kubectl create secret generic $secret_name \
        --from-file=./tools.yaml \
        --namespace=$namespace
    ```

1. Create a kubernetes manifest file (`k8s_deployment.yaml`) to build deployment and create service.

    ```yaml
    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: toolbox
      namespace: toolbox-namespace
    spec:
      selector:
        matchLabels:
          app: toolbox
      template:
        metadata:
          labels:
            app: toolbox
        spec:
          serviceAccountName: k8s-toolbox-sa
          containers:
            - name: toolbox
              # use the newest version for toolbox
              image: us-central1-docker.pkg.dev/database-toolbox/toolbox/toolbox:0.0.5
              volumeMounts:
                - name: toolbox-config
                  mountPath: "/etc/tools.yaml"
                  subPath: tools.yaml
                  readOnly: true
          volumes:
            - name: toolbox-config
              secret:
                secretName: toolbox-config
                items:
                - key: tools.yaml
                  path: tools.yaml
    ---
    apiVersion: v1
    kind: Service
    metadata:
      name: toolbox-service
      namespace: toolbox-namespace
    spec:
      selector:
        app: toolbox
      ports:
        - port: 80
          targetPort: 8080
      type: LoadBalancer
    ```

1. Create the deployment.

    ```bash
    kubectl apply -f k8s_deployment.yaml --namespace $namespace
    ```

1. Check the status of deployment.

    ```bash
    kubectl get deployments --namespace $namespace
    ```

1. You can find your IP address created for your service by getting the service
   information through the following.

   ```bash
   kubectl describe services $service_name --namespace $namespace
   ```

1. To look at logs, run the following.

    ```bash
    kubectl logs -f deploy/$deployment_name --namespace toolbox-namespace
    ```

## Clean up resources
1. Delete secret.

    ```bash
    kubectl delete secret $secret_name --namespace $namespace
    ```

1. Delete deployment.

    ```bash
    kubectl delete deployment $deployment_name --namespace $namespace
    ```

1. Delete the application's service.

    ```bash
    kubectl delete service $service_name --namespace $namespace
    ```

1. Delete the kubernetes cluster.

    ```bash
    gcloud container clusters delete $cluster_name \
        --location=$region
    ```