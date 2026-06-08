# Requirements

`mcpd` does not require every runtime on every machine.
Install the `mcpd` binary, then install the runtime(s) needed by the MCP servers you plan to run.

## Runtime requirements

Use the package prefix in `.mcpd.toml` to tell which runtime a server needs:

| Package Prefix | Install | Required When | URL |
|----------------|---------|---------------|-----|
| `uvx::` | `uv` | At least one configured server package starts with `uvx::` | [https://docs.astral.sh/uv/getting-started/installation/](https://docs.astral.sh/uv/getting-started/installation/) |
| `npx::` | Node.js and `npx` | At least one configured server package starts with `npx::`, or you want to use `mcpd inspector` | [https://docs.npmjs.com/downloading-and-installing-node-js-and-npm](https://docs.npmjs.com/downloading-and-installing-node-js-and-npm) |
| `docker::` | Docker | At least one configured server package starts with `docker::`, or you want to run `mcpd` itself in Docker | [https://www.docker.com/products/docker-desktop/](https://www.docker.com/products/docker-desktop/) |

!!! tip "Quick start and tutorial"
    The quick start and tutorial use the `time` server through `uvx`, so `uv` is required for those examples.

## Optional tools used in examples

| Tool | Why You Might Want It | URL |
|------|-----------------------|-----|
| `curl` | Calling the `mcpd` HTTP API from the shell | [https://curl.se/](https://curl.se/) |
| `jq` | Pretty-printing JSON responses in shell examples | [https://jqlang.org/](https://jqlang.org/) |

## Development-only requirements

| Tool | Purpose | URL |
|------|---------|-----|
| `Go >= 1.26.0` | Building `mcpd`, running tests, and contributing to the Go codebase | [https://go.dev/doc/install](https://go.dev/doc/install) |
| `uv` | Serving the local docs site via `make docs` | [https://docs.astral.sh/uv/getting-started/installation/](https://docs.astral.sh/uv/getting-started/installation/) |

!!! note "Internet Connectivity"
    `mcpd` typically needs internet access to resolve packages from remote registries and to download server dependencies on first run.
