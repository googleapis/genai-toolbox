---
title: "Deploy to Kubernetes"
type: docs
weight: 4
description: >
  How to set up and configure Toolbox to deploy on Kubernetes with Google Kubernetes Engine (GKE).
---


## Before you begin

1. Set the PROJECT_ID environment variable:

    ```bash
    export PROJECT_ID="my-project-id"
    ```

1. [Install the `gcloud` CLI](https://cloud.google.com/sdk/docs/install).

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
    gcloud iam service-accounts create $SA_NAME
    ```

1. Grant any IAM roles necessary to the IAM service account. Each source have a
    list of necessary IAM permissions listed on it's page. The example below is
    for cloud sql postgres source:

    ```bash
    gcloud projects add-iam-policy-binding $PROJECT_ID \
        --member serviceAccount:$SA_NAME@$PROJECT_ID.iam.gserviceaccount.com \
        --role roles/cloudsql.client
    ```

    - [AlloyDB IAM Identity](../resources/sources/alloydb-pg.md#iam-permissions)
    - [CloudSQL IAM Identity](../resources/sources/cloud-sql-pg.md#iam-permissions)
    - [Spanner IAM Identity](../resources/sources/spanner.md#iam-permissions)

## Deploy to Kubernetes

1. Set environment variables:

    ```bash
    export CLUSTER_NAME=toolbox-cluster
    export DEPLOYMENT_NAME=toolbox
    export SERVICE_NAME=toolbox-service
    export REGION=us-central1
    export NAMESPACE=toolbox-namespace
    export SECRET_NAME=toolbox-config
    export KSA_NAME=toolbox-service-account
    ```

1. Create a [GKE cluster](https://cloud.google.com/kubernetes-engine/docs/concepts/cluster-architecture).

    ```bash
    gcloud container clusters create-auto $CLUSTER_NAME \
        --location=us-central1
    ```

1. Get authentication credentials to interact with the cluster. This also
   configures `kubectl` to use the cluster.

    ```bash
    gcloud container clusters get-credentials $CLUSTER_NAME \
        --region=$REGION \
        --project=$PROJECT_ID
    ```

1. View the current context for `kubectl`.

    ```bash
    kubectl config current-context
    ```

1. Create namespace for the deployment.

    ```bash
    kubectl create namespace $NAMESPACE
    ```

1. Create a Kubernetes Service Account (KSA).

    ```bash
    kubectl create serviceaccount $KSA_NAME --namespace $NAMESPACE
    ```

1. Enable the IAM binding between Google Service Account (GSA) and Kubernetes
   Service Account (KSA).

    ```bash
    gcloud iam service-accounts add-iam-policy-binding \
        --role="roles/iam.workloadIdentityUser" \
        --member="serviceAccount:$PROJECT_ID.svc.id.goog[$NAMESPACE/$KSA_NAME]" \
        $SA_NAME@$PROJECT_ID.iam.gserviceaccount.com
    ```

1. Add annotation to KSA to complete binding:

    ```bash
    kubectl annotate serviceaccount \
        $KSA_NAME \
        iam.gke.io/gcp-service-account=$SA_NAME@$PROJECT_ID.iam.gserviceaccount.com \
        --namespace $NAMESPACE
    ```

1. Prepare the Kubernetes secret for your `tools.yaml` file.

    ```bash
    kubectl create secret generic $SECRET_NAME \
        --from-file=./tools.yaml \
        --namespace=$NAMESPACE
    ```

1. Create a Kubernetes manifest file (`k8s_deployment.yaml`) to build deployment.

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
          serviceAccountName: toolbox-service-account
          containers:
            - name: toolbox
              # Recommend to use the latest version of toolbox
              image: us-central1-docker.pkg.dev/database-toolbox/toolbox/toolbox:latest
              args: ["--address", "0.0.0.0"]
              ports:
                - containerPort: 5000
              volumeMounts:
                - name: toolbox-config
                  mountPath: "/app/tools.yaml"
                  subPath: tools.yaml
                  readOnly: true
          volumes:
            - name: toolbox-config
              secret:
                secretName: toolbox-config
                items:
                - key: tools.yaml
                  path: tools.yaml
    ```

1. Create the deployment.

    ```bash
    kubectl apply -f k8s_deployment.yaml --namespace $NAMESPACE
    ```

1. Check the status of deployment.

    ```bash
    kubectl get deployments --namespace $NAMESPACE
    ```

1. Create a Kubernetes manifest file (`k8s_service.yaml`) to build service.

    ```yaml
    apiVersion: v1
    kind: Service
    metadata:
      name: toolbox-service
      namespace: toolbox-namespace
      annotations:
        cloud.google.com/l4-rbs: "enabled"
    spec:
      selector:
        app: toolbox
      ports:
        - port: 5000
          targetPort: 5000
      type: LoadBalancer
    ```

1. Create the service.

    ```bash
    kubectl apply -f k8s_service.yaml --namespace $NAMESPACE
    ```

1. You can find your IP address created for your service by getting the service
   information through the following.

   ```bash
   kubectl describe services $SERVICE_NAME --namespace $NAMESPACE
   ```

1. To look at logs, run the following.

    ```bash
    kubectl logs -f deploy/$DEPLOYMENT_NAME --namespace $NAMESPACE
    ```

1. You might have to wait a couple of minutes. It is ready when you can see
   `EXTERNAL-IP` with the following command:

    ```bash
    kubectl get svc -n $NAMESPACE
    ```

1. Access toolbox locally.

    ```bash
    curl <EXTERNAL-IP>:5000
    ```

## Clean up resources

1. Delete secret.

    ```bash
    kubectl delete secret $SECRET_NAME --namespace $NAMESPACE
    ```

1. Delete deployment.

    ```bash
    kubectl delete deployment $DEPLOYMENT_NAME --namespace $NAMESPACE
    ```

1. Delete the application's service.

    ```bash
    kubectl delete service $SERVICE_NAME --namespace $NAMESPACE
    ```

1. Delete the Kubernetes cluster.

    ```bash
    gcloud container clusters delete $CLUSTER_NAME \
        --location=$REGION
    ```
