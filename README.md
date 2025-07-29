# mcpd

**mcpd** is a tool to declaratively manage [Model Context Protocol](https://modelcontextprotocol.io/overview) (MCP) servers, providing a consistent interface to define and run tools across environments, from local development to containerized cloud deployments.

Today, mcpd launches MCP servers as subprocesses using stdio and acts as an HTTP proxy between agents and the tools they expose. This enables agent-compatible workflows with support for secrets, runtime arguments, and reproducible configurations ,no matter where mcpd is running.

We're developing a Kubernetes operator, guided by our internal roadmap, to extend mcpd for deploying and managing MCP servers as long-lived services in production. It will use the same .mcpd.toml configuration and proxy model, making it easier to scale and manage lifecycles without changing the developer experience.

> Built by [Mozilla AI](https://mozilla.ai)

## 🚀 Features

- Declarative `.mcpd.toml` to define servers/tools
- Run and manage language-agnostic MCP servers via a single CLI
- Secure execution context for secrets and runtime args
- Smooth dev-to-prod transition via the `mcpd` daemon
- Rich CLI and SDK tooling, see supported languages below:


| Language   | Repository                                                       | Status             |
|------------|------------------------------------------------------------------|--------------------|
| Python     | [mcpd-sdk-python](https://github.com/mozilla-ai/mcpd-sdk-python) | :white_check_mark: |
| JavaScript | _Coming soon_                                                    | :yellow_circle:    |

## 📖 Documentation

Full documentation available at:

👉 **[https://mozilla-ai.github.io/mcpd/](https://mozilla-ai.github.io/mcpd/)**

Covers setup, CLI usage, configuration, secrets, the daemon, Makefile commands, and full tutorials.

Explore the Python SDK, with a list of examples using it with different agent frameworks, at:

👉 **[https://github.com/mozilla-ai/mcpd-sdk-python](https://github.com/mozilla-ai/mcpd-sdk-python)**

## ⚙️ Quickstart

Install dependencies:

- [Go](https://go.dev/doc/install)
- [npx](https://docs.npmjs.com/downloading-and-installing-node-js-and-npm)
- [uvx](https://docs.astral.sh/uv/getting-started/installation/)

Clone the repo:
```bash
git clone git@github.com:mozilla-ai/mcpd.git
cd mcpd
```

Build the binary:
```bash
make build
```

Install globally:
```bash
sudo make install
```

In your agentic application code...

Initialize a new project:
```bash
mcpd init
```

Add a server (e.g. `time`):
```bash
mcpd add time
```

Start the daemon:
```bash
mcpd daemon
```

API docs will be available at `/docs`, e.g. `http://localhost:8090/docs` 

## 🧰 Development

Run tests:
```bash
make test
```

Run the local documentation site (requires `uv`):
```bash
make docs-local
```

Generate CLI documentation and related docs site navigation:
```bash
make docs-cli
make docs-nav
```

---

## 🤝 Contributing

Please see our [Contributing to mcpd](CONTRIBUTING.md) guide for more information. 

## 📄 License

[Licensed](LICENSE) under the [Apache License 2.0](https://www.apache.org/licenses/LICENSE-2.0).

