# `mcpd` CLI

The primary interface for developers to interact with `mcpd`, define their agent projects, and manage MCP server dependencies.

## Requirements

* [`uv`/`uvx`](https://docs.astral.sh/uv/getting-started/installation/)
* [Go](https://go.dev/doc/install) to build the binary, or carry out development
* Internet access when running `mcpd` in order to contact package repos and allow MCP servers access to the internet if required

## Global (optional) settings

### Config file path

### Precedence

The order of precedence for these options is:

```
CLI flag > environment variable > default value
```

Where CLI is preferred over env var, and env var is preferred over the default value.

### Config file path

All commands (excepting `init`) support an optional parameter which specifies a specific location for the `mcpd` config file.

#### via CLI

```console
mcpd --config-file <config-file-path>
```

#### via environment variable

```console
MCPD_CONFIG_FILE=<config-file-path>
```

#### Default location

`.mcpd.toml` in current directory that the CLI is executed.

#### Sample configuration file

```toml
[[servers]]
  name = "fetch"
  package = "pypi::mcp-server-fetch@2025.4.7"

[[servers]]
  name = "time"
  package = "pypi::mcp-server-time@0.6.2"
  tools = ["get_current_time"]

```

### Log level

Sets the logging level for `mcpd`.

#### via CLI

```console
mcpd --log-level=<log-level>
```

#### via environment variable

```console
MCPD_LOG_LEVEL=<log-level>
```

#### Default value

```console
INFO
```

### Log path

Sets the log file path for `mcpd`.

#### via CLI

```console
mcpd --log-pathl=<log-path>
```

#### via environment variable

```console
MCPD_LOG_PATH=<log-path>
```

#### Default value

By default, logging will be discarded unless the log path is configured. Some terminal output is still emitted.

## Commands

### `mcpd init`

Initializes `mcpd` in the current directory, creating the `.mcpd.toml` file.

### `mcpd add`

Attempts to add the package to the `mcpd.toml` file based on finding the right package and version on PyPi.

The format is given as `<server-name>` but will attempt to resolve `mcp-server-<server-name>` on PyPi.

Output is send to the terminal showing the tools and args that `mcpd` thinks exist for the package, config can be 
added for them using `mcpd config set-args` below.

### `mcpd remove`

Removes a MCP server from `mcpd`'s project configuration file (not from any user specific configuration).

### `mcpd config set-args`

Used to set arguments that should be supplied to the MCP server on start-up, 
these values are stored in the 'execution context configuration' (see below).

Example:

```console
mcpd config set-args fetch --arg --ignore-robots-txt --arg --user-agent=mcpd/1.0.0
```

### `mcpd config set-env`

Used to set environment variables that should be supplied to the MCP server on start-up,
these values are stored in the 'execution context configuration' (see below).

Example:

```console
mcpd config set-env fetch --env foo=bar --env baz=123
```

## Execution context configuration 

User specific secrets/config is stored separately from the project specific package configuration for `mcpd`.

The current location for all stored config for an MCP server's execution context: 

```console
~/.mcpd/secrets.dev.toml
```

The file is modified by using the `mcpd config set-args` and `mcpd config set-env` commands.

### Sample file configuration

```toml
[servers]
  [servers.fetch]
    args = ["--ignore-robots-txt", "--user-agent=mcpd/1.0.0"]
    [servers.fetch.env]
      foo = "bar"
  [servers.time]
    args = ["--local-timezone=Europe/London"]
    [servers.time.env]
      baz = "123"
      qwerty = "xyz"

```

## Makefile

With `make` installed you can use the following commands:

* `make build` - builds the project using Go
* `make install` - installs the binary to `/usr/local/bin` (usually requires `sudo`)
* `make uninstall` - removes the binary from `/usr/local/bin` (usually  requires `sudo`)
* `make clean` - removes the built binary from the working directory
* `make test` - runs all Go tests

## Basic tutorial

This tutorial uses `mcpd` and some command line tools (`cat`, `curl`, `jq`, `make`) to demonstrate adding/configuring, 
starting and calling MCP server.

### Verifying file contents

In this tutorial we use the default path for the config-file, so file contents can be verified at any point after running 
the relevant command.

### Building

```bash
go version          # Should output 1.24.3 (or higher): go version go1.24.3 darwin/arm64
make build          # Builds the binary
sudo make install   # Installs into /usr/local/bin on the PATH
```

#### `.mcpd.toml` configuration file

Holds the name, package, version and allowed tools for all configured MCP servers in the project. 

```bash
cat .mcpd.toml
```

#### MCP server execution context configuration

Holds the specific contextual configuration for configured MCP servers (think secrets, or user specific config)

```bash
cat ~/.mcpd/secrets.dev.toml
```

### Building

```bash
go version          # Should output 1.24.3 (or higher): go version go1.24.3 darwin/arm64
make build          # Builds the binary
sudo make install   # Installs into /usr/local/bin on the PATH
```

### Initialize `mcpd`

Initialize `mcpd` in your existing project directory:

```bash
mcpd init
```

Console output:

```console
Initializing mcpd project in current directory...
.mcpd.toml created successfully.
```

### Adding an MCP server

Add the latest version fo the `time` MCP server:  

```bash
mcpd add time
```

Console output:

```console
✓ Added server 'time' (version: 0.6.2), exposing only tools: convert_time, get_current_time

📦 PyPI package information...
  ⚙️ Found startup args: --local-timezone
  🔨 Found tools: convert_time, get_current_time
```

`.mcpd.toml` file contents:

```toml
[[servers]]
    name = "time"
    package = "pypi::mcp-server-time@0.6.2"
    tools = ["convert_time", "get_current_time"]
```

When `mcpd add` is used without the `--version` flag, the latest version is pinned (`latest` at the time of writing is `0.6.2`).

When `mcpd add` is used without any `--tool` flags, all the tools that `mcpd` can parse are added to the `tools` allow list in config for the MCP server.

Perhaps we decide that we only want to allow `get_current_time`, 
we can remove and re-add with the correct tools (for the sake of examples, let's use the `--version` flag too):

```bash
mcpd add time --version 0.6.2 --tool get_current_time
```

Console output:

```console
Added server 'time' (version: 0.6.2), exposing only tool: get_current_time

📦 PyPI package information...
  ⚙️ Found startup args: --local-timezone
  🔨 Found tools: convert_time, get_current_time
```

`.mcpd.toml` file contents:

```toml
[[servers]]
    name = "time"
    package = "pypi::mcp-server-time@0.6.2"
    tools = ["get_current_time"]
```

### Configuring an MCP server

The `mcpd add` command output showed the following:

```console
⚙️ Found startup args: --local-timezone
```

Which tells the user that we may need to configure `--local-timezone` for this MCP server (mileage may vary with the default).

To do this, use the `mcpd config set-args` command:

```bash
mcpd config set-args time --arg --local-timezone=Europe/London
```

Console output:

```console
✓ Startup arguments set for server 'time': [--local-timezone=Europe/London]
```

`~/.mcpd/secrets.dev.toml` file contents:

```toml
[servers]
  [servers.time]
    args = ["--local-timezone=Europe/London"]
```

### Start the `mcpd` daemon

To start our MCP servers and expose an API endpoint to communicate with them from our agentic applications use:

```bash
mcpd daemon
```

Console output:

```console
Attempting to start 1 MCP server(s)
Starting MCP server: 'time'...
MCP server started
HTTP REST API listening on http://localhost:8090/api/v1/servers
Press CTRL+C to shut down.
```

`CTRL+C` will stop the daemon.

### Querying API endpoints 

#### All running MCP servers

```bash
curl -s http://localhost:8090/api/v1/servers | jq
```

#### All tools allowed on a specific MCP server

```bash
curl -s  http://localhost:8090/api/v1/servers | jq
```

#### Calling a tool for an MCP server

```bash
curl -X POST -H "Content-Type: application/json" \
    -d '{"timezone": "America/New_York"}' \
    http://localhost:8090/api/v1/servers/time/get_current_time | jq -r '.[0]' | jq
```

## Notes

### Package resolution

Currently only PyPi is supported, and also expects the format `mcp-server-<server-name>` as the package.

The warnings for args and tools from `mcpd add` are 'best effort', and likely quite brittle as `mcpd` is parsing READMEs from PyPi.

