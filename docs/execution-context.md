# Execution Context Configuration

User-specific secrets and runtime arguments are stored in:

```bash
~/.mcpd/secrets.dev.toml
```

This file is modified using the following commands:

- `mcpd config set-args`
- `mcpd config set-env`

These values apply at runtime and are separate from your project-specific `.mcpd.toml`.

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
    The Execution Context Configuration file is automatically updated by `mcpd config` commands, you shouldn't edit it by hand unless absolutely necessary.
