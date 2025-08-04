# <img src="docs/assets/mcpd-logo.png" alt="mcpd" width="300"/>

> **Run your agents, not your infrastructure.**

**mcpd** is a tool to declaratively manage [Model Context Protocol](https://modelcontextprotocol.io/overview) (MCP) servers, providing a consistent interface to define and run tools across environments, from local development to containerized cloud deployments.

Built by [Mozilla AI](https://mozilla.ai)

---

Today, `mcpd` launches MCP servers as subprocesses using STDIO (Standard Input/Output) and acts as an HTTP proxy between agents and the tools they expose. This enables agent-compatible workflows with support for secrets, runtime arguments, and reproducible configurations, no matter where `mcpd` is running.

We're developing a Kubernetes operator, guided by our internal roadmap, to extend `mcpd` for deploying and managing MCP servers as long-lived services in production. It will use the same `.mcpd.toml` configuration and proxy model, making it easier to scale and manage lifecycles without changing the developer experience.


## The Problem

ML teams build agents that work perfectly locally. Operations teams get handed Python scripts and told "make this production-ready across dev/UAT/prod." 
The gap between local development and enterprise deployment kills AI initiatives.

`mcpd` solves this with declarative configuration, secure secrets management, and seamless environment promotion - all while keeping the developer experience simple.


## Why mcpd?

**Zero-Config Tool Setup**  
No cloning repos or installing language-specific dependencies. `mcpd add` and `mcpd daemon` handle everything.

**Language-Agnostic Tooling**  
Use MCP servers written in Python, JavaScript, TypeScript via a unified HTTP API.

**Declarative Configuration**  
Version-controlled `.mcpd.toml` files define your agent infrastructure. Reproducible, auditable, CI-friendly.

**Enterprise-Ready Secrets**  
Separate project configuration from runtime variables, and export sanitized secrets templates. Never commit secrets to Git again.

**Seamless Local-to-Prod**  
Same configuration works in development, CI, and cloud environments without modification.


## Built for Dev & Production

| Development Workflow                                                              | Production Benefit                                         |
|-----------------------------------------------------------------------------------|------------------------------------------------------------|
| `mcpd daemon` runs everything locally                                             | Same daemon runs in containers                             |
| `.mcpd.toml` version-controlled configs                                           | Declarative infrastructure as code                         |
| Local secrets in `~/.config/mcpd/`                                                | Secure secrets injection via control plane                 |
| `mcpd config export` exports version-control safe snapshot of local configuration | Sanitized secrets config and templates for CI/CD pipelines |

## Features

- Focus on Developer Experience via `mcpd` CLI
- Declarative configuration (`.mcpd.toml`) to define required servers/tools
- Run and manage language-agnostic MCP servers
- Secure execution context for secrets and runtime args
- Smooth dev-to-prod transition via the `mcpd` daemon
- Rich CLI and SDK tooling, see supported languages below:


## 🚀 Quick Start

### Prerequisites

- [Go](https://go.dev/doc/install) (only required for development of `mcpd`)
- [npx](https://docs.npmjs.com/downloading-and-installing-node-js-and-npm)
- [uvx](https://docs.astral.sh/uv/getting-started/installation/)

### Installation

#### via Homebrew

Add the Mozilla.ai tap:

```bash
brew tap mozilla-ai/tap
```

Then install `mcpd`:

```bash
brew install mcpd
````

Or install directly from the cask:

```bash
brew install --cask mozilla-ai/tap/mcpd
```

#### via GitHub releases

Official releases can be found on [mcpd's GitHub releases page](https://github.com/mozilla-ai/mcpd/releases).

The following is an example of manually downloading and installing `mcpd` using `curl` and `jq` by running `install_mcpd`:

```bash
function install_mcpd() {
    command -v curl >/dev/null || { echo "curl not found"; return 1; }
    command -v jq >/dev/null || { echo "jq not found"; return 1; }

    latest_version=$(curl -s https://api.github.com/repos/mozilla-ai/mcpd/releases/latest | jq -r .tag_name)
    os=$(uname)
    arch=$(uname -m)

    zip_name="mcpd_${os}_${arch}.tar.gz"
    url="https://github.com/mozilla-ai/mcpd/releases/download/${latest_version}/${zip_name}"

    echo "Downloading: $url"
    curl -sSL "$url" -o "$zip_name" || { echo "Download failed"; return 1; }

    echo "Extracting: $zip_name"
    tar -xzf "$zip_name" mcpd || { echo "Extraction failed"; return 1; }

    echo "Installing to /usr/local/bin"
    sudo mv mcpd /usr/local/bin/mcpd && sudo chmod +x /usr/local/bin/mcpd || { echo "Install failed"; return 1; }

    rm -f "$zip_name"
    echo "mcpd installed successfully"
}
```

#### via local Go binary build

```bash
# Clone and build
git clone git@github.com:mozilla-ai/mcpd.git
cd mcpd
make build
sudo make install # Install mcpd 'globally' to /usr/local/bin
```

### Using mcpd

```bash
# Initialize a new project
mcpd init

# Add an MCP server
mcpd add time

# Set the local timezone for the MCP server
mcpd config args set time -- --local-timezone=Europe/London

# Start the daemon in dev mode with debug logging
mcpd daemon --dev --log-level=DEBUG --log-path=$(pwd)/mcpd.log

# You can tail the log file
tail -f mcpd.log
```

API docs will be available at [http://localhost:8090/docs](http://localhost:8090/docs).


## Deploy Anywhere

`mcpd` is runtime-flexible and infrastructure-agnostic:

- ⚙️ Works in any container or host with `uv` and `npx`
- ☁️ Multi-cloud ready (AWS, GCP, Azure, on-prem)
- ♻️ Low resource overhead via efficient server management


## 📚 Documentation & SDKs

**Full documentation:** [https://mozilla-ai.github.io/mcpd/](https://mozilla-ai.github.io/mcpd/)

**SDKs available:**

| Language   | Repository                                                       | Status |
|------------|------------------------------------------------------------------|--------|
| Python     | [mcpd-sdk-python](https://github.com/mozilla-ai/mcpd-sdk-python) | ✅      |
| JavaScript | _Coming soon_                                                    | 🟡     |


## 💻 Development

Build local code:
```bash
make build
```

Run tests:
```bash
make test
```

Run the local documentation site (requires `uv`), dynamically generates command line documentation:
```bash
make docs
```


---

## 🤝 Contributing

Please see our [Contributing to mcpd](CONTRIBUTING.md) guide for more information. 

## 📄 License

[Licensed](LICENSE) under the [Apache License 2.0](https://www.apache.org/licenses/LICENSE-2.0).

