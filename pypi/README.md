# MCP Toolbox Server - Python Package

This package provides the MCP Toolbox Server as a Python package, bundling the pre-compiled Go binary for your platform. It is designed to be executed directly via `uvx` or run as a standalone server.

## Usage

### Run with `uvx` (Recommended)

You can run the MCP Toolbox Server instantly without installing it manually using `uvx`:

```bash
uvx toolbox-server
```

### Manual Installation

Alternatively, you can install the package in your Python environment:

```bash
uv pip install toolbox-server
# or
pip install toolbox-server
```

After installation, run the server using the `toolbox-server` command:

```bash
toolbox-server
```
