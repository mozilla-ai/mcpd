# Plugin Configuration

## Overview

The `mcpd` daemon supports a plugin subsystem for extending request/response processing.

---

## Plugin Categories

!!! info Plugin execution order
    Within each category, plugins execute in the order they appear in the configuration file.

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

!!! warning "Required Plugin Failures"
    If a required (serial) plugin fails or rejects a request/response, the overall request is rejected immediately.

Mark plugins as required when their successful execution is critical:

```toml
[[plugins.authentication]]
  name = "jwt-auth"
  required = true
  flows = ["request"]
```

When `required` is not specified or set to `false`, plugin failures are logged but do not block the request.

---

## Content Mutation

!!! info "Content Plugin Behavior"
    Only plugins in the `content` category may mutate requests or responses. Modified content is passed to the next plugin in the chain.

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

!!! note "Parallel Execution"
    Observability plugins run in *parallel* and cannot modify requests or responses.

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

## Validation

Plugin configurations are validated when the daemon starts or during hot reload. Common validation errors:

* Empty plugin name
* Missing or empty `flows` array
* Invalid flow values (must be `request` or `response`)
* Duplicate flow values
