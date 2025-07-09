---
title: "Dataplex"
type: docs
weight: 1
description: >
  Dataplex Universal Catalog is a unified, intelligent governance solution for data and AI assets in Google Cloud. Dataplex Universal Catalog powers AI, analytics, and business intelligence at scale.
---

# Dataplex Source

[Dataplex][bigquery-docs] Universal Catalog is a unified, intelligent governance solution for data and AI assets in Google Cloud. Dataplex Universal Catalog powers AI, analytics, and business intelligence at scale.

At the heart of these governance capabilities is a catalog that contains a centralized inventory of the data assets in your organization. Dataplex Universal Catalog holds business, technical, and runtime metadata for all of your data. It helps you discover relationships and semantics in the metadata by applying artificial intelligence and machine learning.

[dataplex-docs]: https://cloud.google.com/dataplex/docs

## Example

```yaml
sources:
  my-dataplex-source:
    kind: "dataplex"
    project: "my-project-id"
```

## Reference

| **field** | **type** | **required** | **description**                                                               |
|-----------|:--------:|:------------:|-------------------------------------------------------------------------------|
| kind      |  string  |     true     | Must be "dataplex".                                                           |
| project   |  string  |     true     | Id of the GCP project (e.g. "my-project-id").                                 |
