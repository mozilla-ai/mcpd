package config

// Config represents the .mcpd.toml file structure.
type Config struct {
	Servers        []ServerEntry `toml:"servers"`
	configFilePath string
}

// ServerEntry represents the configuration of a single versioned MCP Server and optional tools.
type ServerEntry struct {
	// Name is the unique name referenced by the user.
	// e.g. 'github-server'
	Name string `toml:"name"`

	// Package contains the identifier including version.
	// e.g. 'modelcontextprotocol/github-server@latest'
	Package string `toml:"package"`

	// Tools are optional and list the names of the allowed tools on this server.
	// e.g. 'create_repository'
	Tools []string `toml:"tools,omitempty"`
}

type serverKey struct {
	Name    string
	Package string // NOTE: without version
}
