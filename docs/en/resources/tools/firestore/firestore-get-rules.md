---
title: "firestore-get-rules"
type: docs
weight: 1
description: >
  A "firestore-get-rules" tool retrieves the active Firestore security rules for the current project.
aliases:
- /resources/tools/firestore-get-rules
---

## About

A `firestore-get-rules` tool retrieves the active [Firestore security
rules](https://firebase.google.com/docs/firestore/security/get-started) for the
current project.
It's compatible with the following sources:

- [firestore](../../sources/firestore.md)

`firestore-get-rules` takes no input parameters and returns the security rules
content along with metadata such as the ruleset name, and timestamps.

## Example

```yaml
tools:
  get_firestore_rules:
    kind: firestore-get-rules
    source: my-firestore-source
    description: Use this tool to retrieve the active Firestore security rules.
```

## Reference

| **field**   |    **type**   | **required** | **description**                                       |
|-------------|:-------------:|:------------:|-------------------------------------------------------|
| kind        |     string    |     true     | Must be "firestore-get-rules".                        |
| source      |     string    |     true     | Name of the Firestore source to retrieve rules from.  |
| description |     string    |     true     | Description of the tool that is passed to the LLM.    |
