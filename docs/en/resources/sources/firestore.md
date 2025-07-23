---
title: "Firestore"
type: docs
weight: 1
description: >
  Firestore is a NoSQL document database built for automatic scaling, high performance, and ease of application development. It's a fully managed, serverless database that supports mobile, web, and server development.

---

# Firestore Source

[Firestore][firestore-docs] is a NoSQL document database built for automatic
scaling, high performance, and ease of application development. While the
Firestore interface has many of the same features as traditional databases,
as a NoSQL database it differs from them in the way it describes relationships
between data objects.

If you are new to Firestore, you can [create a database and learn the
basics][firestore-quickstart].

[firestore-docs]: https://cloud.google.com/firestore/docs
[firestore-quickstart]: https://cloud.google.com/firestore/docs/quickstart-servers

## Requirements

### IAM Permissions

Firestore uses [Identity and Access Management (IAM)][iam-overview] to control
user and group access to Firestore resources. Toolbox will use your [Application
Default Credentials (ADC)][adc] to authorize and authenticate when interacting
with [Firestore][firestore-docs].

In addition to [setting the ADC for your server][set-adc], you need to ensure
the IAM identity has been given the correct IAM permissions for accessing
Firestore. Common roles include:
- `roles/datastore.user` - Read and write access to Firestore
- `roles/datastore.viewer` - Read-only access to Firestore

See [Firestore access control][firestore-iam] for more information on
applying IAM permissions and roles to an identity.

[iam-overview]: https://cloud.google.com/firestore/docs/security/iam
[adc]: https://cloud.google.com/docs/authentication#adc
[set-adc]: https://cloud.google.com/docs/authentication/provide-credentials-adc
[firestore-iam]: https://cloud.google.com/firestore/docs/security/iam

### Database Selection

Firestore allows you to create multiple databases within a single project. Each
database is isolated from the others and has its own set of documents and
collections. If you don't specify a database in your configuration, the default
database named `(default)` will be used.

## Example

```yaml
sources:
  my-firestore-source:
    kind: "firestore"
    project: "my-project-id"
    database: "my-database"
```

## Reference

| **field** | **type** | **required** | **description**                                                                                          |
|-----------|:--------:|:------------:|----------------------------------------------------------------------------------------------------------|
| kind      |  string  |     true     | Must be "firestore".                                                                                     |
| project   |  string  |     true     | Id of the GCP project that contains the Firestore database (e.g. "my-project-id").                       |
| database  |  string  |     true     | Name of the Firestore database to connect to.               |
