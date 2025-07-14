# Requirements

To use `mcpd`, ensure the following tools are installed:

| Tool           | Purpose                                                     | Notes                                                             | 
|----------------|-------------------------------------------------------------|-------------------------------------------------------------------| 
| `Docker`       | Required if you want to run `mcpd` in a local container     | https://www.docker.com/products/docker-desktop/                   | 
| `Go >= 1.24.4` | Required for building `mcpd` and running tests              | https://go.dev/doc/install                                        | 
| `uv`           | for running `uvx` Python packages in `mcpd`, and local docs | https://docs.astral.sh/uv/getting-started/installation/           |
| `npx`          | for running JavaScript/TypeScript packages in `mcpd`        | https://docs.npmjs.com/downloading-and-installing-node-js-and-npm |

!!! note "Internet Connectivity"
    `mcpd` requires internet access to contact package registries and to allow MCP servers access to the internet if required when running.
