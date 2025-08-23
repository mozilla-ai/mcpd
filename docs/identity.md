# Identity Support

mcpd includes optional support for AGNTCY Identity standards, enabling verifiable identities for MCP servers.

## Quick Start

1. Enable identity:
```bash
export MCPD_IDENTITY_ENABLED=true
```

2. Initialize identity for a server:
```bash
mcpd identity init github-server --org "MyOrg"
```

3. Start mcpd normally:
```bash
mcpd daemon
```

## How It Works

When enabled, mcpd:
- Creates AGNTCY-compatible Verifiable Credentials for servers
- Stores credentials locally in `~/.config/mcpd/identity/`
- Verifies server identities on startup (optional, non-blocking)

## Configuration

Identity is disabled by default. Enable with:
- Environment variable: `MCPD_IDENTITY_ENABLED=true`

## Future

This minimal implementation provides a foundation for:
- Integration with AGNTCY Identity Nodes
- Agent-to-Agent (A2A) secure communication
- Cross-organizational trust networks