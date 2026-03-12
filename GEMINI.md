# MCP Toolbox Context

This file (symlinked as `CLAUDE.md` and `AGENTS.md`) provides context and guidelines for AI agents working on the MCP Toolbox for Databases project. It summarizes key information from `CONTRIBUTING.md` and `DEVELOPER.md`.

## Project Overview

**MCP Toolbox for Databases** is a Go-based project designed to provide Model Context Protocol (MCP) tools for various data sources and services. It allows Large Language Models (LLMs) to interact with databases and other tools safely and efficiently.

## Tech Stack

-   **Language:** Go (1.23+)
-   **Documentation:** Hugo (Extended Edition v0.146.0+)
-   **Containerization:** Docker
-   **CI/CD:** GitHub Actions, Google Cloud Build
-   **Linting:** `golangci-lint`

## Key Directories

-   `cmd/`: Application entry points.
-   `internal/sources/`: Implementations of database sources (e.g., Postgres, BigQuery).
-   `internal/tools/`: Implementations of specific tools for each source.
-   `tests/`: Integration tests.
-   `docs/`: Project documentation (Hugo site).

## Development Workflow

### Prerequisites

-   Go 1.23 or later.
-   Docker (for building container images and running some tests).
-   Access to necessary Google Cloud resources for integration testing (if applicable).

### Building and Running

1.  **Build Binary:** `go build -o toolbox`
2.  **Run Server:** `go run .` (Listens on port 5000 by default)
3.  **Run with Help:** `go run . --help`
4.  **Test Endpoint:** `curl http://127.0.0.1:5000`

### Testing

-   **Unit Tests:** `go test -race -v ./cmd/... ./internal/...`
-   **Integration Tests:**
    -   Run specific source tests: `go test -race -v ./tests/<source_dir>`
    -   Example: `go test -race -v ./tests/alloydbpg`
    -   Add new sources to `.ci/integration.cloudbuild.yaml`
-   **Linting:** `golangci-lint run --fix`

## Developing Documentation

### Prerequisites

-   Hugo (Extended Edition v0.146.0+)
-   Node.js (for `npm ci`)

### Running Local Server

1.  Navigate to `.hugo` directory: `cd .hugo`
2.  Install dependencies: `npm ci`
3.  Start server: `hugo server`

### Versioning Workflows

Documentation builds automatically generate standard HTML alongside AI-friendly text files (`llms.txt` and `llms-full.txt`).

There are 6 workflows in total, handling parallel deployments to both GitHub Pages and Cloudflare Pages.

1.  **Deploy In-development docs**: Commits merged to `main` deploy to the `/dev/` path. Automatically defaults to version `Dev`.
2.  **Deploy Versioned Docs**: New GitHub releases deploy to `/<version>/` and the root path. The release tag is automatically injected into the build as the documentation version. *(Note: Developers must manually add the new version to the `[[params.versions]]` dropdown array in `hugo.toml` prior to merging a release PR).*
3.  **Deploy Previous Version Docs**: A manual workflow to rebuild older versions by explicitly passing the target tag via the GitHub Actions UI.

## Coding Conventions

### Tool Naming

-   **Tool Name:** `snake_case` (e.g., `list_collections`, `run_query`).
    -   Do *not* include the product name (e.g., avoid `firestore_list_collections`).
-   **Tool Type:** `kebab-case` (e.g., `firestore-list-collections`).
    -   *Must* include the product name.

### Branching and Commits

-   **Branch Naming:** `feat/`, `fix/`, `docs/`, `chore/` (e.g., `feat/add-gemini-md`).
-   **Commit Messages:** [Conventional Commits](https://www.conventionalcommits.org/) format.
    -   Format: `<type>(<scope>): <description>`
    -   Example: `feat(source/postgres): add new connection option`
    -   Types: `feat`, `fix`, `docs`, `chore`, `test`, `ci`, `refactor`, `revert`, `style`.

## Adding New Features

### Adding a New Data Source

1.  Create a new directory: `internal/sources/<newdb>`.
2.  Define `Config` and `Source` structs in `internal/sources/<newdb>/<newdb>.go`.
3.  Implement `SourceConfig` interface (`SourceConfigType`, `Initialize`).
4.  Implement `Source` interface (`SourceType`).
5.  Implement `init()` to register the source.
6.  Add unit tests in `internal/sources/<newdb>/<newdb>_test.go`.

### Adding a New Tool

1.  Create a new directory: `internal/tools/<newdb>/<toolname>`.
2.  Define `Config` and `Tool` structs.
3.  Implement `ToolConfig` interface (`ToolConfigType`, `Initialize`).
4.  Implement `Tool` interface (`Invoke`, `ParseParams`, `Manifest`, `McpManifest`, `Authorized`).
5.  Implement `init()` to register the tool.
6.  Add unit tests.

### Adding Documentation

-   For a new source: Add source documentation to `docs/en/integrations/<source_name>/`. Be sure to include the `{{< list-tools >}}` shortcode on this page to dynamically display its available tools.
-   For a new tool: Add tool documentation to `docs/en/integrations/<source_name>/<tool_name>`. Be sure to include the `{{< compatible-sources >}}` shortcode on this page to list its supported data sources.
-   **New Top-Level Directories:** If adding a completely new top-level section to the documentation site, you must update the "Diátaxis Narrative Framework" section inside both `.hugo/layouts/index.llms.txt` and `.hugo/layouts/index.llms-full.txt` to keep the AI context synced with the site structure.


#### Integration Documentation Rules

When generating or editing documentation for this repository, you must strictly adhere to the following CI-enforced rules. Failure to do so will break the build.

##### Source Page Constraints (`integrations/**/_index.md`)

1.  **Title Convention:** The YAML frontmatter `title` must always end with "Source" (e.g., `title: "Postgres Source"`).
2.  **No H1 Tags:** Never generate H1 (`#`) headings in the markdown body.
3.  **Strict H2 Ordering:** You must use the following H2 (`##`) headings in this exact sequence.
    *   `## About` (Required)
    *   `## Available Tools` (Optional)
    *   `## Requirements` (Optional)
    *   `## Example` (Required)
    *   `## Reference` (Required)
    *   `## Advanced Usage` (Optional)
    *   `## Troubleshooting` (Optional)
    *   `## Additional Resources` (Optional)
4.  **Shortcode Placement:** If you generate the `## Available Tools` section, you must include the `{{< list-tools >}}` shortcode beneath it.

##### Tool Page Constraints (`integrations/**/*.md`)

1.  **Title Convention:** The YAML frontmatter `title` must always end with "Tool" (e.g., `title: "Execute SQL Tool"`).
2.  **No H1 Tags:** Never generate H1 (`#`) headings in the markdown body.
3.  **Strict H2 Ordering:** You must use the following H2 (`##`) headings in this exact sequence.
    *   `## About` (Required)
    *   `## Compatible Sources` (Optional)
    *   `## Requirements` (Optional)
    *   `## Parameters` (Optional)
    *   `## Example` (Required)
    *   `## Output Format` (Optional)
    *   `## Reference` (Optional)
    *   `## Advanced Usage` (Optional)
    *   `## Troubleshooting` (Optional)
    *   `## Additional Resources` (Optional)
4.  **Shortcode Placement:** If you generate the `## Compatible Sources` section, you must include the `{{< compatible-sources >}}` shortcode beneath it.

##### Asset Constraints (`docs/`)

1.  **File Size Limits:** Never add files larger than 24MB to the `docs/` directory.
