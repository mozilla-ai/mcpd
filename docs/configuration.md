# Project configuration

## Global Configuration

!!! info "Precedence"
    The order of precedence for these options is:  
    `CLI flag > environment variable > default value`

## Config File Path

All commands support an optional parameter to specify the location of the `mcpd` config file.

You can provide this path in multiple ways:

- CLI flag: `--config-file <path>`
- Environment variable: `MCPD_CONFIG_FILE=<path>`
- Default: `.mcpd.toml` in the current working directory

---

## Sample Configuration File

```toml
[[servers]]
  name = "fetch"
  package = "uvx::mcp-server-fetch@2025.4.7"
  tools = ["fetch"]

[[servers]]
  name = "time"
  package = "uvx::mcp-server-time@2025.8.4"
  tools = ["get_current_time", "convert_time"]
```

---

## Log Level

Sets the logging level for `mcpd`.

You can configure it using:

- CLI flag: `--log-level=<level>`
- Environment variable: `MCPD_LOG_LEVEL=<level>`

Default:

```
INFO
```

---

## Log Path

Sets the log file path for `mcpd`.

Options:

- CLI flag: `--log-path=<path>`
- Environment variable: `MCPD_LOG_PATH=<path>`

!!! warning "Setting Log Path"
    Log entries will be discarded by default, unless a log path is configured. 
    Output intended for the terminal is still emitted.

---

## Hot Reload

The `mcpd` daemon supports hot-reloading of MCP server configurations without requiring a full restart. This allows you to add, remove, or modify server configurations while keeping the daemon running.

Hot reload processes both:

- **Server configuration** (`--config-file`) e.g. `.mcpd.toml`
- **Execution context** (`--runtime-file`) e.g. `secrets.dev.toml`

### SIGHUP Signal

Send a `SIGHUP` signal to the running daemon process to trigger a configuration reload:

```bash
# Find the daemon process ID
ps aux | grep mcpd

# Send reload signal (replace PID with actual process ID)
kill -HUP <PID>
```

### Reload Behavior

During a hot reload, the daemon intelligently categorizes changes and responds accordingly:

| Change Type           | Action   | Description                                                                                                                                                       |
|-----------------------|----------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| Unchanged servers     | Preserve | Servers with identical configurations keep their existing connections, tools, and health status                                                                   |
| Removed servers       | Stop     | Servers no longer in the config file are gracefully shut down                                                                                                     |
| New servers           | Start    | Newly added servers are initialized and connected                                                                                                                 |
| 'Tools-Only' changes  | Update   | When only the `tools` change, the daemon updates the allowed tools without restarting the server process                                                          |
| Configuration changes | Restart  | Servers with other configuration changes (package version, environment variables, arguments, execution context, etc.) are stopped and restarted with new settings |

### Example: 'Tools-Only' Update

Consider this server configuration:

```toml
[[servers]]
  name = "github"
  package = "uvx::modelcontextprotocol/github-server@1.2.3"
  tools = ["create_repository", "get_repository"]
```

If you modify only the tools list:

```toml
[[servers]]
  name = "github" 
  package = "uvx::modelcontextprotocol/github-server@1.2.3"
  tools = ["create_repository", "get_repository", "list_repositories"] # Additional tools
```

The daemon will:

1. Detect that only the `tools` array changed
2. Update the allowed tools list in-place
3. Keep the existing server process and connections intact
4. Log a message that tools for a server were updated (including the server name and list of tools)

### Example: Package Version Update

If you change the package version:

```toml
[[servers]]
  name = "github"
  package = "uvx::modelcontextprotocol/github-server@1.3.0"  # Version changed
  tools = ["create_repository", "get_repository", "list_repositories"]
```

The daemon will:

1. Detect configuration changes beyond just tools
2. Gracefully stop the existing server
3. Start a new server with the updated configuration
4. Log a message that the server is being restarted (including the server name)

### Execution Context and Environment Variables

!!! warning "Environment Variable Visibility"
    The `mcpd` process can only see environment variables that existed when it started.

    If you export new environment variables in your shell after starting `mcpd`, you must restart the daemon for those variables to become available for shell expansion.

When the execution context file is reloaded, shell expansion of environment variables (`${VAR}` syntax) 
occurs using the environment available to the running `mcpd` process when it was started.

#### What Works During Hot Reload

Direct values are applied immediately:

```toml
[servers.jira]
  args = ["--confluence-token=test123", "--confluence-url=http://jira-test.mozilla.ai"]
[servers.mcp-discord.env]
  DISCORD_TOKEN = "qwerty123!1one"
```

Shell expansion of existing environment variables works:

```toml
[servers.myserver]
  args = ["--home=${HOME}", "--user=${USER}"]  # These expand to current values
[servers.myserver.env]
  CONFIG_PATH = "${HOME}/.config/myapp"  # Expands using mcpd's environment
```

#### What Requires an `mcpd` Restart

New environment variables added to the system after `mcpd` started won't be visible:

```toml
[servers.myserver]
  args = ["--token=${NEW_TOKEN}"]  # NEW_TOKEN added after mcpd started
[servers.myserver.env]
  API_KEY = "${NEWLY_EXPORTED_VAR}"  # Won't expand until mcpd restarts
```

### Limitations


Hot reload does **NOT** apply to:

- Daemon-level config settings (timeouts, CORS, etc.)
- New environment variables added to the system

Both require `mcpd` to be restarted for changes to take effect

### Error Handling

The reload process maintains strict consistency - any error causes the daemon to exit:

- **Configuration errors**: Invalid configuration files or loading failures cause the daemon to exit
- **Validation errors**: Invalid server configurations cause the daemon to exit  
- **No tools configured**: If a server configuration has no tools (empty tools list or manually removed from config), the daemon will exit with an error
- **Server operation failures**: Any failure to start, stop, or restart a server causes the daemon to exit

This ensures the daemon never runs in an inconsistent or partially-failed state, matching the behavior during initial startup where any server failure prevents the daemon from running.

!!! warning "Reload Failures"
    Unlike some systems that allow partial reloads, `mcpd` exits on any reload error to prevent inconsistent state. You'll need to fix the configuration and restart the daemon.

---

## Configuration Export

The `mcpd config export` command generates portable configuration files for deployment across different environments. It creates template variables using the naming pattern `MCPD__{SERVER_NAME}__{VARIABLE_NAME}`.

### Template Variable Generation

Environment variables and command-line arguments are both converted to template variables using the same naming scheme:

- Environment variable`DATABASE_URL` becomes `MCPD__{SERVER_NAME}__DATABASE_URL`  
- Command-line argument `--database-url` becomes `MCPD__{SERVER_NAME}__DATABASE_URL`

### Variable Name Collisions

!!! danger "Naming Collisions"
    If a server has both an environment variable and a command-line argument that normalize to the same name (e.g., `DATABASE_URL` and `--database-url`), they will generate the same template variable name.
    
In most cases, this is intentional, the same configuration value is being used in different ways. The collision results in a single template variable that can be used for both the environment variable and command-line argument.
    
#### Example Collision

```toml
[[servers]]
name = "example"
required_env = ["DATABASE_URL"]
required_args = ["--database-url"]
```
    
Both will use the template variable `MCPD__EXAMPLE__DATABASE_URL` in the generated files.
