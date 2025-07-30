package config

import (
	"strings"
)

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
	configFilePath string        `toml:"-"`
}

// ServerEntry represents the configuration of a single versioned MCP Server and tools.
type ServerEntry struct {
	// Name is the unique name/ID from the registry, referenced by the user.
	// e.g. 'github-server'
	Name string `toml:"name" json:"name" yaml:"name"`

	// Package contains the identifier including runtime and version.
	// e.g. 'uvx::modelcontextprotocol/github-server@1.2.3'
	Package string `toml:"package" json:"package" yaml:"package"`

	// Tools lists the names of the tools which should be allowed on this server.
	// e.g. 'create_repository'
	Tools []string `toml:"tools" json:"tools" yaml:"tools"`

	// RequiredEnvVars captures any environment variables required to run the server.
	RequiredEnvVars []string `toml:"required_env,omitempty" json:"required_env,omitempty" yaml:"required_env,omitempty"`

	// RequiredArgs captures any command line args required to run the server.
	RequiredArgs []string `toml:"required_args,omitempty" json:"required_args,omitempty" yaml:"required_args,omitempty"`
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

// argEntry represents a parsed command line argument.
type argEntry struct {
	key   string
	value string
}

// hasValue is used to determine if an argEntry is a bool flag or contains a value.
func (e *argEntry) hasValue() bool {
	return strings.TrimSpace(e.value) != ""
}

func (e *argEntry) String() string {
	if e.hasValue() {
		return e.key + FlagValueSeparator + e.value
	}
	return e.key
}
