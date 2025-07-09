# Execution Context (Runtime) Configuration

## Global Configuration

!!! info "Precedence"
    The order of precedence for these options is:  
    `CLI flag > environment variable > default value`

## Runtime File Path

All commands support an optional parameter to specify the location of the `mcpd` runtime file which 
provides the execution context.

You can provide this path in multiple ways:

- CLI flag: `--runtime-file <path>`
- Environment variable: `MCPD_RUNTIME_FILE=<path>`
- Default: `~/.config/mcpd/secrets.dev.toml`

!!! note "XDG_CONFIG_HOME environment variable"
    mcpd honors the [XDG Base Directory Specification](https://specifications.freedesktop.org/basedir-spec/latest/), 
    respecting the `XDG_CONFIG_HOME` environment variable. This forms the base directory where `mcpd` will create an 
    application folder.

---

The runtime file is modified using the following commands:

- `mcpd config args set`
- `mcpd config env set`

These values apply at runtime and are separate from your **project-specific** `.mcpd.toml`.

---

## Sample Configuration File
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

!!! warning "Manual Changes"
    The Execution Context Configuration file is automatically updated by `mcpd config` commands, 
    you shouldn't edit it by hand unless absolutely necessary.
