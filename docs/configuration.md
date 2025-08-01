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

[[servers]]
  name = "time"
  package = "uvx::mcp-server-time@0.6.2"
  tools = ["get_current_time"]
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
    
#### Example collision

```toml
[[servers]]
name = "example"
required_env = ["DATABASE_URL"]
required_args = ["--database-url"]
```
    
Both will use the template variable `MCPD__EXAMPLE__DATABASE_URL` in the generated files.
