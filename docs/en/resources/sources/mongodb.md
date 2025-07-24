---
title: "MongoDB"
type: docs
weight: 1
description: >
  MongoDB is a no-sql data platform that can not only serve general purpose data requirements also perform VectorSearch where both operational data and embeddings used of search can reside in same document.

---

## About

[MongoDB][mongodb-docs] is a popular NoSQL database that stores data in flexible, JSON-like documents, making it easy to develop and scale applications.

[mongodb-docs]: https://www.mongodb.com/docs/atlas/atlas-vector-search/vector-search-overview/

## Example

```yaml
sources:
    my-mongodb:
        kind: mongodb
        uri: "mongodb+srv://username:password@host.mongodb.net"
        database: sample_mflix
        
```


## Reference

| **field** | **type** | **required** | **description**                                                   |
|-----------|:--------:|:------------:|-------------------------------------------------------------------|
| kind      |  string  |     true     | Must be "mongodb".                                                |
| uri       |  string  |     true     | connection string to connect to MongoDB                           |                      |
| database  |  string  |     true     | Name of the mongodb database to connect to (e.g. "sample_mflix"). |