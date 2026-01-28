
<p align="center">
  <picture>
    <img src="docs/assets/mcpd-logo.png" alt="mcpd" width="200"/>
  </picture>
</p>

<div align="center"><b>Run your agents, not your infrastructure.</b></div>


Built by [Mozilla AI](https://mozilla.ai)

📚 [mcpd official docs](https://mozilla-ai.github.io/mcpd/)

---

`mcpd` is a daemon that manages your MCP servers via declarative configuration, exposing them as clean HTTP endpoints. This bridges the gap between your agents and your infrastructure, handling the messy work of lifecycle management, secret injection, and environment promotion so you don't have to.

## ⚙️ How it Works

Under the hood, mcpd spawns MCP servers as STDIO subprocesses and proxies requests over HTTP.

<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="docs/assets/mcpd-architecture-dark.svg">
    <source media="(prefers-color-scheme: light)" srcset="docs/assets/mcpd-architecture-light.svg">
    <img alt="mcpd Architecture Diagram" src="docs/assets/mcpd-architecture-light.svg" width="800">
  </picture>
</p>

## 🚀 Quick Start

### Prerequisites

You need the following installed before running `mcpd`:
- [**Docker**](https://www.docker.com/products/docker-desktop/)
- [**npx**](https://docs.npmjs.com/downloading-and-installing-node-js-and-npm) (Node.js package runner)
- [**uvx**](https://docs.astral.sh/uv/getting-started/installation/) (Universal virtual environment manager)


### Installation

#### via Homebrew

(Works for both macOS and Linux)

Add the Mozilla.ai tap:

```bash
brew tap mozilla-ai/tap
```

Then install `mcpd`:

```bash
brew install mcpd
```

Or install directly from the cask:

```bash
brew install --cask mozilla-ai/tap/mcpd
```

Please read our docs to install via [GitHub releases](https://mozilla-ai.github.io/mcpd/installation/#via-github-releases) or [local Go Binary build](https://mozilla-ai.github.io/mcpd/installation/#via-local-go-binary-build).

### Using mcpd

```bash
# Initialize a new project and create a new .mcpd.toml file
mcpd init

# Add an MCP server to .mcpd.toml
mcpd add time

# Set the local timezone for the MCP server
mcpd config args set time -- --local-timezone=Europe/London

# Start the daemon in dev mode with debug logging
mcpd daemon --dev --log-level=DEBUG --log-path=$(pwd)/mcpd.log
```

Now that the daemon is running, let's call the `get_current_time` tool provided by the `time` MCP server

```bash
# Check the time
curl -s --request POST \
  --url http://localhost:8090/api/v1/servers/time/tools/get_current_time \
  --header 'Accept: application/json, application/problem+json' \
  --header 'Content-Type: application/json' \
  --data '{
  "timezone": "Europe/Warsaw"
}'
```

API docs will be available at [http://localhost:8090/docs](http://localhost:8090/docs).

## 💡 Why `mcpd`? 

Engineering teams build agents that work locally, then struggle to make them production-ready across environments. mcpd bridges this gap with declarative configuration and secure secrets management.

- Declarative & reproducible – .mcpd.toml defines your tool infrastructure
- Language-agnostic – Python, JS, Docker containers via unified HTTP API
- Dev-to-prod ready – Same config works locally and in containers

## 🏗️ Built for Dev & Production

| Development Workflow                                                              | Production Benefit                                         |
|-----------------------------------------------------------------------------------|------------------------------------------------------------|
| `mcpd daemon` runs everything locally                                             | Same daemon runs in containers                             |
| `.mcpd.toml` version-controlled configs                                           | Declarative infrastructure as code                         |
| Local secrets in `~/.config/mcpd/`                                                | Secure secrets injection via control plane                 |
| `mcpd config export` exports version-control safe snapshot of local configuration | Sanitized secrets config and templates for CI/CD pipelines |

## 📦 SDKs

### `mcpd` SDKs

| Language   | Repository                                                               | Status |
|------------|--------------------------------------------------------------------------|--------|
| Python     | [mcpd-sdk-python](https://github.com/mozilla-ai/mcpd-sdk-python)         | ✅      |
| JavaScript | [mcpd-sdk-javascript](https://github.com/mozilla-ai/mcpd-sdk-javascript) | ✅      |


### `mcpd` plugin SDKs

Plugin SDKs are built using the [mcpd plugin Protocol Buffers specification](https://github.com/mozilla-ai/mcpd-proto).

| Language | Repository                                                                       | Status |
|----------|----------------------------------------------------------------------------------|--------|
| Go       | [mcpd-plugins-sdk-go](https://github.com/mozilla-ai/mcpd-plugins-sdk-go)         | ✅      |
| .NET     | [mcpd-plugins-sdk-dotnet](https://github.com/mozilla-ai/mcpd-plugins-sdk-dotnet) | ✅      |
| Python   | [mcpd-plugins-sdk-python](https://github.com/mozilla-ai/mcpd-plugins-sdk-python) | ✅      |

## 💻 Development

If you are developing `mcpd`, you will need:
- [**Go**](https://go.dev/doc/install) (v1.25.6+ recommended)

Build local code:
```bash
make build
```

Run tests:
```bash
make test
```

Validate Mozilla AI registry (when modifying registry files):
```bash
make validate-registry
```

Run the local documentation site (requires `uv`), dynamically generates command line documentation:
```bash
make docs
```

## 🧩 The Mozilla.ai Stack
`mcpd` is the "Tooling Layer" of the Mozilla.ai ecosystem. These tools are designed to work together or standalone.

| Layer | Tool                                                                       | Function |
|----------|----------------------------------------------------------------------------------|--------|
| Compute       | [llamafile](https://github.com/mozilla-ai/llamafile)         |  Local LLM inference server |
|Interface      | [any-llm](https://github.com/mozilla-ai/any-llm)| Unified Python library for LLM inference |
| Logic     | [any-agent](https://github.com/mozilla-ai/any-agent) | Orchestration and agent loops |
| Tools   | [mcpd](https://github.com/mozilla-ai/mcpd) | **(You are here)** Tool sandbox & router |
| Safety  | [any-guardrail](https://github.com/mozilla-ai/any-guardrail)| Input/Output validation |

## 🤝 Contributing

Please see our [Contributing to mcpd](CONTRIBUTING.md) guide for more information. 

## 📄 License

[Licensed](LICENSE) under the [Apache License 2.0](https://www.apache.org/licenses/LICENSE-2.0).

