# Requirements

To use `mcpd`, ensure the following tools are installed:

| Tool           | Purpose                                                     | URL                                                                                                                                    | 
|----------------|-------------------------------------------------------------|----------------------------------------------------------------------------------------------------------------------------------------| 
| `Docker`       | For running MCP servers as Docker containers                | [https://www.docker.com/products/docker-desktop/](https://www.docker.com/products/docker-desktop/)                                     | 
| `Go >= 1.25.0` | Required for building `mcpd` and running tests              | [https://go.dev/doc/install](https://go.dev/doc/install)                                                                               | 
| `uv`           | For running `uvx` Python packages in `mcpd`, and local docs | [https://docs.astral.sh/uv/getting-started/installation/](https://docs.astral.sh/uv/getting-started/installation/)                     |
| `npx`          | For running JavaScript/TypeScript packages in `mcpd`        | [https://docs.npmjs.com/downloading-and-installing-node-js-and-npm](https://docs.npmjs.com/downloading-and-installing-node-js-and-npm) |

!!! note "Internet Connectivity"
    `mcpd` requires internet access to contact package registries and to allow MCP servers access to the internet if required when running.
