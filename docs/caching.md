# Registry Caching

`mcpd` includes a built-in caching system to improve performance when working with remote MCP server registries. 
The caching system stores registry manifests locally to avoid repeated network requests.

## Cache Directory

All commands that access remote registries ([add](/mcpd/commands/mcpd_add/) and [search](/mcpd/commands/mcpd_search/)) support optional parameters to configure caching behavior.

You can specify the cache directory in multiple ways:

- CLI flag: `--cache-dir <path>`
- Default: `~/.cache/mcpd/registries/`

!!! note "XDG_CACHE_HOME environment variable"
    `mcpd` honors the [XDG Base Directory Specification](https://specifications.freedesktop.org/basedir-spec/latest/), 
    respecting the `XDG_CACHE_HOME` environment variable. This forms the base directory where `mcpd` will create a 
    cache folder for registry manifests.

## Cache Time-to-Live (TTL)

You can configure how long cached registry manifests remain valid:

- CLI flag: `--cache-ttl <duration>`
- Default: `24h`

The duration format accepts values like:

- `1h` (1 hour)
- `30m` (30 minutes)
- `24h` (24 hours)
- `1h30m` (1 hour 30 minutes)

## Caching Control

### Disabling Cache

To disable caching entirely and always fetch fresh data:

```bash
mcpd add my-server --no-cache
mcpd search my-query --no-cache
```

### Refreshing Cache

To force refresh cached manifests (ignoring TTL):

```bash
mcpd add my-server --refresh-cache
mcpd search my-query --refresh-cache
```

## Cache Behavior

### When Caching is Enabled (Default)

1. **First Request**: Downloads registry manifest from remote URL and stores it in the cache directory
2. **Subsequent Requests**: Uses cached file if it exists and hasn't expired (based on TTL)
3. **Expired Cache**: Automatically downloads fresh manifest when TTL expires
4. **Cache Miss**: Falls back to remote URL if cache file is corrupted or missing

### When Caching is Disabled

1. **No Directory Creation**: Cache directory is never created on the filesystem
2. **Always Remote**: All requests go directly to remote registry URLs
3. **No Storage**: No files are written to disk

### Cache File Naming

Cache files are stored using SHA-256 hashes of the registry URLs:
```
~/.cache/mcpd/registries/
├── a1b2c3d4e5f6...1234.json  # mcpm registry manifest
└── ...
```

## Examples

### Basic Usage with Custom Cache Directory

```bash
# Use temporary cache directory
mcpd add github-mcp --cache-dir /tmp/mcpd-cache

# Set custom TTL to 1 hour
mcpd search database --cache-ttl 1h
```

### Combining Cache Options

```bash
# Custom directory with forced refresh
mcpd search api --cache-dir ./project-cache --refresh-cache

# Disable caching but specify directory (directory won't be created)
mcpd add server --no-cache --cache-dir /unused/path
```

## Troubleshooting

### Cache Issues

If you experience issues with cached data:

1. **Force Refresh**: Use `--refresh-cache` to download fresh manifests
2. **Clear Cache**: Delete cache directory contents manually
3. **Disable Temporarily**: Use `--no-cache` to bypass cache entirely

### Disk Space

Cache files are relatively small JSON manifests (typically a few MB each), but you can:

- Set shorter TTL to reduce cache lifetime: `--cache-ttl 1h`
- Use `--no-cache` for one-off operations
- Periodically clean the cache directory

### Permissions

If cache directory creation fails, ensure:

- Parent directory is writable
- Sufficient disk space is available  
- No conflicting files exist at the cache path

!!! tip "Performance"
    Caching significantly improves performance for repeated operations. The default 24-hour TTL 
    provides a good balance between freshness and performance for most use cases.