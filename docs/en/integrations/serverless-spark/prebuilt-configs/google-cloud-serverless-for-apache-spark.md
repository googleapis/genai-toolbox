---
title: "Google Cloud Serverless for Apache Spark"
type: docs
description: "Details of the Google Cloud Serverless for Apache Spark prebuilt configuration."
---

## Google Cloud Serverless for Apache Spark

*   `--prebuilt` value: `serverless-spark`
*   **Environment Variables:**
    *   `SERVERLESS_SPARK_PROJECT`: The GCP project ID
    *   `SERVERLESS_SPARK_LOCATION`: The GCP Location.
*   **Permissions:**
    *   **Dataproc Serverless Viewer** (`roles/dataproc.serverlessViewer`) to
        view serverless batches.
    *   **Dataproc Serverless Editor** (`roles/dataproc.serverlessEditor`) to
        view serverless batches.
*   **Tools:**
    *   `list_batches`: Lists Spark batches.
    *   `get_batch`: Gets information about a Spark batch.
    *   `cancel_batch`: Cancels a Spark batch.
    *   `create_pyspark_batch`: Creates a PySpark batch.
    *   `create_spark_batch`: Creates a Spark batch.
    *   `list_sessions`: Lists Spark sessions.
    *   `get_session`: Gets a Spark session.
    *   `get_session_template`: Gets a Spark session template.
