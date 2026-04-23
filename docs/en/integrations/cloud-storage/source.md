---
title: "Cloud Storage Source"
linkTitle: "Source"
type: docs
weight: 1
description: >
  Cloud Storage is Google Cloud's managed service for storing unstructured objects (files) in buckets. Toolbox connects at the project level, allowing tools to list, read, and manage objects across any bucket the credentials can access.
no_list: true
---

## About

[Cloud Storage][gcs-docs] is Google Cloud's managed service for storing
unstructured data (blobs) in containers called *buckets*. Buckets live in a GCP
project; objects are addressed by `gs://<bucket>/<object>`.

If you are new to Cloud Storage, you can try the
[quickstart][gcs-quickstart] to create a bucket and upload your first objects.

The Cloud Storage source is configured at the **project** level. Individual
tools take a `bucket` parameter, so a single configured source can operate
against any bucket the underlying credentials are authorized for.

[gcs-docs]: https://cloud.google.com/storage/docs
[gcs-quickstart]: https://cloud.google.com/storage/docs/discover-object-storage-console

## Available Tools

{{< list-tools >}}

## Requirements

### IAM Permissions

Cloud Storage uses [Identity and Access Management (IAM)][iam-overview] to
control access to buckets and objects. Toolbox uses your
[Application Default Credentials (ADC)][adc] to authorize and authenticate when
interacting with Cloud Storage.

In addition to [setting the ADC for your server][set-adc], ensure the IAM
identity has the appropriate role for the tools being exposed. Common roles:

- `roles/storage.objectViewer` — read-only access to objects (sufficient for
  `cloud-storage-list-objects` and `cloud-storage-read-object`)
- `roles/storage.objectUser` — read and write access to objects
- `roles/storage.admin` — full control, including bucket management

See [Cloud Storage IAM roles][gcs-iam] for the full list.

[iam-overview]: https://cloud.google.com/storage/docs/access-control/iam
[adc]: https://cloud.google.com/docs/authentication#adc
[set-adc]: https://cloud.google.com/docs/authentication/provide-credentials-adc
[gcs-iam]: https://cloud.google.com/storage/docs/access-control/iam-roles

## Example

```yaml
kind: source
name: my-gcs-source
type: "cloud-storage"
project: "my-project-id"
```

## Reference

| **field** | **type** | **required** | **description**                                                                 |
|-----------|:--------:|:------------:|---------------------------------------------------------------------------------|
| type      |  string  |     true     | Must be "cloud-storage".                                                        |
| project   |  string  |     true     | Id of the GCP project the configured source is associated with (e.g. "my-project-id"). |
