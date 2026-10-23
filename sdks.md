# SDKs

`mcpd` exposes an HTTP REST API, so any language that can make an HTTP request can drive it. The
SDKs on this page remove the boilerplate.

There are two kinds, and they answer different questions:

| Kind       | Question it answers                              | Runs in                |
|------------|--------------------------------------------------|------------------------|
| Client SDK | How does my application call the tools?          | Your agent application |
| Plugin SDK | How do I intercept what passes through `mcpd`?   | A plugin process       |

---

## Client SDKs

A client SDK lists the servers `mcpd` manages, reads tool schemas, and calls tools from your own
code.

| Language              | Package              | Repository                                                                  | Install                      |
|-----------------------|----------------------|-----------------------------------------------------------------------------|------------------------------|
| Python                | `mcpd`               | [mcpd-sdk-python](https://github.com/mozilla-ai/mcpd-sdk-python)             | `uv add mcpd`                |
| JavaScript/TypeScript | `@mozilla-ai/mcpd`   | [mcpd-sdk-javascript](https://github.com/mozilla-ai/mcpd-sdk-javascript)     | `npm install @mozilla-ai/mcpd` |

Both follow the same shape. You construct a client against the daemon address, then reach a tool
through the server that provides it:

```python
from mcpd import McpdClient

client = McpdClient(api_endpoint="http://localhost:8090")
result = client.call.time.get_current_time(timezone="UTC")
```

```typescript
import { McpdClient } from "@mozilla-ai/mcpd";

const client = new McpdClient({ apiEndpoint: "http://localhost:8090" });
const result = await client.servers.time.tools.get_current_time({ timezone: "UTC" });
```

Each SDK also hands a framework its tool definitions in one call, so you do not write a conversion
layer. The Python SDK targets [any-agent](https://github.com/mozilla-ai/any-agent), and the
JavaScript SDK targets LangChain JS and the Vercel AI SDK.

{% hint style="info" %}
**The endpoint is the daemon's API address**

`mcpd daemon` binds `0.0.0.0:8090` unless `api.addr` says otherwise, so `http://localhost:8090`
works for a daemon on the same machine. See [Daemon Configuration](daemon-configuration.md) when
the daemon runs elsewhere, and [Execution Context](execution-context.md) for API key handling.
{% endhint %}

---

## Plugin SDKs

A plugin SDK wraps the gRPC stubs generated from
[`mcpd-proto`](https://github.com/mozilla-ai/mcpd-proto) and the startup handshake, so a plugin only
has to implement the behavior you care about. [Writing a Plugin](writing-a-plugin.md) covers the
protocol itself.

| Language | Package                           | Repository                                                                       | Install                                         |
|----------|-----------------------------------|----------------------------------------------------------------------------------|-------------------------------------------------|
| .NET     | `MozillaAI.Mcpd.Plugins.Sdk`      | [mcpd-plugins-sdk-dotnet](https://github.com/mozilla-ai/mcpd-plugins-sdk-dotnet)  | `dotnet add package MozillaAI.Mcpd.Plugins.Sdk` |
| Go       | `mcpd-plugins-sdk-go`             | [mcpd-plugins-sdk-go](https://github.com/mozilla-ai/mcpd-plugins-sdk-go)          | `go get github.com/mozilla-ai/mcpd-plugins-sdk-go` |
| Python   | `mcpd-plugins`                    | [mcpd-plugins-sdk-python](https://github.com/mozilla-ai/mcpd-plugins-sdk-python)  | `pip install mcpd-plugins`                      |
| Rust     | `mcpd-plugins-sdk`                | [mcpd-plugins-sdk-rust](https://github.com/mozilla-ai/mcpd-plugins-sdk-rust)      | `cargo add mcpd-plugins-sdk`                    |

The Python, Rust, and .NET repositories each include an `examples/` directory with runnable
plugins.

### Community plugin SDKs

These are maintained by their authors. Mozilla AI does not maintain or test them.

| Language | Repository                                                                | Maintainer                                 |
|----------|---------------------------------------------------------------------------|--------------------------------------------|
| PHP      | [mcpd-plugins-sdk-php](https://github.com/shoemoney/mcpd-plugins-sdk-php)  | [@shoemoney](https://github.com/shoemoney) |

---

## No SDK for your language

Write against the API directly. The [API Reference](api-reference.md) documents every endpoint, and
`mcpd daemon` serves the same specification at `/docs`.

To build a plugin instead, generate stubs from
[`mcpd-proto`](https://github.com/mozilla-ai/mcpd-proto) with your language's gRPC tooling and
implement the handshake described in [Writing a Plugin](writing-a-plugin.md).
