# Requirements

To use `mcpd`, ensure the following tools are installed:

| Tool           | Purpose                                        | Notes                                                             | 
|----------------|------------------------------------------------|-------------------------------------------------------------------| 
| `Go >= 1.24.4` | Required for building `mcpd` and running tests | https://go.dev/doc/install                                        | 
| `uv`           | for running `uvx` Python packages              | https://docs.astral.sh/uv/getting-started/installation/           |
| `npx`          | for running JavaScript/TypeScript packages     | https://docs.npmjs.com/downloading-and-installing-node-js-and-npm |

!!! note "Internet Connectivity"
    `mcpd` requires internet access to contact package registries and to allow MCP servers access to the internet if required when running.
