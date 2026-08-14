# Plugin Configuration

## Overview

The `mcpd` daemon supports a plugin subsystem for extending request/response processing.

This page covers configuring plugins. To build one, see [Writing a Plugin](writing-a-plugin.md).

---

## Plugin Categories

{% hint style="info" %}
**Plugin execution order**

Within each category, plugins execute in the order they appear in the configuration file.
{% endhint %}

Plugins are organized into categories and execute during specific phases of the request lifecycle.

Categories execute in the order shown below for both request and response phases.

| Order | Category         | Purpose                                      | Execution  |
|-------|------------------|----------------------------------------------|------------|
| 1     | `observability`  | Collect metrics and traces (non-blocking)    | Parallel   |
| 2     | `authentication` | Validate client identity                     | Sequential |
| 3     | `authorization`  | Verify permissions after authentication      | Sequential |
| 4     | `rate_limiting`  | Enforce request rate limits                  | Sequential |
| 5     | `validation`     | Check request/response structure and content | Sequential |
| 6     | `content`        | Transform request/response payloads          | Sequential |
| 7     | `audit`          | Log compliance and security events           | Sequential |

---

## Plugin Execution Flows

Plugins can execute during one or both flows/phases:

* `request`: Executes during the request phase
* `response`: Executes during the response phase

---

## Configuration Format

```toml
[[servers]]
  name = "api-server"
  package = "uvx::api-server@1.0.0"
  tools = ["create", "read", "update", "delete"]

[plugins]
  dir = "/etc/mcpd/plugins"

[[plugins.authentication]]
  name = "jwt-auth"
  commit_hash = "abc123"
  required = true
  flows = ["request"]

[[plugins.authentication]]
  name = "api-key-auth"
  flows = ["request", "response"]

[[plugins.authorization]]
  name = "rbac"
  required = true
  flows = ["request"]

[[plugins.observability]]
  name = "metrics"
  flows = ["request", "response"]
```

---

## Plugin Directory

Plugins are native executables. `[plugins].dir` is the directory `mcpd` scans to find them, and
every plugin's `name` must match a binary in that directory.

| Field | Type   | Required            | Description                                    |
|-------|--------|---------------------|------------------------------------------------|
| `dir` | string | Yes, if any plugins | Directory containing the plugin binaries       |

When scanning the directory, `mcpd` only considers **regular files with the execute bit set**.
Subdirectories, dotfiles, and non-executable files are skipped silently, so a plugin whose binary
lost its execute permission looks identical to one that was never there.

Both problems are caught at startup rather than at request time:

```console
# dir does not exist
Error: failed to load configuration: plugin directory /etc/mcpd/plugins: open /etc/mcpd/plugins: no such file or directory

# binary missing, not executable, or a dotfile
Error: failed to load configuration: plugin jwt-auth not found in directory /etc/mcpd/plugins
```

If you see the second error and the file is definitely there, check `ls -l` for the execute bit
before anything else.

---

## Plugin Fields

| Field         | Type    | Required | Description                                          |
|---------------|---------|----------|------------------------------------------------------|
| `name`        | string  | Yes      | Name of the plugin binary in the plugins directory   |
| `commit_hash` | string  | No       | SHA/hash for validating plugin version               |
| `required`    | boolean | No       | Whether plugin failure should block the request      |
| `flows`       | array   | Yes      | Execution phases: ["request"], ["response"], or both |

---

## Execution Order

Plugins execute in the order they appear in the configuration file within their category.

```toml
[[plugins.authentication]]
  name = "jwt-auth"
  flows = ["request"]

[[plugins.authentication]]
  name = "api-key-auth"
  flows = ["request"]
```

During the request phase, `jwt-auth` executes first, followed by `api-key-auth`.

---

## Required Plugins

{% hint style="warning" %}
**Required Plugin Failures**

If a required plugin fails, the request is rejected with HTTP 500 (Internal Server Error)
and a `Mcpd-Error-Type` header indicating the failure phase.
{% endhint %}


Mark plugins as required when their successful execution is critical:

