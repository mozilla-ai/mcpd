# Global Configuration

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
