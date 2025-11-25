# How to contribute

We'd love to accept your patches and contributions to this project.

## Before you begin

### Sign our Contributor License Agreement

Contributions to this project must be accompanied by a
[Contributor License Agreement](https://cla.developers.google.com/about) (CLA).
You (or your employer) retain the copyright to your contribution; this simply
gives us permission to use and redistribute your contributions as part of the
project.

If you or your current employer have already signed the Google CLA (even if it
was for a different project), you probably don't need to do it again.

Visit <https://cla.developers.google.com/> to see your current agreements or to
sign a new one.

### Review our community guidelines

This project follows
[Google's Open Source Community Guidelines](https://opensource.google/conduct/).

## Contribution process

> [!NOTE]
> New contributions should always include both unit and integration tests.

All submissions, including submissions by project members, require review. We
use GitHub pull requests for this purpose. Consult
[GitHub Help](https://help.github.com/articles/about-pull-requests/) for more
information on using pull requests.

### Code reviews

* Within 2-5 days, a reviewer will review your PR. They may approve it, or request
changes.
* When requesting changes, reviewers should self-assign the PR to ensure
they are aware of any updates.
* If additional changes are needed, push additional commits to your PR branch -
this helps the reviewer know which parts of the PR have changed.
* Commits will be
squashed when merged.
* Please follow up with changes promptly.
* If a PR is awaiting changes by the
author for more than 10 days, maintainers may mark that PR as Draft. PRs that
are inactive for more than 30 days may be closed.

### Automated Code Reviews

This repository uses **Gemini Code Assist** to provide automated code reviews on Pull Requests. While this does not replace human review, it provides immediate feedback on code quality and potential issues.

You can manually trigger the bot by commenting on your Pull Request:

*   `/gemini`: Manually invokes Gemini Code Assist in comments
*   `/gemini review`: Posts a code review of the changes in the pull request
*   `/gemini summary`: Posts a summary of the changes in the pull request.
*   `/gemini help`: Overview of the available commands

## Adding a New Database Source or Tool

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

### Adding a New Database Source

We recommend looking at an [example source
implementation](https://github.com/googleapis/genai-toolbox/blob/main/internal/sources/postgres/postgres.go).

* **Create a new directory** under `internal/sources` for your database type
  (e.g., `internal/sources/newdb`).
* **Define a configuration struct** for your data source in a file named
  `newdb.go`. Create a `Config` struct to include all the necessary parameters
  for connecting to the database (e.g., host, port, username, password, database
  name) and a `Source` struct to store necessary parameters for tools (e.g.,
  Name, Kind, connection object, additional config).
* **Implement the
  [`SourceConfig`](https://github.com/googleapis/genai-toolbox/blob/fd300dc606d88bf9f7bba689e2cee4e3565537dd/internal/sources/sources.go#L57)
  interface**. This interface requires two methods:
  * `SourceConfigKind() string`: Returns a unique string identifier for your
    data source (e.g., `"newdb"`).
  * `Initialize(ctx context.Context, tracer trace.Tracer) (Source, error)`:
    Creates a new instance of your data source and establishes a connection to
    the database.
* **Implement the
  [`Source`](https://github.com/googleapis/genai-toolbox/blob/fd300dc606d88bf9f7bba689e2cee4e3565537dd/internal/sources/sources.go#L63)
  interface**. This interface requires one method:
  * `SourceKind() string`: Returns the same string identifier as `SourceConfigKind()`.
* **Implement `init()`** to register the new Source.
* **Implement Unit Tests** in a file named `newdb_test.go`.

### Adding a New Tool

> [!NOTE]
> Please follow the tool naming convention detailed
> [here](./DEVELOPER.md#tool-naming-conventions).

We recommend looking at an [example tool
implementation](https://github.com/googleapis/genai-toolbox/tree/main/internal/tools/postgres/postgressql).

* **Create a new directory** under `internal/tools` for your tool type (e.g., `internal/tools/newdb/newdbtool`).
* **Define a configuration struct** for your tool in a file named `newdbtool.go`.
Create a `Config` struct and a `Tool` struct to store necessary parameters for
tools.
* **Implement the
  [`ToolConfig`](https://github.com/googleapis/genai-toolbox/blob/fd300dc606d88bf9f7bba689e2cee4e3565537dd/internal/tools/tools.go#L61)
  interface**. This interface requires one method:
  * `ToolConfigKind() string`: Returns a unique string identifier for your tool
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

### Adding Integration Tests

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

* **Add the new database to the integration test workflow** in
  [integration.cloudbuild.yaml](.ci/integration.cloudbuild.yaml).

[tool-get]:
    https://github.com/googleapis/genai-toolbox/blob/fd300dc606d88bf9f7bba689e2cee4e3565537dd/tests/tool.go#L31
[tool-call]:
    <https://github.com/googleapis/genai-toolbox/blob/fd300dc606d88bf9f7bba689e2cee4e3565537dd/tests/tool.go#L79>
[mcp-call]:
    https://github.com/googleapis/genai-toolbox/blob/fd300dc606d88bf9f7bba689e2cee4e3565537dd/tests/tool.go#L554
[execute-sql]:
    <https://github.com/googleapis/genai-toolbox/blob/fd300dc606d88bf9f7bba689e2cee4e3565537dd/tests/tool.go#L431>
[temp-param]:
    <https://github.com/googleapis/genai-toolbox/blob/fd300dc606d88bf9f7bba689e2cee4e3565537dd/tests/tool.go#L297>
[temp-param-doc]:
    https://googleapis.github.io/genai-toolbox/resources/tools/#template-parameters

### Adding Documentation

* **Update the documentation** to include information about your new data source
  and tool. This includes:
  * Adding a new page to the `docs/en/resources/sources` directory for your data
    source.
  * Adding a new page to the `docs/en/resources/tools` directory for your tool.

* **(Optional) Add samples** to the `docs/en/samples/<newdb>` directory.

### (Optional) Adding Prebuilt Tools

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

## Submitting a Pull Request

Submit a pull request to the repository with your changes. Be sure to include a
detailed description of your changes and any requests for long term testing
resources.

* **Title:** All pull request title should follow the formatting of
  [Conventional
  Commit](https://www.conventionalcommits.org/) guidelines: `<type>[optional
  scope]: description`. For example, if you are adding a new field in postgres
  source, the title should be `feat(source/postgres): add support for
  "new-field" field in postgres source`.
  
  Here are some commonly used `type` in this GitHub repo.

  |     **type**    |                                **description**                                                        |
  |-----------------|-------------------------------------------------------------------------------------------------------|
  | Breaking Change | Anything with this type of a `!` after the type/scope introduces a breaking change.                   |
  | feat            | Adding a new feature to the codebase.                                                                 |
  | fix             | Fixing a bug or typo in the codebase. This does not include fixing docs.                              |
  | test            | Changes made to test files.                                                                           |
  | ci              | Changes made to the cicd configuration files or scripts.                                              |
  | docs            | Documentation-related PRs, including fixes on docs.                                                   |
  | chore           | Other small tasks or updates that don't fall into any of the above types.                             |
  | refactor        | Change src code but unlike feat, there are no tests broke and no line lost coverage.                  |
  | revert          | Revert changes made in another commit.                                                                |
  | style           | Update src code, with only formatting and whitespace updates (e.g. code formatter or linter changes). |

  Pull requests should always add scope whenever possible. The scope is
  formatted as `<scope-type>/<scope-kind>` (e.g., `sources/postgres`, or
  `tools/mssql-sql`).
  
  Ideally, **each PR covers only one scope**, if this is
  inevitable, multiple scopes can be seaparated with a comma (e.g.
  `sources/postgres,sources/alloydbpg`). If the PR covers multiple `scope-type`
  (such as adding a new database), you can disregard the `scope-type`, e.g.
  `feat(new-db): adding support for new-db source and tool`.

* **PR Description:** PR description should **always** be included. It should
  include a concise description of the changes, it's impact, along with a
  summary of the solution. If the PR is related to a specific issue, the issue
  number should be mentioned in the PR description (e.g. `Fixes #1`).