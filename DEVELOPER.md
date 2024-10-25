# DEVELOPER.md

## Before you begin

1. Make sure you've setup your databases.

1. Install [Go](go-getting-started).

## Run the app locally

### Running toolbox

1. Locate and download dependencies:

    ```bash
    go mod tidy
    ```

1. Open a local connection to your database by starting the [Cloud SQL Auth Proxy][cloudsql-proxy].

1. You should already have a `tools.yaml` created with your [sources and tools configurations](./README.md#Configuration).

1. To run the server, execute the following:

    ```bash
    go run .
    ```

    The server will listen on port 5000.

1. Test endpoint using the following:

    ```bash
    curl http://127.0.0.1:5000`
    ```

## Testing

### Run tests locally

1. Run all tests with the following:

    ```bash
    go test -race -v ./...
    ```

### CI Platform Setup

Cloud Build is used to run tests against Google Cloud resources in test project.

#### Trigger Setup
Create a Cloud Build trigger via the UI or `gcloud` with the following specs:

* Event: Pull request
* Region:
    * global - for default worker pools
* Source:
  * Generation: 1st gen
  * Repo: googleapis/genai-toolbox (GitHub App)
  * Base branch: `^main$`
* Comment control: Required except for owners and collaborators
* Filters: add directory filter
* Config: Cloud Build configuration file
  * Location: Repository (add path to file)
* Service account: set for demo service to enable ID token creation to use to authenticated services

#### Trigger

To run Cloud Build tests on GitHub from external contributors, ie RenovateBot, comment: `/gcbrun`.

## Versioning

This app will be released based on version number MAJOR.MINOR.PATCH:

- MAJOR: Breaking change is made, requiring user to redeploy all or some of the app.
- MINOR: Backward compatible feature change or addition that doesn't require redeploying.
- PATCH: Backward compatible bug fixes and minor updates

[cloudsql-proxy]: https://cloud.google.com/sql/docs/mysql/sql-proxy
