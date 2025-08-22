package config

import (
	"slices"
	"strings"

	"github.com/mozilla-ai/mcpd/v2/internal/context"
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
	SaveConfig() error
}

// Setter defines the interface for setting configuration values using dot-separated path notation.
// Implementations should handle path routing, value parsing, and validation appropriate to their level.
type Setter interface {
	// Set applies a configuration value using dot-separated path notation.
	// An empty value removes/clears the configuration at the given path.
	// Returns the operation performed (Created, Updated, Deleted, Noop) and any validation error.
	Set(path string, value string) (context.UpsertResult, error)
}

// Getter defines the interface for getting configuration values using path segments.
// Implementations should handle path routing and value retrieval appropriate to their level.
type Getter interface {
	// Get retrieves a configuration value using path segments, or all configured values when no key specified.
	// Single argument gets a specific key, multiple arguments traverse nested structure.
	// Returns the value or any error encountered during retrieval.
	// NOTE: When used without any keys, no errors are returned for missing configuration.
	Get(keys ...string) (any, error)
}

// SchemaProvider defines the interface for getting available configuration keys.
// Implementations should return all possible configuration keys with their types and descriptions.
type SchemaProvider interface {
	// AvailableKeys returns all configuration keys available at this level.
	// Keys are returned without prefixes - parent sections add prefixes when recursing.
	AvailableKeys() []SchemaKey
}

// Validator defines the interface for validating configuration values.
// Implementations should validate their own fields and recurse to child sections.
type Validator interface {
	// Validate checks the configuration for errors and returns combined validation errors.
	// Uses errors.Join to combine multiple validation errors from child sections.
	Validate() error
}

// SchemaKey represents a single configuration key with metadata.
type SchemaKey struct {
	// Path is the configuration key path (e.g., "addr", "enable", "shutdown").
	Path string
	// Type describes the expected value type (e.g., "string", "bool", "duration", "[]string").
	Type string
	// Description provides a human-readable explanation of the configuration key.
	Description string
}

type DefaultLoader struct{}

// Config represents the .mcpd.toml file structure.
type Config struct {
	Servers        []ServerEntry `toml:"servers"`
	Daemon         *DaemonConfig `toml:"daemon,omitempty"`
	configFilePath string        `toml:"-"`
}

// ServerEntry represents the configuration of a single versioned MCP Server and tools.
type ServerEntry struct {
	// Name is the unique name/ID from the registry, referenced by the user.
	// e.g. 'github-server'
	Name string `json:"name" toml:"name" yaml:"name"`

	// Package contains the identifier including runtime and version.
	// e.g. 'uvx::modelcontextprotocol/github-server@1.2.3'
	Package string `json:"package" toml:"package" yaml:"package"`

	// Tools lists the names of the tools which should be allowed on this server.
	// e.g. 'create_repository'
	Tools []string `json:"tools" toml:"tools" yaml:"tools"`

	// RequiredEnvVars captures any environment variables required to run the server.
	RequiredEnvVars []string `json:"requiredEnv,omitempty" toml:"required_env,omitempty" yaml:"required_env,omitempty"`

	// RequiredPositionalArgs captures any command line args that must be positional, and which are required to run the server.
	// The arguments must be ordered by their position (ascending).
	RequiredPositionalArgs []string `json:"requiredPositionalArgs,omitempty" toml:"required_args_positional,omitempty"`

	// RequiredValueArgs captures any command line args that need values, which are required to run the server.
	RequiredValueArgs []string `json:"requiredArgs,omitempty" toml:"required_args,omitempty" yaml:"required_args,omitempty"`

	// RequiredBoolArgs captures any command line args that are boolean flags when present, which are required to run the server.
	RequiredBoolArgs []string `json:"requiredArgsBool,omitempty" toml:"required_args_bool,omitempty" yaml:"required_args_bool,omitempty"`
}

type serverKey struct {
	Name    string
	Package string // NOTE: without version
}

// argEntry represents a parsed command line argument.
type argEntry struct {
	key   string
	value string
}

func (s *ServerEntry) PackageVersion() string {
	versionDelim := "@"
	pkg := stripPrefix(s.Package)

	if idx := strings.LastIndex(pkg, versionDelim); idx != -1 {
		return pkg[idx+len(versionDelim):]
	}
	return pkg
}

func (s *ServerEntry) PackageName() string {
	return stripPrefix(stripVersion(s.Package))
}

func (e *argEntry) String() string {
	if e.hasValue() {
		return e.key + FlagValueSeparator + e.value
	}
	return e.key
}

// RequiredArguments returns all required CLI arguments, including positional, value-based and boolean flags.
// NOTE: The order of these arguments matters, so positional arguments appear first.
func (s *ServerEntry) RequiredArguments() []string {
	out := make([]string, 0, len(s.RequiredPositionalArgs)+len(s.RequiredValueArgs)+len(s.RequiredBoolArgs))

	// Add positional args first.
	out = append(out, s.RequiredPositionalArgs...)
	out = append(out, s.RequiredValueArgs...)
	out = append(out, s.RequiredBoolArgs...)

	return out
}

// Equals compares two ServerEntry instances for equality.
// Returns true if all fields are equal.
// RequiredPositionalArgs order matters (positional), all other slices are order-independent.
func (s *ServerEntry) Equals(other *ServerEntry) bool {
	if other == nil {
		return false
	}

	// Compare basic fields.
	if s.Name != other.Name {
		return false
	}

	if s.Package != other.Package {
		return false
	}

	// RequiredPositionalArgs order matters since they're positional.
	if !slices.Equal(s.RequiredPositionalArgs, other.RequiredPositionalArgs) {
		return false
	}

	// All other slices are flags, so order doesn't matter.
	// NOTE: We are assuming that tools are always already normalized, ready for comparison.
	if !equalStringSlicesUnordered(s.Tools, other.Tools) {
		return false
	}

	if !equalStringSlicesUnordered(s.RequiredEnvVars, other.RequiredEnvVars) {
		return false
	}

	if !equalStringSlicesUnordered(s.RequiredValueArgs, other.RequiredValueArgs) {
		return false
	}

	if !equalStringSlicesUnordered(s.RequiredBoolArgs, other.RequiredBoolArgs) {
		return false
	}

	return true
}

// EqualExceptTools compares this server with another and returns true if only the Tools field differs.
// All other configuration fields must be identical for this to return true.
func (s *ServerEntry) EqualExceptTools(other *ServerEntry) bool {
	if other == nil {
		return false
	}

	// Create copies with identical Tools to compare everything else.
	a := s
	b := other

	// Temporarily set tools to be identical for comparison.
	bTools := b.Tools
	b.Tools = a.Tools

	// If everything else is equal, then only tools differ.
	equalIgnoringTools := a.Equals(b)

	// Restore original tools.
	b.Tools = bTools

	// Return true only if everything else is equal AND tools actually differ.
	// NOTE: We are assuming that tools are always already normalized, ready for comparison.
	return equalIgnoringTools && !equalStringSlicesUnordered(s.Tools, other.Tools)
}

// equalStringSlicesUnordered compares two string slices for equality, ignoring order.
func equalStringSlicesUnordered(a []string, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	x := slices.Clone(a)
	y := slices.Clone(b)

	slices.Sort(x)
	slices.Sort(y)

	return slices.Equal(x, y)
}

// hasValue is used to determine if an argEntry is a bool flag or contains a value.
func (e *argEntry) hasValue() bool {
	return strings.TrimSpace(e.value) != ""
}
