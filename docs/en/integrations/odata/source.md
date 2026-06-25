---
title: "OData Source"
type: docs
linkTitle: "Source"
weight: 1
description: >
  An "odata" source connects to an OData service (v2 or v4), supporting SAP-specific gateway strategies, CSRF handshakes, and dynamic principal propagation.
no_list: true
---

## About

An `odata` source establishes a connection to an OData service. It supports standard v2 and v4 OData protocol features, including batch processing, deep inserts, and server-side pagination, as well as SAP Gateway strategies (such as automatic fetching and recycling of CSRF security tokens).

## Available Tools

{{< list-tools >}}

## Requirements

### Authentication

Toolbox supports multiple authentication methods for OData sources:

1.  **Basic Authentication:** Standard username and password.
2.  **Bearer Token:** A static OAuth or session Bearer token.
3.  **TLS Client Certificates (x509):** Password-encrypted or unencrypted client certificates for mutual TLS (mTLS) environments.
4.  **Dynamic User OAuth (Principal Propagation):** Forwards the user's OAuth access token from the client application dynamically to the OData backend.

### SAP Gateway Strategy

When `authStrategy` is configured to `sap-gateway`, Toolbox automatically performs a pre-flight `HEAD` request with the client credentials to fetch the necessary CSRF security tokens and session cookies. It manages session persistence per user using an LRU cache and handles token eviction seamlessly if a request fails with a `403 CSRF token validation` error.

## Example

Initialize an OData source with Basic authentication:

```yaml
kind: source
name: sap-sales-order-srv
type: "odata"
baseUrl: "https://sap-gateway.example.com/sap/opu/odata/sap/API_SALES_ORDER_SRV"
disableSslVerification: true # Set to true to skip server certificate verification for development
auth:
  type: "basic"
  username: "MY_SAP_USER"
  password: "MY_SAP_PASSWORD"
authStrategy: "sap-gateway"
compatibility:
  sapUrlQuoting: true
```

Initialize an OData source with dynamic pass-through client authorization (OAuth 2.0):

```yaml
kind: source
name: sap-sales-order-srv-oauth
type: "odata"
baseUrl: "https://sap-gateway.example.com/sap/opu/odata/sap/API_SALES_ORDER_SRV"
useClientOauth: "true"
authStrategy: "sap-gateway"
compatibility:
  sapUrlQuoting: true
```

## Reference

| **field** | **type** | **required** | **description** |
| :--- | :---: | :---: | :--- |
| type | string | true | Must be "odata". |
| baseUrl | string | true | The root URL of the OData service. |
| timeout | string | false | Request timeout duration. Defaults to "30s" (e.g. "10s", "1m"). |
| headers | map[string]string | false | Custom HTTP headers to include in all requests. |
| queryParams | map[string]string | false | Custom query parameters to include in all requests. |
| disableSslVerification | boolean | false | If true, skips TLS certificate verification (equivalent to `curl -k`). |
| auth | object | false | Optional credential configuration for static auth (Basic, Bearer, or x509). |
| useClientOauth | string | false | Enables dynamic pass-through of the user's OAuth token. Set to `'true'` to read the token from the default `Authorization` header, or set to a custom header name (e.g., `X-SAP-OAuth-Token`). |
| authStrategy | string | false | Specifies a special pre-flight or caching strategy. Set to `'sap-gateway'` to enable automatic CSRF handshakes and cookie jar session tracking. |
| compatibility | object | false | Special configuration flags for non-standard OData implementations. |
