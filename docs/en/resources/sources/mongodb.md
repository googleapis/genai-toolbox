---
title: "MongoDB"
type: docs
weight: 1
description: >
  MongoDB is a no-sql data platform that can not only serve general purpose data requirements also perform VectorSearch where both operational data and embeddings used of search can reside in same document.

---

## About

[MongoDB][mongodb-docs] is a popular NoSQL database that stores data in
flexible, JSON-like documents, making it easy to develop and scale applications.

[mongodb-docs]: https://www.mongodb.com/docs/atlas/getting-started/

## Example

```yaml
sources:
    my-mongodb:
        kind: mongodb
        uri: "mongodb+srv://username:password@host.mongodb.net"
        
        
```

## Reference

| **field** | **type** | **required** | **description**                                                   |
|-----------|:--------:|:------------:|-------------------------------------------------------------------|
| kind      |  string  |     true     | Must be "mongodb".                                                |
| uri       |  string  |     true     | connection string to connect to MongoDB                           |
