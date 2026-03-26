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
-   `docs/en`: Project documentation. Separated logically into:
    - `documentation/`: Documentation and concepts (Section I).
    - `integrations/`: Reference architectures for DB connectivity and tools (Section II).
    - `samples/`: Tutorials and code samples (Section III).
    - `reference/`: CLI info and FAQs (Section IV).

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
3.  **Generate Search Index:** Because Pagefind requires physical files, `hugo server` alone will not populate the search bar. Build the local index first (using the development environment to block analytics) by running:
    `hugo --environment development && npx pagefind --site public --output-path static/pagefind`
4.  Start server: `hugo server`

### Versioning Workflows

Documentation builds automatically generate standard HTML alongside AI-friendly text files (`llms.txt` and `llms-full.txt`).

There are 6 workflows in total, handling parallel deployments to both GitHub Pages and Cloudflare Pages. **All deployment workflows automatically execute `npx pagefind --site public` to generate version-scoped search indexes.**

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

-   **For a new source:** Add source documentation to `docs/en/integrations/<source_name>/source.md`. Ensure the root `_index.md` file contains **strictly only frontmatter** and no markdown body text.
-   **For a new native tool:** Add tool documentation to `docs/en/integrations/<source_name>/tools/<tool_name>.md`. Ensure the `tools/_index.md` file contains **strictly only frontmatter**.
-   **Adding Integration Samples:** Add integration-specific samples to `docs/en/integrations/<source_name>/samples/`. Ensure the `samples/_index.md` file contains **strictly only frontmatter**.
-   **Tool Inheritance (Shared Tools):** Managed databases (e.g., Cloud SQL Postgres) that use the tools of their underlying engine (e.g., Postgres) map their inherited tools by utilizing the `shared_tools` frontmatter parameter inside their `tools/_index.md` file. This file must contain only frontmatter.
-   **New Top-Level Directories:** If adding a completely new top-level section to the documentation site, you must update the "Diátaxis Narrative Framework" section inside both `.hugo/layouts/index.llms.txt` and `.hugo/layouts/index.llms-full.txt` to keep the AI context synced with the site structure.


#### Integration Documentation Rules

When generating or editing documentation for this repository, you must strictly adhere to the following CI-enforced rules. Failure to do so will break the build.

##### Source Page Constraints (`integrations/**/source.md`)

1.  **File Naming:** The primary connection guide for a source must be named `source.md`. Use `_index.md` solely as an empty structural folder wrapper containing **only YAML frontmatter**.
2.  **LinkTitle:** The linkTitle has to be set to the string `Source` always.
3.  **Title Convention:** The YAML frontmatter `title` must always end with "Source" (e.g., `title: "Postgres Source"`).
4.  **No H1 Tags:** Never generate H1 (`#`) headings in the markdown body.
5.  **Strict H2 Ordering:** You must use the following H2 (`##`) headings in this exact sequence.
    *   `## About` (Required)
    *   `## Available Tools` (Optional)
    *   `## Requirements` (Optional)
    *   `## Example` (Required)
    *   `## Reference` (Required)
    *   `## Advanced Usage` (Optional)
    *   `## Troubleshooting` (Optional)
    *   `## Additional Resources` (Optional)
6.  **Shortcode Placement:** If you generate the `## Available Tools` section, you must include the `{{< list-tools >}}` shortcode beneath it.

##### Tool Page Constraints (`integrations/**/tools/*.md`)

1.  **Location:** All native tools must reside inside a nested `tools/` subdirectory. The `tools/` directory must contain an `_index.md` file consisting **strictly of frontmatter**.
2.  **Title Convention:** The YAML frontmatter `title` must always end with "Tool" (e.g., `title: "Execute SQL Tool"`).
3.  **No H1 Tags:** Never generate H1 (`#`) headings in the markdown body.
4.  **Strict H2 Ordering:** You must use the following H2 (`##`) headings in this exact sequence.
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
5.  **Shortcode Placement:** If you generate the `## Compatible Sources` section, you must include the `{{< compatible-sources >}}` shortcode beneath it.

##### Samples Architecture Constraints
Sample code is aggregated visually in the UI via the Samples section, but the physical markdown files are distributed logically based on their scope. When adding samples, use the correct location and apply proper frontmatter tags (`sample_filters`, `is_sample`) are added so the UI Gallery filters can index them.for the gallery filtering:
1.  **Quickstarts:** `docs/en/documentation/getting-started/`
2.  **Integration-Specific Samples:** `docs/en/integrations/<source_name>/samples/`. (The `samples/_index.md` wrapper must contain **strictly only frontmatter**).
3.  **General/Cross-Category Samples:** `docs/en/samples/`


##### Asset Constraints (`docs/`)

1.  **File Size Limits:** Never add files larger than 24MB to the `docs/` directory.
