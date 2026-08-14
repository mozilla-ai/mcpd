# Writing a Plugin

[Plugin Configuration](plugin-configuration.md) covers how to declare a plugin in `.mcpd.toml`.
This page covers how to build one.

A plugin is a standalone executable that `mcpd` launches as a child process and communicates with
over gRPC on a Unix socket. It is not a shared library and it is not loaded into the daemon, so it
can be written in any language with gRPC support.

---

## Protocol and SDKs

The canonical protocol definition is [`mcpd-proto`](https://github.com/mozilla-ai/mcpd-proto)
(`plugins/v1/plugin.proto`, package `mozilla.mcpd.plugins.v1`). Everything below is generated from
or implements that service.

Four SDKs wrap the generated stubs and the startup handshake:

| Language | Repository                                                                          | Install                                                       |
|----------|-------------------------------------------------------------------------------------|---------------------------------------------------------------|
| Python   | [mcpd-plugins-sdk-python](https://github.com/mozilla-ai/mcpd-plugins-sdk-python)     | `pip install mcpd-plugins`                                     |
| Go       | [mcpd-plugins-sdk-go](https://github.com/mozilla-ai/mcpd-plugins-sdk-go)             | `go get github.com/mozilla-ai/mcpd-plugins-sdk-go`             |
| Rust     | [mcpd-plugins-sdk-rust](https://github.com/mozilla-ai/mcpd-plugins-sdk-rust)         | `cargo add mcpd-plugins-sdk`                                   |
| .NET     | [mcpd-plugins-sdk-dotnet](https://github.com/mozilla-ai/mcpd-plugins-sdk-dotnet)     | `dotnet add package MozillaAI.Mcpd.Plugins.Sdk`                |

The Python, Rust, and .NET repositories each include an `examples/` directory with runnable
plugins.

For background, see
[mcpd plugins: extend your agent infrastructure without touching your code](https://blog.mozilla.ai/mcpd-plugins-extend-your-agent-infrastructure-without-touching-your-code/).

---

## Startup Handshake

`mcpd` owns the process lifecycle. It executes the plugin binary with two flags:

```bash
/path/to/plugins/jwt-auth --address /tmp/plugin-jwt-auth-1.sock --network unix
```

The plugin is responsible for creating and listening on that socket. `mcpd` polls the address
until it accepts a connection, then dials it and, in order, calls:

| Step | RPC           | Purpose                                                  |
|------|---------------|----------------------------------------------------------|
| 1    | `Configure`   | Deliver plugin configuration                             |
| 2    | `CheckReady`  | Confirm the plugin can serve requests                    |
| 3    | `GetMetadata` | Read the plugin's reported name, version and commit hash |

`GetCapabilities` is not part of startup: `mcpd` calls it lazily the first time a configured flow
needs checking, then caches the result. A plugin must still implement it.

If the socket does not accept a connection within the start timeout (10 seconds by default), the
process is killed and the plugin fails to load. Listen on the socket before doing any slow
initialisation work.

{% hint style="info" %}
**This is not hashicorp/go-plugin**

Although the transport is the same idea, `mcpd` implements its own startup sequence. There is
no magic cookie, no handshake line written to stdout, and no protocol version negotiation via
environment variables. The plugin's only contract is: bind the socket it was given, then serve
the `Plugin` service.
{% endhint %}

Plugin `stdout` and `stderr` are captured and forwarded into the daemon's logs, with levels
inferred from the output, so a plugin should log to stderr rather than trying to manage its own
log files.

---

## Version Pinning with `commit_hash`

`GetMetadata` returns a `commit_hash` that the plugin reports about itself. If the corresponding
entry in `.mcpd.toml` sets `commit_hash`, the two must match:

```toml
[[plugins.authentication]]
  name = "jwt-auth"
  commit_hash = "a1b2c3d4"
  flows = ["request"]
```

A mismatch fails the plugin with:

```console
commit hash mismatch: expected "a1b2c3d4", got "e5f6g7h8"
```

If `commit_hash` is omitted from the configuration, whatever the plugin reports is accepted. The
check exists so a deployment can pin the exact plugin build it expects, so it is only as strong as
the plugin's honesty about its own build — treat it as a deployment guard, not a security control.

---

## Handling Requests

Two RPCs do the actual work. A plugin reports the flows it supports through `GetCapabilities`; the
`flows` setting in `.mcpd.toml` selects which of those it is actually run for:

- `HandleRequest` runs during the request phase, before the call reaches the MCP server.
- `HandleResponse` runs during the response phase, on every upstream response regardless of status.

Both return a response carrying a `Continue` flag. From `HandleRequest`, `Continue=false` rejects
the request before it reaches the MCP server and returns the plugin's response to the client. From
`HandleResponse` the upstream call has already happened, so `Continue=false` stops the remaining
response plugins and returns that response rather than rejecting the original request. See
[Required Plugins](plugin-configuration.md#required-plugins) for how rejections and failures differ
depending on whether the plugin is marked `required`.

### Category Constraints

The category a plugin is configured under is not just ordering metadata; it changes what the
plugin is permitted to do.

| Category                        | Constraint                                                                  |
|---------------------------------|------------------------------------------------------------------------------|
| `content`                       | The only category that may **mutate** a request or response                  |
| `observability`                 | Runs in **parallel** with other observability plugins; may not mutate        |
| all others                      | May observe or reject, but not mutate                                        |

A plugin in a non-`content` category that returns modified content will not have those
modifications applied. If your plugin needs to rewrite payloads, it belongs in `content`.

Because observability plugins run concurrently, an optional observability plugin's rejection is
ignored rather than stopping the pipeline. See
[Observability Plugin Execution](plugin-configuration.md#observability-plugin-execution).

---

## Health and Shutdown

`CheckHealth` and `CheckReady` are separate calls, in the Kubernetes sense: readiness gates
whether the plugin is brought into service at startup, health reports ongoing liveness.

On shutdown `mcpd` calls `Stop`, closes the gRPC connection, and waits for the process to exit.
A plugin that does not exit within the force-kill timeout (2 seconds) is killed, and the daemon
allows 5 seconds for the graceful `Stop` call itself. Long-running flush or export work in a
plugin's shutdown path should be bounded accordingly, or it will be cut short.

Request and response RPCs are each subject to a per-call timeout (5 seconds by default), so a
plugin that blocks on a slow external dependency will fail the call rather than stall the pipeline.

---

## Deployment

The plugin binary must be placed in the directory named by `[plugins].dir` and be executable. If
`mcpd` runs in a container, the plugin must be built for the container's platform, not the host's.
See [Plugin Directory](plugin-configuration.md#plugin-directory) and
[Architecture and libc compatibility](plugin-configuration.md#architecture-and-libc-compatibility).