```toml
[[plugins.authentication]]
  name = "jwt-auth"
  required = true
  flows = ["request"]
```

### Failure Behavior

When a required plugin fails, `mcpd` returns:

* Status: 500 Internal Server Error
* Header: `Mcpd-Error-Type` with one of:
    * `request-pipeline-failure` - Plugin failed during request processing (before upstream call)
    * `response-pipeline-failure` - Plugin failed during response processing (after upstream call)

{% hint style="info" %}
**Response Pipeline Execution**

The response pipeline runs on all upstream responses, regardless of status (200 OK, 500 error, etc.).
This ensures critical plugins (PII redaction, audit logging, security headers) run consistently.
{% endhint %}

### Optional Plugin Behavior

When `required` is not specified or set to `false`:

* **Plugin errors** (crashes, exceptions): Logged as warnings, pipeline continues.
* **Plugin rejections** (returning `Continue=false`): Pipeline respects the rejection and stops processing, **except**:
    * **Observability category only**: Pipeline ignores optional plugin rejections and continues (necessary for parallel execution model).

---

## Content Mutation

{% hint style="info" %}
**Content Plugin Behavior**

Only plugins in the `content` category may mutate requests or responses. Modified content is passed to the next plugin in the chain.
{% endhint %}

Content plugins modify the request by setting the modified request in their response. Other plugin categories can only observe or reject requests.

### Example Content Plugin Flow

```toml
[[plugins.content]]
  name = "encryption"
  flows = ["request"]

[[plugins.content]]
  name = "compression"
  flows = ["request"]
```

The `encryption` plugin processes the request first and may modify it. The modified request is then passed to the `compression` plugin.

---

## Observability Plugin Execution

{% hint style="info" %}
**Parallel Execution**

Observability plugins run in *parallel* and cannot modify requests or responses.
{% endhint %}

Observability plugins are designed for metrics collection, tracing, and monitoring. They execute concurrently for performance.

### Required Observability Plugins

If any observability plugin is marked as `required`, request processing waits for all observability plugins to complete before aggregating results. 
If any required observability plugin fails, the request is rejected after all have completed.

```toml
[[plugins.observability]]
  name = "metrics"
  required = true
  flows = ["request", "response"]

[[plugins.observability]]
  name = "tracing"
  flows = ["request", "response"]
```

In this example, both `metrics` and `tracing` run in parallel, but the request will be rejected if `metrics` fails 
(once `metrics` and `tracing` have completed).

---

## Multiple Plugins Per Category

You can configure multiple plugins within the same category. They execute in the order defined:

```toml
[[plugins.authentication]]
  name = "jwt-auth"
  required = true
  flows = ["request"]

[[plugins.authentication]]
  name = "api-key-auth"
  flows = ["request"]

[[plugins.authentication]]
  name = "oauth2"
  flows = ["request"]
```

Request processing order: `jwt-auth` → `api-key-auth` → `oauth2`

---

## Minimal Configuration

Plugins are optional. A configuration file without plugins is valid:

```toml
[[servers]]
  name = "simple-server"
  package = "uvx::simple@1.2.3"
  tools = ["tool1"]
```

---

## Complete Example

```toml
[[servers]]
  name = "production-api"
  package = "uvx::api-server@2.0.0"
  tools = ["create_user", "get_user", "update_user", "delete_user"]

[[plugins.authentication]]
  name = "jwt-auth"
  commit_hash = "a1b2c3d4"
  required = true
  flows = ["request"]

[[plugins.authorization]]
  name = "rbac"
  commit_hash = "e5f6g7h8"
  required = true
  flows = ["request"]

[[plugins.rate_limiting]]
  name = "token-bucket"
  flows = ["request"]

[[plugins.validation]]
  name = "schema-validator"
  required = true
  flows = ["request", "response"]

[[plugins.content]]
  name = "encryption"
  flows = ["request", "response"]

[[plugins.observability]]
  name = "prometheus-metrics"
  required = true
  flows = ["request", "response"]

[[plugins.observability]]
  name = "distributed-tracing"
  flows = ["request", "response"]

[[plugins.audit]]
  name = "compliance-logger"
  required = true
  flows = ["response"]
```

