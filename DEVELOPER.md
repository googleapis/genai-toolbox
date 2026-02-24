# DEVELOPER.md

This document provides instructions for setting up your development environment
and contributing to the Toolbox project.

## Prerequisites

Before you begin, ensure you have the following:

1. **Databases:** Set up the necessary databases for your development
   environment.
1. **Go:** Install the latest version of [Go](https://go.dev/doc/install).
1. **Dependencies:** Download and manage project dependencies:

    ```bash
    go get
    go mod tidy
    ```

## Developing Toolbox

### Running from Local Source

1. **Configuration:** Create a `tools.yaml` file to configure your sources and
   tools. See the [Configuration section in the
   README](./README.md#Configuration) for details.
1. **CLI Flags:** List available command-line flags for the Toolbox server:

    ```bash
    go run . --help
    ```

1. **Running the Server:** Start the Toolbox server with optional flags. The
   server listens on port 5000 by default.

    ```bash
    go run .
    ```

1. **Testing the Endpoint:** Verify the server is running by sending a request
   to the endpoint:

    ```bash
    curl http://127.0.0.1:5000
    ```

### Tool Naming Conventions

This section details the purpose and conventions for MCP Toolbox's tools naming
properties, **tool name** and **tool type**.

```
kind: tools
name: cancel_hotel <- tool name
type: postgres-sql  <- tool type
source: my_pg_source
```

#### Tool Name

Tool name is the identifier used by a Large Language Model (LLM) to invoke a
specific tool.

* Custom tools: The user can define any name they want. The below guidelines
  do not apply.
* Pre-built tools: The tool name is predefined and cannot be changed. It
should follow the guidelines.

The following guidelines apply to tool names:

* Should use underscores over hyphens (e.g., `list_collections` instead of
  `list-collections`).
* Should not have the product name in the name (e.g., `list_collections` instead
  of `firestore_list_collections`).
* Superficial changes are NOT considered as breaking (e.g., changing tool name).
* Non-superficial changes MAY be considered breaking (e.g. adding new parameters
  to a function) until they can be validated through extensive testing to ensure
  they do not negatively impact agent's performances.

#### Tool Type

Tool type serves as a category or type that a user can assign to a tool.

The following guidelines apply to tool types:

* Should use hyphens over underscores (e.g. `firestore-list-collections` or
  `firestore_list_colelctions`).
* Should use product name in name (e.g. `firestore-list-collections` over
  `list-collections`).
* Changes to tool type are breaking changes and should be avoided.

## Implementation Guides

### Adding a New Database Source or Tool

Please create an
[issue](https://github.com/googleapis/genai-toolbox/issues) before
implementation to ensure we can accept the contribution and no duplicated work.
This issue should include an overview of the API design. If you have any
questions, reach out on our [Discord](https://discord.gg/Dmm69peqjh) to chat
directly with the team.

> [!NOTE]
> New tools can be added for [pre-existing data
> sources](https://github.com/googleapis/genai-toolbox/tree/main/internal/sources).
> However, any new database source should also include at least one new tool
> type.

#### Adding a New Database Source

We recommend looking at an [example source
implementation](https://github.com/googleapis/genai-toolbox/blob/main/internal/sources/postgres/postgres.go).

* **Create a new directory** under `internal/sources` for your database type
  (e.g., `internal/sources/newdb`).
* **Define a configuration struct** for your data source in a file named
  `newdb.go`. Create a `Config` struct to include all the necessary parameters
  for connecting to the database (e.g., host, port, username, password, database
  name) and a `Source` struct to store necessary parameters for tools (e.g.,
  Name, Type, connection object, additional config).
* **Implement the
  [`SourceConfig`](https://github.com/googleapis/genai-toolbox/blob/fd300dc606d88bf9f7bba689e2cee4e3565537dd/internal/sources/sources.go#L57)
  interface**. This interface requires two methods:
  * `SourceConfigType() string`: Returns a unique string identifier for your
    data source (e.g., `"newdb"`).
  * `Initialize(ctx context.Context, tracer trace.Tracer) (Source, error)`:
    Creates a new instance of your data source and establishes a connection to
    the database.
* **Implement the
  [`Source`](https://github.com/googleapis/genai-toolbox/blob/fd300dc606d88bf9f7bba689e2cee4e3565537dd/internal/sources/sources.go#L63)
  interface**. This interface requires one method:
  * `SourceType() string`: Returns the same string identifier as `SourceConfigType()`.
* **Implement `init()`** to register the new Source.
* **Implement Unit Tests** in a file named `newdb_test.go`.

#### Adding a New Tool

> [!NOTE]
> Please follow the tool naming convention detailed
> [here](#tool-naming-conventions).

We recommend looking at an [example tool
implementation](https://github.com/googleapis/genai-toolbox/tree/main/internal/tools/postgres/postgressql).

Remember to keep your PRs small. For example, if you are contributing a new Source, only include one or two core Tools within the same PR, the rest of the Tools can come in subsequent PRs. 

* **Create a new directory** under `internal/tools` for your tool type (e.g., `internal/tools/newdb/newdbtool`).
* **Define a configuration struct** for your tool in a file named `newdbtool.go`.
Create a `Config` struct and a `Tool` struct to store necessary parameters for
tools.
* **Implement the
  [`ToolConfig`](https://github.com/googleapis/genai-toolbox/blob/fd300dc606d88bf9f7bba689e2cee4e3565537dd/internal/tools/tools.go#L61)
  interface**. This interface requires one method:
  * `ToolConfigType() string`: Returns a unique string identifier for your tool
    (e.g., `"newdb-tool"`).
  * `Initialize(sources map[string]Source) (Tool, error)`: Creates a new
    instance of your tool and validates that it can connect to the specified
    data source.
* **Implement the `Tool` interface**. This interface requires the following
  methods:
  * `Invoke(ctx context.Context, params map[string]any) ([]any, error)`:
    Executes the operation on the database using the provided parameters.
  * `ParseParams(data map[string]any, claims map[string]map[string]any)
    (ParamValues, error)`: Parses and validates the input parameters.
  * `Manifest() Manifest`: Returns a manifest describing the tool's capabilities
    and parameters.
  * `McpManifest() McpManifest`: Returns an MCP manifest describing the tool for
    use with the Model Context Protocol.
  * `Authorized(services []string) bool`: Checks if the tool is authorized to
    run based on the provided authentication services.
* **Implement `init()`** to register the new Tool.
* **Implement Unit Tests** in a file named `newdbtool_test.go`.

#### Adding Integration Tests

* **Add a test file** under a new directory `tests/newdb`.
* **Add pre-defined integration test suites** in the
  `/tests/newdb/newdb_integration_test.go` that are **required** to be run as
  long as your code contains related features. Please check each test suites for
  the config defaults, if your source require test suites config updates, please
  refer to [config option](./tests/option.go):

     1. [RunToolGetTest][tool-get]: tests for the `GET` endpoint that returns the
            tool's manifest.

     2. [RunToolInvokeTest][tool-call]: tests for tool calling through the native
        Toolbox endpoints.

     3. [RunMCPToolCallMethod][mcp-call]: tests tool calling through the MCP
            endpoints.

     4. (Optional) [RunExecuteSqlToolInvokeTest][execute-sql]: tests an
        `execute-sql` tool for any source. Only run this test if you are adding an
        `execute-sql` tool.

     5. (Optional) [RunToolInvokeWithTemplateParameters][temp-param]: tests for [template
            parameters][temp-param-doc]. Only run this test if template
            parameters apply to your tool.

* **Add additional tests** for the tools that are not covered by the predefined tests. Every tool must be tested!

* **Add the new database to the integration test workflow** in
  [integration.cloudbuild.yaml](.ci/integration.cloudbuild.yaml).

[tool-get]:
    https://github.com/googleapis/genai-toolbox/blob/v0.23.0/tests/tool.go#L41
[tool-call]:
    https://github.com/googleapis/genai-toolbox/blob/v0.23.0/tests/tool.go#L229
[mcp-call]:
    https://github.com/googleapis/genai-toolbox/blob/v0.23.0/tests/tool.go#L789
[execute-sql]:
    https://github.com/googleapis/genai-toolbox/blob/v0.23.0/tests/tool.go#L609
[temp-param]:
    https://github.com/googleapis/genai-toolbox/blob/v0.23.0/tests/tool.go#L454
[temp-param-doc]:
    https://googleapis.github.io/genai-toolbox/resources/tools/#template-parameters

#### Adding Documentation

* **Update the documentation** to include information about your new data source
  and tool. This includes:
  * Adding a new page to the `docs/en/resources/sources` directory for your data
    source.
  * Adding a new page to the `docs/en/resources/tools` directory for your tool.

* **(Optional) Add samples** to the `docs/en/samples/<newdb>` directory.

#### Adding Prebuilt Tools

You can provide developers with a set of "build-time" tools to aid common
software development user journeys like viewing and creating tables/collections
and data.

* **Create a set of prebuilt tools** by defining a new `tools.yaml` and adding
  it to `internal/tools`. Make sure the file name matches the source (i.e. for
  source "alloydb-postgres" create a file named "alloydb-postgres.yaml").
* **Update `cmd/root.go`** to add new source to the `prebuilt` flag.
* **Add tests** in
  [internal/prebuiltconfigs/prebuiltconfigs_test.go](internal/prebuiltconfigs/prebuiltconfigs_test.go)
  and [cmd/root_test.go](cmd/root_test.go).

## Testing

### Infrastructure

Toolbox uses both GitHub Actions and Cloud Build to run test workflows. Cloud
Build is used when Google credentials are required. Cloud Build uses test
project "toolbox-testing-438616".

### Linting

Run the lint check to ensure code quality:

```bash
golangci-lint run --fix
```

### Unit Tests

Execute unit tests locally:

```bash
go test -race -v ./cmd/... ./internal/...
```

### Integration Tests

#### Running Locally

1. **Environment Variables:** Set the required environment variables. Refer to
   the [Cloud Build testing configuration](./.ci/integration.cloudbuild.yaml)
   for a complete list of variables for each source.
    * `SERVICE_ACCOUNT_EMAIL`: Use your own GCP email.
    * `CLIENT_ID`: Use the Google Cloud SDK application Client ID. Contact
      Toolbox maintainers if you don't have it.
1. **Running Tests:** Run the integration test for your target source. Specify
   the required Go build tags at the top of each integration test file.

    ```shell
    go test -race -v ./tests/<YOUR_TEST_DIR>
    ```

    For example, to run the AlloyDB integration test:

    ```shell
    go test -race -v ./tests/alloydbpg
    ```

1. **Timeout:** The integration test should have a timeout on the server.
   Look for code like this:

   ```go
   ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
   defer cancel()

   cmd, cleanup, err := tests.StartCmd(ctx, toolsFile, args...)
   if err != nil {
     t.Fatalf("command initialization returned an error: %s", err)
   }
   defer cleanup()
   ```

   Be sure to set the timeout to a reasonable value for your tests.

#### Running on Pull Requests

* **Internal Contributors:** Testing workflows should trigger automatically.
* **External Contributors:** Request Toolbox maintainers to trigger the testing
  workflows on your PR.
  * Maintainers can comment `/gcbrun` to execute the integration tests.
  * Maintainers can add the label `tests:run` to execute the unit tests.

#### Test Resources

The following databases have been added as test resources. To add a new database
to test against, please contact the Toolbox maintainer team via an issue or PR.
Refer to the [Cloud Build testing
configuration](./.ci/integration.cloudbuild.yaml) for a complete list of
variables for each source.

* AlloyDB - setup in the test project
  * AI Natural Language ([setup
    instructions](https://cloud.google.com/alloydb/docs/ai/use-natural-language-generate-sql-queries))
    has been configured for `alloydb-ai-nl` tool tests
  * The Cloud Build service account is a user
* Bigtable - setup in the test project
  * The Cloud Build service account is a user
* BigQuery - setup in the test project
  * The Cloud Build service account is a user
* Cloud SQL Postgres - setup in the test project
  * The Cloud Build service account is a user
* Cloud SQL MySQL - setup in the test project
  * The Cloud Build service account is a user
* Cloud SQL SQL Server - setup in the test project
  * The Cloud Build service account is a user
* Couchbase - setup in the test project via the Marketplace
* DGraph - using the public dgraph interface <https://play.dgraph.io> for
  testing
* Looker
  * The Cloud Build service account is a user for conversational analytics
  * The Looker instance runs under google.com:looker-sandbox.
* Memorystore Redis - setup in the test project using a Memorystore for Redis
  standalone instance
  * Memorystore Redis Cluster, Memorystore Valkey standalone, and Memorystore
    Valkey Cluster instances all require PSC connections, which requires extra
    security setup to connect from Cloud Build. Memorystore Redis standalone is
    the only one allowing PSA connection.
  * The Cloud Build service account is a user
* Memorystore Valkey - setup in the test project using a Memorystore for Redis
  standalone instance
  * The Cloud Build service account is a user
* MySQL - setup in the test project using a Cloud SQL instance
* Neo4j - setup in the test project on a GCE VM
* Postgres - setup in the test project using an AlloyDB instance
* Spanner - setup in the test project
  * The Cloud Build service account is a user
* SQL Server - setup in the test project using a Cloud SQL instance
* SQLite -  setup in the integration test, where we create a temporary database
  file

### Link Checking and Fixing with Lychee

We use **[lychee](https://github.com/lycheeverse/lychee-action)** for repository link checks.

* To run the checker **locally**, see the [command-line usage guide](https://github.com/lycheeverse/lychee?tab=readme-ov-file#commandline-usage).

####  Fixing Broken Links

1.  **Update the Link:** Correct the broken URL or update the content where it is used.
2.  **Ignore the Link:** If you can't fix the link (e.g., due to **external rate-limits** or if it's a **local-only URL**), tell Lychee to **ignore** it.

    * List **regular expressions** or **direct links** in the **[.lycheeignore](https://github.com/googleapis/genai-toolbox/blob/main/.lycheeignore)** file, one entry per line.
    * **Always add a comment** explaining **why** the link is being skipped to prevent link rot. **Example `.lycheeignore`:**
       ```text
       # These are email addresses, not standard web URLs, and usually cause check failures.
       ^mailto:.*
       ```
> [!NOTE]
> To avoid build failures in GitHub Actions, follow the linking pattern demonstrated here: <br>
> **Avoid:** (Works in Hugo, breaks Link Checker): `[Read more](docs/setup)` or `[Read more](docs/setup/)` <br>
> **Reason:** The link checker cannot find a file named "setup" or a directory with that name containing an index. <br>
> **Preferred:** `[Read more](docs/setup.md)` <br>
> **Reason:** The GitHub Action finds the physical file. Hugo then uses its internal logic (or render hooks) to resolve this to the correct `/docs/setup/` web URL. <br>

### Other GitHub Checks

* License header check (`.github/header-checker-lint.yml`) - Ensures files have
  the appropriate license
* CLA/google - Ensures the developer has signed the CLA:
  <https://cla.developers.google.com/>
* conventionalcommits.org - Ensures the commit messages are in the correct
  format. This repository uses tool [Release
  Please](https://github.com/googleapis/release-please) to create GitHub
  releases. It does so by parsing your git history, looking for [Conventional
  Commit messages](https://www.conventionalcommits.org/), and creating release
  PRs. Learn more by reading [How should I write my
  commits?](https://github.com/googleapis/release-please?tab=readme-ov-file#how-should-i-write-my-commits)

## Developing Documentation

### Running a Local Hugo Server

Follow these steps to preview documentation changes locally using a Hugo server:

1. **Install Hugo:** Ensure you have
   [Hugo](https://gohugo.io/installation/macos/) extended edition version
   0.146.0 or later installed.
1. **Navigate to the Hugo Directory:**

    ```bash
    cd .hugo
    ```

1. **Install Dependencies:**

    ```bash
    npm ci
    ```

1. **Start the Server:**

    ```bash
    hugo server
    ```

### Previewing Documentation on Pull Requests

### Document Versioning Setup

There are 3 GHA workflows we use to achieve document versioning:

1. **Deploy In-development docs:**
    This workflow is run on every commit merged into the main branch. It deploys
    the built site to the `/dev/` subdirectory for the in-development
    documentation.

1. **Deploy Versioned Docs:**
    When a new GitHub Release is published, it performs two deployments based on
    the new release tag. One to the new version subdirectory and one to the root
    directory of the versioned-gh-pages branch.

    **Note:** Before the release PR from release-please is merged, add the
    newest version into the hugo.toml file.

1. **Deploy Previous Version Docs:**
    This is a manual workflow, started from the GitHub Actions UI.
    To rebuild and redeploy documentation for an already released version that
    were released before this new system was in place. This workflow can be
    started on the UI by providing the git version tag which you want to create
    the documentation for. The specific versioned subdirectory and the root docs
    are updated on the versioned-gh-pages branch.

#### Contributors

Request a repo owner to run the preview deployment workflow on your PR. A
preview link will be automatically added as a comment to your PR.


#### Maintainers

1. **Inspect Changes:** Review the proposed changes in the PR to ensure they are
   safe and do not contain malicious code. Pay close attention to changes in the
   `.github/workflows/` directory.
1. **Deploy Preview:** Apply the `docs: deploy-preview` label to the PR to
   deploy a documentation preview.

### Shortcodes

This repository includes custom shortcodes to help with documentation consistency and maintenance.
For more information on how they work, see the [Hugo Shortcodes](https://gohugo.io/content-management/shortcodes/) documentation and the guide to [creating custom shortcodes](https://gohugo.io/templates/shortcode/).

#### `include` Shortcode

The `include` shortcode reads a file and optionally fences it with a language.

**Syntax:**
`{{< include "path/to/file" "language" >}}`

**Example:**
`{{< include "static/headers/license_header.txt" >}}`
`{{< include "samples/program.js" "javascript" >}}`

**Source:** [.hugo/layouts/shortcodes/include.html](.hugo/layouts/shortcodes/include.html)

#### `regionInclude` Shortcode

The `regionInclude` shortcode reads a file, extracts content between `[START region_name]` and `[END region_name]`, and optionally fences it.

**Syntax:**
`{{< regionInclude "path/to/file" "region_name" "language" >}}`

**Example Markdown:**
`{{< regionInclude "samples/program.js" "program_setup" "javascript" >}}`

**Example Code Snippet (`samples/program.js`):**
```javascript
// [START program_setup]
import { Toolbox } from '@googleapis/genai-toolbox';
const toolbox = new Toolbox();
// [END program_setup]
```

**Source:** [.hugo/layouts/shortcodes/regionInclude.html](.hugo/layouts/shortcodes/regionInclude.html)

## Building Toolbox

### Building the Binary

1. **Build Command:** Compile the Toolbox binary:

    ```bash
    go build -o toolbox
    ```

1. **Running the Binary:** Execute the compiled binary with optional flags. The
   server listens on port 5000 by default:

    ```bash
    ./toolbox
    ```

1. **Testing the Endpoint:** Verify the server is running by sending a request
   to the endpoint:

    ```bash
    curl http://127.0.0.1:5000
    ```

### Building Container Images

1. **Build Command:** Build the Toolbox container image:

    ```bash
    docker build -t toolbox:dev .
    ```

1. **View Image:** List available Docker images to confirm the build:

    ```bash
    docker images
    ```

1. **Run Container:** Run the Toolbox container image using Docker:

    ```bash
    docker run -d toolbox:dev
    ```

## Developing Toolbox SDKs

Refer to the [SDK developer
guide](https://github.com/googleapis/mcp-toolbox-sdk-python/blob/main/DEVELOPER.md)
for instructions on developing Toolbox SDKs.

## Maintainer Information

### Team

Team `@googleapis/senseai-eco` has been set as
[CODEOWNERS](.github/CODEOWNERS). The GitHub TeamSync tool is used to create
this team from MDB Group, `senseai-eco`. Additionally, database-specific GitHub
teams (e.g., `@googleapis/toolbox-alloydb`) have been created from MDB groups to
manage code ownership and review for individual database products.

### Issue/PR Triage and SLO
After an issue is created, maintainers will assign the following labels:
* `Priority` (defaulted to P0)
* `Type` (if applicable)
* `Product` (if applicable)

All incoming issues and PRs will follow the following SLO:
| Type            | Priority | Objective                                                              |
|-----------------|----------|------------------------------------------------------------------------|
| Feature Request | P0       | Must respond within **5 days**                                         |
| Process         | P0       | Must respond within **5 days**                                         |
| Bugs            | P0       | Must respond within **5 days**, and resolve/closure within **14 days** |
| Bugs            | P1       | Must respond within **7 days**, and resolve/closure within **90 days** |
| Bugs            | P2       | Must respond within **30 days**

_Types that are not listed in the table do not adhere to any SLO._

### Releasing

Toolbox has two types of releases: versioned and continuous. It uses Google
Cloud project, `database-toolbox`.

* **Versioned Release:** Official, supported distributions tagged as `latest`.
  The release process is defined in
  [versioned.release.cloudbuild.yaml](.ci/versioned.release.cloudbuild.yaml).
* **Continuous Release:** Used for early testing of features between official
  releases and for end-to-end testing. The release process is defined in
  [continuous.release.cloudbuild.yaml](.ci/continuous.release.cloudbuild.yaml).
* **GitHub Release:** `.github/release-please.yml` automatically creates GitHub
  Releases and release PRs.

### How-to Release a new Version

1. [Optional] If you want to override the version number, send a
   [PR](https://github.com/googleapis/genai-toolbox/pull/31) to trigger
   [release-please](https://github.com/googleapis/release-please?tab=readme-ov-file#how-do-i-change-the-version-number).
   You can generate a commit with the following line: `git commit -m "chore:
   release 0.1.0" -m "Release-As: 0.1.0" --allow-empty`
1. [Optional] If you want to edit the changelog, send commits to the release PR
1. Approve and merge the PR with the title “[chore(main): release
   x.x.x](https://github.com/googleapis/genai-toolbox/pull/16)”
1. The
   [trigger](https://pantheon.corp.google.com/cloud-build/triggers;region=us-central1/edit/27bd0d21-264a-4446-b2d7-0df4e9915fb3?e=13802955&inv=1&invt=AbhU8A&mods=logs_tg_staging&project=database-toolbox)
   should automatically run when a new tag is pushed. You can view [triggered
   builds here to check the
   status](https://pantheon.corp.google.com/cloud-build/builds;region=us-central1?query=trigger_id%3D%2227bd0d21-264a-4446-b2d7-0df4e9915fb3%22&e=13802955&inv=1&invt=AbhU8A&mods=logs_tg_staging&project=database-toolbox)
1. Update the Github release notes to include the following table:
    1. Run the following command (from the root directory):

        ```
        export VERSION="v0.0.0"
        .ci/generate_release_table.sh
        ```

    1. Copy the table output
    1. In the GitHub UI, navigate to Releases and click the `edit` button.
    1. Paste the table at the bottom of release note and click `Update release`.
1. Post release in internal chat and on Discord.

#### Supported Binaries

The following operating systems and architectures are supported for binary
releases:

* linux/amd64
* darwin/arm64
* darwin/amd64
* windows/amd64

#### Supported Container Images

The following base container images are supported for container image releases:

* distroless

### Automated Tests

Integration and unit tests are automatically triggered via Cloud Build on each
pull request. Integration tests run on merge and nightly.

#### Failure notifications

On-merge and nightly tests that fail have notification setup via Cloud Build
Failure Reporter [GitHub Actions
Workflow](.github/workflows/schedule_reporter.yml).

#### Trigger Setup

Configure a Cloud Build trigger using the UI or `gcloud` with the following
settings:

* **Event:** Pull request
* **Region:** global (for default worker pools)
* **Source:**
  * Generation: 1st gen
  * Repo: googleapis/genai-toolbox (GitHub App)
  * Base branch: `^main$`
* **Comment control:** Required except for owners and collaborators
* **Filters:** Add directory filter
* **Config:** Cloud Build configuration file
  * Location: Repository (add path to file)
* **Service account:** Set for demo service to enable ID token creation for
  authenticated services

### Triggering Tests

Trigger pull request tests for external contributors by:

* **Cloud Build tests:** Comment `/gcbrun`
* **Unit tests:** Add the `tests:run` label

## Repo Setup & Automation

* .github/blunderbuss.yml - Auto-assign issues and PRs from GitHub teams. Use a
  product label to assign to a product-specific team member.
* .github/renovate.json5 - Tooling for dependency updates. Dependabot is built
  into the GitHub repo for GitHub security warnings
* go/github-issue-mirror - GitHub issues are automatically mirrored into buganizer
* (Suspended) .github/sync-repo-settings.yaml - configure repo settings
* .github/release-please.yml - Creates GitHub releases
* .github/ISSUE_TEMPLATE - templates for GitHub issues

