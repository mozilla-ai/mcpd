# Daemon Configuration

## Global Configuration

{% hint style="info" %}
**Precedence**

The order of precedence for these options is:  
`CLI flag > configuration file > default value`
{% endhint %}

The `mcpd daemon` command supports persistent configuration through configuration files and CLI commands. 
This allows you to configure API server settings, [CORS](https://developer.mozilla.org/en-US/docs/Glossary/CORS) policies, 
and various timeout values that persist across daemon restarts.

The daemon configuration is managed using the following commands:

- `mcpd config daemon get [key]` - Retrieve configuration values
- `mcpd config daemon set <key=value>` - Set configuration values  
- `mcpd config daemon remove <key>` - Remove configuration values
- `mcpd config daemon list` - List currently configured and all available configuration keys
- `mcpd config daemon validate` - Validate current configuration

For further information, please visit our [CLI Overview](commands/mcpd.md).

---

## Configuration Structure

The daemon configuration is organized into two main sections:

### API Configuration (`api.*`)

Controls the HTTP API server settings.

| Setting                | Type       | Description                     | Default        | Example          |
|------------------------|------------|---------------------------------|----------------|------------------|
| `api.addr`             | `string`   | Server bind address (host:port) | `127.0.0.1:8090` | `0.0.0.0:8080`   |
| `api.timeout.shutdown` | `duration` | Graceful shutdown timeout       | `30s`          | `60s`            |

The daemon binds a loopback address by default, so only the machine it runs on can reach the API. Set `api.addr` to `0.0.0.0:8090` to accept connections from elsewhere, and restrict who can reach that port.

A container is the exception. Bind `0.0.0.0` inside the container, because the container network namespace is the boundary, then publish the port on the host loopback with `-p 127.0.0.1:8090:8090`. A client on a Docker network, a Compose project, a Kubernetes cluster or a CI job reaches the daemon by service name and needs no published port.

#### CORS Configuration (`api.cors.*`)

Cross-Origin Resource Sharing settings for browser clients.

| Setting                      | Type       | Description                                             | Default                             | Example                                                |
|------------------------------|------------|---------------------------------------------------------|-------------------------------------|--------------------------------------------------------|
| `api.cors.enable`            | `bool`     | Enable CORS support                                     | `false`                             | `true`                                                 |
| `api.cors.allow_origins`     | `[]string` | Allowed request origins (required when CORS is enabled) | `[]`                                | `["http://localhost:3000", "https://app.example.com"]` |
| `api.cors.allow_methods`     | `[]string` | Allowed HTTP methods                                    | `["GET", "POST", "PUT", "DELETE", "OPTIONS"]`  | `["GET", "POST"]`                                      |
| `api.cors.allow_headers`     | `[]string` | Allowed request headers                                 | `["Content-Type", "Authorization"]` | `["Content-Type", "API-Key"]`                          |
| `api.cors.expose_headers`    | `[]string` | Headers exposed to client                               | `[]`                                | `["ETag", "Last-Modified"]`                            |
| `api.cors.allow_credentials` | `bool`     | Allow credentials in requests                           | `false`                             | `true`                                                 |
| `api.cors.max_age`           | `duration` | Preflight cache duration                                | `0s`                                | `24h`                                                  |

An origin is a scheme, a host and an optional port, such as `http://localhost:3000`. A bare `localhost:3000` is not an origin, because a browser never sends one in the `Origin` header, so it is rejected.

`api.cors.allow_origins` is required when `api.cors.enable` is `true`. An empty list allows every origin, so the choice must be explicit. Use `["*"]` to allow any origin.

### MCP Configuration (`mcp.*`)

Model Context Protocol server management settings.

| Setting                | Type       | Description                   | Default | Example |
|------------------------|------------|-------------------------------|---------|---------|
| `mcp.timeout.init`     | `duration` | Server initialization timeout | `30s`   | `60s`   |
| `mcp.timeout.shutdown` | `duration` | Server shutdown timeout       | `10s`   | `30s`   |
| `mcp.timeout.health`   | `duration` | Health check timeout          | `5s`    | `10s`   |
| `mcp.timeout.request`  | `duration` | Tool call request timeout     | `15s`   | `60s`   |
| `mcp.interval.health`  | `duration` | Health check interval         | `30s`   | `60s`   |

## Configuration Examples

### Basic API Configuration

```bash
# Set server address
mcpd config daemon set api.addr="localhost:8080"

# Configure shutdown timeout
mcpd config daemon set api.timeout.shutdown="60s"
```

### CORS Configuration

```bash
# Set allowed origins, which CORS cannot be enabled without
mcpd config daemon set api.cors.allow_origins="http://localhost:3000,https://app.example.com"

# Enable CORS
mcpd config daemon set api.cors.enable=true

# Allow credentials
mcpd config daemon set api.cors.allow_credentials=true

# Set preflight cache duration
mcpd config daemon set api.cors.max_age="24h"
```

### MCP Server Configuration

```bash
# Set longer initialization timeout
mcpd config daemon set mcp.timeout.init="60s"

# Configure health check frequency
mcpd config daemon set mcp.interval.health="60s"

# Set health check timeout
mcpd config daemon set mcp.timeout.health="10s"

# Set tool call request timeout
mcpd config daemon set mcp.timeout.request="60s"
```

### Retrieving Configuration

```bash
# Get all configuration
mcpd config daemon get

# Get API configuration only
mcpd config daemon get api

# Get specific setting
mcpd config daemon get api.cors.enable

# List all configured keys
mcpd config daemon list

# List all available keys
mcpd config daemon list --available
```

### Removing Configuration

```bash
# Remove a specific setting (reverts to default)
mcpd config daemon remove api.cors.enable

# Remove entire section
mcpd config daemon remove api.cors

# Remove multiple settings
mcpd config daemon remove api.cors.enable api.cors.max_age
```

## Configuration File Storage

Configuration is stored in the `.mcpd.toml` file in TOML format:

```toml
[[servers]]
  name = "time"
  package = "uvx::mcp-server-time@2025.8.4"
  tools = ["get_current_time", "convert_time"]

[daemon]
  [daemon.api]
    addr = "localhost:8080"
    [daemon.api.timeout]
      shutdown = "1m0s"
    [daemon.api.cors]
      enable = true
      allow_origins = ["http://localhost:3000", "https://app.example.com"]
      allow_credentials = true
      max_age = "24h0m0s"
  [daemon.mcp]
    [daemon.mcp.timeout]
      shutdown = "30s"
      init = "1m0s"
      health = "10s"
    [daemon.mcp.interval]
      health = "1m0s"
```

## Data Types

### Duration Format

Duration values follow this time/duration format:

- `30s` - 30 seconds  
- `5m` - 5 minutes
- `2h` - 2 hours
- `1m30s` - 1 minute 30 seconds

### String Arrays

String arrays can be provided as comma-separated values:
```bash
mcpd config daemon set api.cors.allow_origins "http://localhost:3000,https://app.example.com"
```

### Boolean Values

Boolean values should use: `true`, `false`.

## Configuration Validation

Use the `validate` command to check your configuration:

```bash
mcpd config daemon validate
```

Common validation errors:
- Invalid address formats (must be `host:port`)
- Invalid duration formats  
- Invalid CORS origin URLs

For step-by-step diagnosis of browser requests, CORS issues, or unexpected API addresses, see [Troubleshooting](troubleshooting.md).
