---
title: "Cloud Storage"
type: docs
description: "Details of the Cloud Storage prebuilt configuration."
---

## Cloud Storage

*   `--prebuilt` value: `cloud-storage`
*   **Environment Variables:**
    *   `CLOUD_STORAGE_PROJECT`: The GCP project ID that owns the buckets.
*   **Permissions:**
    *   **Storage Object User** (`roles/storage.objectUser`) for object reads and
        writes (`list_objects`, `read_object`, `download_object`,
        `get_object_metadata`, `write_object`, `upload_object`, `copy_object`,
        `move_object`, `delete_object`).
    *   **Storage Admin** (`roles/storage.admin`) for bucket lifecycle and IAM
        operations (`list_buckets`, `create_bucket`, `delete_bucket`,
        `get_bucket_metadata`, `get_bucket_iam_policy`).
    *   For read-only deployments, **Storage Object Viewer**
        (`roles/storage.objectViewer`) plus **Storage Legacy Bucket Reader**
        (`roles/storage.legacyBucketReader`) is sufficient for the read-only
        subset.
*   **Tools:**
    *   `list_buckets`: Lists Cloud Storage buckets in the configured project.
    *   `list_objects`: Lists objects in a bucket with optional prefix and
        delimiter filtering.
    *   `get_bucket_metadata`: Returns metadata for a bucket.
    *   `get_bucket_iam_policy`: Returns the IAM policy bindings for a bucket.
    *   `get_object_metadata`: Returns metadata for an object.
    *   `read_object`: Reads a UTF-8 text object (or byte range). Capped at 8
        MiB; binary objects are rejected.
    *   `download_object`: Downloads an object to a local file path.
    *   `create_bucket`: Creates a bucket in the configured project.
    *   `delete_bucket`: Deletes an empty bucket.
    *   `upload_object`: Uploads a local file to an object.
    *   `write_object`: Writes text content directly to an object.
    *   `copy_object`: Copies an object to a destination object.
    *   `move_object`: Atomically renames an object within the same bucket.
    *   `delete_object`: Deletes an object.
*   **Toolsets:**
    *   `cloud-storage-buckets`: Bucket administration (list, create, inspect
        metadata and IAM policy, delete).
    *   `cloud-storage-objects`: Object management (list, read, write, copy,
        move, delete, retrieve metadata).
