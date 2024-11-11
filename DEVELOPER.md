# DEVELOPER.md

## Before you begin

1. Make sure you've setup your databases.

1. Install the latest version of [Go](https://go.dev/doc/install).

1. Locate and download dependencies:

    ```bash
    go get
    go mod tidy
    ```

## Developing Toolbox

### Run Toolbox from local source

1. Open a local connection to your database by starting the [Cloud SQL Auth Proxy][cloudsql-proxy].

1. You should already have a `tools.yaml` created with your [sources and tools configurations](./README.md#Configuration).

1. You can specify flags for the Toolbox server. Execute the following to list the possible CLI flags:

    ```bash
    go run . --help
    ```

1. To run the server, execute the following (with any flags, if applicable):

    ```bash
    go run .
    ```

    The server will listen on port 5000 (by default).

1. Test endpoint using the following:

    ```bash
    curl http://127.0.0.1:5000
    ```

### Run tests locally

1. Run lint with the following:

    ```bash
    golangci-lint run --fix
    ```

1. Run all tests with the following:

    ```bash
    go test -race -v ./...
    ```

## Compile the app locally

### Compile Toolbox binary

1. Run build to compile binary:

    ```bash
    go build -o toolbox
    ```

1. You can specify flags for the Toolbox server. Execute the following to list the possible CLI flags:

    ```bash
    ./toolbox --help
    ```

1. To run the binary, execute the following (with any flags, if applicable):

    ```bash
    ./toolbox
    ```

    The server will listen on port 5000 (by default).

1. Test endpoint using the following:

    ```bash
    curl http://127.0.0.1:5000
    ```

### Compile Toolbox container images

1. Run build to compile container image:

    ```bash
    docker build -t toolbox:dev .
    ```

1. Execute the following to view image:

    ```bash
    docker images
    ```

1. Run container image with Docker:

    ```bash
    docker run -d toolbox:dev
    ```

## Developing Toolbox SDKs

Setting up a Development Environment:

1. Clone the repository:

    ```bash
    git clone https://github.com/googleapis/genai-toolbox.git
    ```

1. Navigate to the SDK directory, e.g.:

    ```bash
    cd genai-toolbox/sdks/langchain
    ```

1. Install the SDK and test dependencies:

    ```bash
    pip install -e .[test]
    ```

1. Run tests and/or contribute to the SDK's development.

    ```bash
    pytest
    ```

## CI/CD Details

Cloud Build is used to run tests against Google Cloud resources in test project.

### Releasing

There are two types of release for Toolbox, including a versioned release and continuous release.

- Versioned release: Official supported distributions with the `latest` tag. The release process for versioned release is in [versioned.release.cloudbuild.yaml](https://github.com/googleapis/genai-toolbox/blob/main/versioned.release.cloudbuild.yaml).
- Continuous release: Used for early testing features between official supported releases and end-to-end testings.

#### Supported OS and Architecture binaries

The following OS and computer architecture is supported within the binary releases.

- linux/amd64
- darwin/arm64
- darwin/amd64
- windows/amd64

#### Supported container images

The following base container images is supported within the container image releases.

- distroless

### Automated tests

Integration and unit tests are automatically triggered via CloudBuild during each PR creation.

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

### Trigger

To run Cloud Build tests on GitHub from external contributors, ie RenovateBot, comment: `/gcbrun`.

[cloudsql-proxy]: https://cloud.google.com/sql/docs/mysql/sql-proxy
