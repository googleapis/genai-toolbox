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

1. Create a `tools.yaml` file with your [sources and tools configurations](./README.md#Configuration).

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

### Testing

- Run the lint check:

    ```bash
    golangci-lint run --fix
    ```

- Run unit tests locally:

    ```bash
    go test -race -v ./...
    ```

- Run integration tests locally:
    1. Set required environment variables. For a complete lists of required
    vairables for each source, check out the [Cloud Build testing
    configuration](./.ci/integration.cloudbuild.yaml).
        - Use your own GCP email as the `SERVICE_ACCOUNT_EMAIL`.
        - Use the Google Cloud SDK application Client ID as the `CLIENT_ID`. Ask the
        Toolbox maintainers if you don't know it already.

    2. Run the integration test for your target source with the required Go
    build tags specified at the top of each integration test file:

        ```shell
            go test -race -v ./tests/<YOUR_TEST_DIR>
        ```

        For example, to run the AlloyDB integration test, run:

        ```shell
            go test -race -v ./tests/alloydbpg
        ```

- Run integration tests on your PR:

    For internal contributors, the testing workflows should trigger
    automatically. For external contributors, ask the Toolbox
    maintainers to trigger the testing workflows on your PR.

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

## Developing Documentation

1. [Install Hugo](https://gohugo.io/installation/macos/) version 0.146.0+.
1. Move into the `.hugo` directory

    ```bash
    cd .hugo
    ```

1. Install dependencies

    ```bash
    npm ci
    ```

1. Run the server

    ```bash
    hugo server
    ```

## Developing Toolbox SDKs

Please refer to the [SDK developer guide](https://github.com/googleapis/mcp-toolbox-sdk-python/blob/main/DEVELOPER.md)

## (Optional) Maintainer Information

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

- Event: Pull request
- Region:
  - global - for default worker pools
- Source:
  - Generation: 1st gen
  - Repo: googleapis/genai-toolbox (GitHub App)
  - Base branch: `^main$`
- Comment control: Required except for owners and collaborators
- Filters: add directory filter
- Config: Cloud Build configuration file
  - Location: Repository (add path to file)
- Service account: set for demo service to enable ID token creation to use to authenticated services

### Trigger

Trigger the PR tests on PRs from external contributors:

- Cloud Build tests: comment `/gcbrun`
- Unit tests: add `tests:run` label
