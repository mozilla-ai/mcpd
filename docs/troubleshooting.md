# Troubleshooting

This page collects the most common setup, configuration, and runtime problems when working with `mcpd`.

## Start Here

Before changing configuration, collect a little context:

```bash
mcpd daemon --dev --log-level=DEBUG --log-path=$(pwd)/mcpd.log
```

Then:

- Reproduce the problem and inspect `mcpd.log`.
- Confirm the API is reachable at `http://localhost:8090/docs` or your configured `api.addr`.
- List the servers the daemon is currently exposing:

```bash
curl -s http://localhost:8090/api/v1/servers
```

!!! tip "Logs are not persisted unless you set a log path"
    `mcpd` discards log entries by default unless `--log-path` or `MCPD_LOG_PATH` is set.

## `mcpd add` or `mcpd search` Cannot Find a Server

If you see errors like `server '...' not found in any registry` or `required source registry not found: ...`:

- Check the server name and version you passed.
- If you used `--runtime`, retry without it if you are not sure which installation method the registry recommends.
- If you used `--source`, make sure the registry ID exists and is spelled correctly.
- If results look stale, retry with `--refresh-cache` or bypass cache once with `--no-cache`.
- Confirm internet access. Remote registry lookups and first-time package resolution need network access.

See also: [Registry Caching](caching.md), [Requirements](requirements.md).

## A Server Was Added, but the Daemon Cannot Start It

`mcpd add` writes configuration, but the daemon still needs the matching runtime installed on the machine where it runs.

Check the package prefix in `.mcpd.toml`:

- `uvx::...` requires `uv`
- `npx::...` requires Node.js and `npx`
- `docker::...` requires Docker

The `mcpd inspector` command always uses `npx`, even if your configured servers do not.

See also: [Requirements](requirements.md), [Installation](installation.md).

## Runtime Values Are Missing or Not What You Expected

Remember the split between project config and runtime config:

- `.mcpd.toml` stores server definitions, versions, and allowed tools.
- The runtime file stores machine-specific args, environment variables, and volumes.
- The default runtime file is `~/.config/mcpd/secrets.dev.toml` unless you override it with `--runtime-file` or `MCPD_RUNTIME_FILE`.

Useful checks:

```bash
mcpd config args list time
mcpd config env list time
```

If the server exists in `.mcpd.toml` but not in the runtime file yet, `args list` or `env list` may report that the server is not found in the runtime configuration. In that case, set the values first:

```bash
mcpd config args set time -- --local-timezone=Europe/London
mcpd config env set time FOO=bar
```

`mcpd config args set` replaces the current argument list by default. Use it carefully if the server already has runtime arguments configured.

See also: [Execution Context](execution-context.md).

## You Changed Config, but Nothing Happened

Not every change is picked up the same way.

- Changes to server configuration in `.mcpd.toml` can be hot-reloaded.
- Changes to the runtime file can be hot-reloaded.
- Changes to daemon-level settings such as `api.addr`, CORS, or timeouts require a daemon restart.
- Newly exported shell environment variables also require a restart. A running `mcpd` process cannot see environment variables that were added after it started.

To trigger a reload:

```bash
kill -HUP <PID>
```

Use `ps aux | grep mcpd` if you need to find the daemon PID.

If reload fails, `mcpd` exits on purpose to avoid inconsistent state. Fix the configuration, then restart the daemon.

See also: [Configuration](configuration.md), [Daemon Configuration](daemon-configuration.md).

## `configuration reload failed`

If a reload fails and the daemon exits:

- Check both `.mcpd.toml` and the runtime file for syntax or validation mistakes.
- Make sure every configured server still has at least one allowed tool.
- Review `mcpd.log` for the first error, not just the final shutdown message.
- Restart the daemon after fixing the problem.

!!! warning "Reload failures are fatal by design"
    `mcpd` exits on reload errors to avoid running with partially applied configuration.

## Browser Requests Fail Because of CORS

If browser requests to the API fail but `curl` works:

- Check whether CORS is enabled at all.
- Make sure you configured at least one allowed origin.
- If you use CLI flags, `--cors-allow-method`, `--cors-allow-header`, `--cors-expose-header`, `--cors-allow-credentials`, and `--cors-max-age` all require `--cors-enable`.
- Validate the daemon config:

```bash
mcpd config daemon validate
mcpd config daemon list
```

- Restart the daemon after changing daemon-level config.

See also: [Daemon Configuration](daemon-configuration.md).

## The API Address or Port Is Wrong

If the API is not where you expect it to be, or another process is already using the port:

- Check the current address:

```bash
mcpd config daemon get api.addr
```

- Set a new one if needed:

```bash
mcpd config daemon set api.addr="localhost:8080"
```

- Restart the daemon after changing daemon config.
- If you started the daemon with CLI flags, remember that flags override values from `.mcpd.toml`.

## The Wrong Tools Are Exposed for a Server

`mcpd` can intentionally expose only a subset of a server's tools.

Useful checks:

```bash
mcpd config tools list time
mcpd config tools list time --all
```

Use the first command to inspect the currently allowed tools from `.mcpd.toml`.
Use the second command to compare them with all tools available from the registry.

If the allowed list is wrong, update it:

```bash
mcpd config tools set time --tool get_current_time
mcpd config tools remove time convert_time
```

If a server ends up with no tools configured, daemon startup or reload will fail.

## Docker-Based Servers Do Not Work When `mcpd` Runs in Docker

If `mcpd` itself is running in a container and one of your servers uses the Docker runtime, mount the host Docker socket:

```bash
docker run -p 8090:8090 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v $PWD/.mcpd.toml:/etc/mcpd/.mcpd.toml \
  -v $HOME/.config/mcpd/secrets.dev.toml:/home/mcpd/.config/mcpd/secrets.prod.toml \
  mzdotai/mcpd:<version>
```

Without the socket mount, containerized `mcpd` cannot control Docker-based MCP servers on the host.

!!! warning "Security note"
    Mounting the Docker socket gives the container broad access to the host Docker daemon. Use only with trusted images.

See also: [Installation](installation.md).

## Plugin Configuration Is Failing

If plugin setup is failing, validate it directly:

```bash
mcpd config plugins validate
```

If you want to verify that plugin binaries exist on the current machine as well:

```bash
mcpd config plugins validate --check-binaries
```

If a required plugin fails at runtime, `mcpd` returns HTTP `500` and sets the `Mcpd-Error-Type` header to indicate whether the failure happened in the request or response pipeline.

See also: [Plugin Configuration](plugin-configuration.md).

## Still Stuck?

When asking for help, include:

- Your OS and how you installed `mcpd`
- The exact command you ran
- The relevant server entry from `.mcpd.toml`
- The relevant runtime prefix: `uvx::`, `npx::`, or `docker::`
- The first useful lines from `mcpd.log`

Do not include secrets from your runtime file.
