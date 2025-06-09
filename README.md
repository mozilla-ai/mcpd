# `mcpd` CLI

The primary interface for developers to interact with `mcpd`, define their agent projects, and manage MCP server dependencies.

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
  package = "pypi::mcp-server-fetch@latest"

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

Output it send to the terminal showing the tools and args that `mcpd` thinks exist for the package, config can be 
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

## Notes

### Package resolution

Currently only PyPi is suported, and also expects the format `mcp-server-<server-name>` as the package.

The warnings for args and tools from `mcpd add` are 'best effort', and likely quite brittle as `mcpd` is parsing READMEs from PyPi.