### Execution Flow

#### Request Phase

1. `jwt-auth` (authentication) - sequential
2. `rbac` (authorization) - sequential
3. `token-bucket` (rate_limiting) - sequential
4. `schema-validator` (validation) - sequential
5. `encryption` (content) - sequential
6. `prometheus-metrics` + `distributed-tracing` (observability) - parallel

#### Response Phase

1. `schema-validator` (validation) - sequential
2. `encryption` (content) - sequential
3. `prometheus-metrics` + `distributed-tracing` (observability) - parallel
4. `compliance-logger` (audit) - sequential

---

## Running Plugins in Docker

A containerized `mcpd` cannot see plugin binaries on the host unless the directory is bind
mounted. Mount it and point `[plugins].dir` at the path **inside** the container:

```bash
docker run  -p 8090:8090 \
            -v $PWD/.mcpd.toml:/etc/mcpd/.mcpd.toml \
            -v $PWD/plugins:/etc/mcpd/plugins:ro \
            -v $HOME/.config/mcpd/secrets.dev.toml:/home/mcpd/.config/mcpd/secrets.prod.toml \
            mzdotai/mcpd:v0.4.0
```

```toml
[plugins]
  dir = "/etc/mcpd/plugins"
```

The image runs as the non-root `mcpd` user, so the mounted binaries must be readable and
executable by that user, not only by whoever owns them on the host. The mount is shown read-only
because `mcpd` only needs to execute the plugins, never write to them.

### Architecture and libc compatibility

Plugins are separate executables that `mcpd` launches and talks to over gRPC, so each one must be
built for the OS, architecture, **and C library** of whatever runs `mcpd` — which is the container
image, not your laptop, whenever `mcpd` itself is containerized.

The published image is multi-arch — `mzdotai/mcpd:v0.4.0` ships both `linux/amd64` and
`linux/arm64` — so Docker pulls the variant matching your host, and the plugin must be built for
*that* architecture. It is also Alpine-based (musl), so a plugin dynamically linked against glibc
will not run even at the correct architecture. For Go plugins, build a static binary for the
architecture Docker pulled:

```bash
# arm64 host (Apple Silicon, arm64 Linux)
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o plugins/jwt-auth ./cmd/jwt-auth

# amd64 host
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o plugins/jwt-auth ./cmd/jwt-auth
```

Run `docker image inspect mzdotai/mcpd:v0.4.0 --format '{{.Architecture}}'` if you are unsure which
one you have, or if you pinned a platform with `--platform`. `CGO_ENABLED=0` produces a static
binary and sidesteps the glibc/musl question entirely; if you need cgo, build against musl.

The failure is easy to misread, because discovery succeeds and only the launch fails:

| Symptom                              | Cause                                                    |
|--------------------------------------|----------------------------------------------------------|
| `exec format error`                  | Wrong architecture or OS for the running platform        |
| `no such file or directory`, but the binary exists | Dynamically linked against a libc the image lacks (typically glibc on musl) |
| `permission denied`                  | Execute bit set, but not executable by the `mcpd` user (wrong owner/group), or unreadable |

A binary with no execute bit at all never reaches this stage — it is skipped during discovery and
produces the *"plugin … not found in directory"* startup error described above instead.

The second row is genuinely confusing: the kernel reports the missing *interpreter*, not the
missing binary. Check with `file plugins/jwt-auth` — it is portable and reports "statically linked"
vs "dynamically linked". (`ldd` is Linux-only, so it is unavailable on the macOS host you may be
cross-compiling from, and glibc's `ldd` prints "not a dynamic executable" for a correct
`CGO_ENABLED=0` binary.)

---

## Validation

Plugin configurations are validated when the daemon starts or during hot reload. Common validation errors:

* Empty plugin name
* Missing or empty `flows` array
* Invalid flow values (must be `request` or `response`)
* Duplicate flow values
* Plugin directory missing, or configured plugin not found in it (see [Plugin Directory](#plugin-directory))

For runtime plugin failures or binary checks, see [Troubleshooting](troubleshooting.md).
