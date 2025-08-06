# Basic Tutorial

This tutorial walks you through using `mcpd` from setup to making requests to a running MCP server.

---

## 1. Install `mcpd` via Homebrew
```bash
brew tap mozilla-ai/tap
brew install mcpd
```

Official releases can be downloaded from [mcpd's GitHub releases page](https://github.com/mozilla-ai/mcpd/releases).

---

## 2. Initialize the Project
```bash
mcpd init
```

!!! note "Config File Location"
    This creates an `.mcpd.toml` file in your current directory.

---

## 3. Add an MCP Server

Add the latest version of the `time` server:
```bash
mcpd add time
```

Or add a specific version and restrict access to specific tools:
```bash
mcpd add time --version 0.6.2 --tool get_current_time
```

---

## 4. Set Startup Arguments

Configure any startup flags required by the server:
```bash
mcpd config args set time -- --local-timezone=Europe/London
```

---

## 5. Start the Daemon

Start `mcpd`, which launches MCP servers and exposes the HTTP API:
```bash
mcpd daemon
```

!!! note "API Endpoint"
    The API docs will be available at `http://localhost:8090/docs`

---

## 6. Query Running Servers

List all running servers:
```bash
curl -s http://localhost:8090/api/v1/servers | jq
```

---

## 7. Call a Tool on a Server

Make a request to a tool on a specific MCP server:
```bash
curl -s -X POST -H "Content-Type: application/json" \
     -d '{"timezone": "America/New_York"}' \
     http://localhost:8090/api/v1/servers/time/tools/get_current_time | jq
```


# Using mcpd in your Python application

For tutorials on using mcpd with agents in Python, please refer to the [Python mcpd SDK](https://github.com/mozilla-ai/mcpd-sdk-python) documentation.