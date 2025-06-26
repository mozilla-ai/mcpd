package config

import "strings"

var (
	_ Provider = (*DefaultLoader)(nil)
	_ Modifier = (*Config)(nil)
)

type Loader interface {
	Load(path string) (Modifier, error)
}

type Initializer interface {
	Init(path string) error
}

type Provider interface {
	Initializer
	Loader
}

type Modifier interface {
	AddServer(entry ServerEntry) error
	RemoveServer(name string) error
	ListServers() []ServerEntry
}

type DefaultLoader struct{}

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

	// Package contains the identifier including runtime and version.
	// e.g. 'uvx::modelcontextprotocol/github-server@latest'
	Package string `toml:"package"`

	// Tools are optional and list the names of the allowed tools on this server.
	// e.g. 'create_repository'
	Tools []string `toml:"tools,omitempty"`
}

type serverKey struct {
	Name    string
	Package string // NOTE: without version
}

func (e *ServerEntry) PackageVersion() string {
	versionDelim := "@"
	pkg := stripPrefix(e.Package)

	if idx := strings.LastIndex(pkg, versionDelim); idx != -1 {
		return pkg[idx+len(versionDelim):]
	}
	return pkg
}

func (e *ServerEntry) PackageName() string {
	return stripPrefix(stripVersion(e.Package))
}
