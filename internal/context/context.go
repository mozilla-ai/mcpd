package context

import (
	"cmp"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/mozilla-ai/mcpd/v2/internal/files"
	"github.com/mozilla-ai/mcpd/v2/internal/perms"
)

// DefaultLoader loads execution context configurations.
type DefaultLoader struct{}

// ExecutionContextConfig stores execution context data for all configured MCP servers.
type ExecutionContextConfig struct {
	Servers  map[string]ServerExecutionContext `toml:"servers"`
	filePath string                            `toml:"-"`
}

// ServerExecutionContext stores execution context data for an MCP server.
//
// The Args, Env, and Volumes fields contain expanded values with environment variables resolved.
// These should not be used directly when starting MCP servers, as they may contain
// cross-server references that pose security risks.
//
// Instead, use the server's SafeArgs(), SafeEnv(), and SafeVolumes() methods (in the runtime package)
// which filter out cross-server references using the RawArgs, RawEnv, and RawVolumes fields.
type ServerExecutionContext struct {
	// Name is the server name.
	Name string `toml:"-"`

	// Args contains command-line arguments with environment variables expanded.
	// NOTE: Use runtime.Server.SafeArgs() for filtered access when starting servers.
	Args []string `toml:"args,omitempty"`

	// Env contains environment variables with values expanded.
	// NOTE: Use runtime.Server.SafeEnv() for filtered access when starting servers.
	Env map[string]string `toml:"env,omitempty"`

	// Volumes maps volume names to their host paths or named volumes with environment variables expanded.
	// NOTE: Use runtime.Server.SafeVolumes() for filtered access when starting servers.
	Volumes VolumeExecutionContext `toml:"volumes,omitempty"`

	// RawArgs stores unexpanded command-line arguments used for cross-server filtering decisions.
	RawArgs []string `toml:"-"`

	// RawEnv stores unexpanded environment variables used for cross-server filtering decisions.
	RawEnv map[string]string `toml:"-"`

	// RawVolumes stores unexpanded volume mappings used for cross-server filtering decisions.
	RawVolumes VolumeExecutionContext `toml:"-"`
}

// VolumeExecutionContext maps volume names to their host paths or named volumes.
// e.g., {"workspace": "/Users/foo/repos/mcpd", "gdrive": "mcp-gdrive"}
type VolumeExecutionContext map[string]string

// Load loads an execution context configuration from the specified path.
func (d *DefaultLoader) Load(path string) (Modifier, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, fmt.Errorf("path cannot be empty")
	}

	cfg, err := loadExecutionContextConfig(path)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("failed to load execution context config: %w", err)
		}

		// Config doesn't exist yet, so create a new instance to interact with.
		cfg = NewExecutionContextConfig(path)
	}

	return cfg, nil
}

// Get retrieves the execution context for the specified server name.
func (c *ExecutionContextConfig) Get(name string) (ServerExecutionContext, bool) {
	name = strings.TrimSpace(name)
	if name == "" {
		return ServerExecutionContext{}, false
	}

	if srv, ok := c.Servers[name]; ok {
		return ServerExecutionContext{
			Name:       name,
			Args:       slices.Clone(srv.Args),
			Env:        maps.Clone(srv.Env),
			Volumes:    maps.Clone(srv.Volumes),
			RawArgs:    slices.Clone(srv.RawArgs),
			RawEnv:     maps.Clone(srv.RawEnv),
			RawVolumes: maps.Clone(srv.RawVolumes),
		}, true
	}

	return ServerExecutionContext{}, false
}

// List returns all server execution contexts sorted by name.
func (c *ExecutionContextConfig) List() []ServerExecutionContext {
	servers := slices.Collect(maps.Values(c.Servers))

	slices.SortFunc(servers, func(a, b ServerExecutionContext) int {
		return cmp.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	})

	return servers
}

// SaveConfig saves the execution context configuration to a file with secure permissions.
// Used for runtime execution contexts that may contain sensitive data.
func (c *ExecutionContextConfig) SaveConfig() error {
	return c.saveConfig(files.EnsureAtLeastSecureDir, perms.SecureFile)
}

// SaveExportedConfig saves the execution context configuration to a file with regular permissions.
// Used for exported configurations that are sanitized and suitable for sharing.
func (c *ExecutionContextConfig) SaveExportedConfig() error {
	return c.saveConfig(files.EnsureAtLeastRegularDir, perms.RegularFile)
}

// Upsert updates the execution context for the given server name.
// If the context is empty and does not exist in config, it does nothing.
// If the context is empty and previously existed in config, it deletes the entry.
// If the context differs from the existing one in config, it updates it.
// If the context is new and non-empty, it adds it.
// Returns the operation performed (Created, Updated, Deleted, or Noop),
// and writes changes to disk if applicable.
func (c *ExecutionContextConfig) Upsert(ec ServerExecutionContext) (UpsertResult, error) {
	if strings.TrimSpace(ec.Name) == "" {
		return Noop, fmt.Errorf("server name cannot be empty")
	}

	if len(c.Servers) == 0 {
		// We've currently got no servers stored in config.
		c.Servers = map[string]ServerExecutionContext{}
	}

	current, exists := c.Servers[ec.Name]
	var op UpsertResult

	switch {
	case !exists && ec.IsEmpty():
		return Noop, nil // Nothing existing and trying to save an empty server.
	case exists && current.Equals(ec):
		return Noop, nil // No change to existing.
	case ec.IsEmpty():
		delete(c.Servers, ec.Name) // Trying to save an empty server over an existing one that wasn't.
		op = Deleted
	case exists:
		op = Updated
		c.Servers[ec.Name] = ec
	default:
		op = Created
		c.Servers[ec.Name] = ec
	}

	if err := c.SaveConfig(); err != nil {
		return Noop, fmt.Errorf("error saving execution context config: %w", err)
	}

	return op, nil
}

// Equals checks if this ServerExecutionContext is equal to another.
func (s *ServerExecutionContext) Equals(b ServerExecutionContext) bool {
	if s.Name != b.Name {
		return false
	}

	if !equalSlices(s.Args, b.Args) {
		return false
	}

	if len(s.Env) != len(b.Env) || !maps.Equal(s.Env, b.Env) {
		return false
	}

	if !maps.Equal(s.Volumes, b.Volumes) {
		return false
	}

	if !equalSlices(s.RawArgs, b.RawArgs) {
		return false
	}

	if len(s.RawEnv) != len(b.RawEnv) || !maps.Equal(s.RawEnv, b.RawEnv) {
		return false
	}

	if !maps.Equal(s.RawVolumes, b.RawVolumes) {
		return false
	}

	return true
}

// IsEmpty returns true if the ServerExecutionContext has no args, env vars, or volumes.
func (s *ServerExecutionContext) IsEmpty() bool {
	return len(s.Args) == 0 && len(s.Env) == 0 && len(s.Volumes) == 0
}

// NewExecutionContextConfig creates a new ExecutionContextConfig with an initialized Servers map and sets its filePath to the provided path trimmed of surrounding whitespace.
func NewExecutionContextConfig(path string) *ExecutionContextConfig {
	return &ExecutionContextConfig{
		Servers:  map[string]ServerExecutionContext{},
		filePath: strings.TrimSpace(path),
	}
}

// equalSlices reports whether two string slices contain the same elements
// with the same multiplicities, regardless of order.
func equalSlices(a []string, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	sortedA := slices.Clone(a)
	slices.Sort(sortedA)

	sortedB := slices.Clone(b)
	slices.Sort(sortedB)

	return slices.Equal(sortedA, sortedB)
}

// loadExecutionContextConfig loads a runtime execution context file from disk and expands environment variables.
//
// The function parses the TOML file at the specified path and automatically expands all ${VAR} references
// in both args and env fields using os.ExpandEnv. Non-existent environment variables are expanded to
// empty strings. This ensures that the loaded configuration contains actual values ready for runtime use,
// rather than template strings that require later expansion.
func loadExecutionContextConfig(path string) (*ExecutionContextConfig, error) {
	cfg := NewExecutionContextConfig(path)

	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("execution context file '%s' does not exist: %w", path, err)
		}

		return nil, fmt.Errorf("could not stat execution context file '%s': %w", path, err)
	}

	if _, err := toml.DecodeFile(path, cfg); err != nil {
		return nil, fmt.Errorf("execution context file '%s' could not be parsed: %w", path, err)
	}

	// Manually set the name field for each ServerExecutionContext and expand all ${VAR} references.
	for name, server := range cfg.Servers {
		server.Name = name

		// Store raw args before expansion for filtering decisions.
		server.RawArgs = slices.Clone(server.Args)

		// Store raw env vars before expansion for filtering decisions.
		server.RawEnv = maps.Clone(server.Env)

		// Store raw volumes before expansion for filtering decisions.
		server.RawVolumes = maps.Clone(server.Volumes)

		// Expand args.
		for i, arg := range server.Args {
			server.Args[i] = os.ExpandEnv(arg)
		}

		// Expand env vars.
		for k, v := range server.Env {
			server.Env[k] = os.ExpandEnv(v)
		}

		// Expand volume paths.
		for k, v := range server.Volumes {
			server.Volumes[k] = os.ExpandEnv(v)
		}

		cfg.Servers[name] = server
	}

	return cfg, nil
}

// saveConfig saves the execution context configuration to a file with the specified directory and file permissions.
func (c *ExecutionContextConfig) saveConfig(ensureDirFunc func(string) error, fileMode os.FileMode) error {
	path := c.filePath
	if path == "" {
		return fmt.Errorf("config file path not present")
	}

	// Ensure the directory exists before creating the file.
	if err := ensureDirFunc(filepath.Dir(path)); err != nil {
		return fmt.Errorf("could not ensure execution context directory exists: %w", err)
	}

	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, fileMode)
	if err != nil {
		return fmt.Errorf("could not create file '%s': %w", path, err)
	}

	// Defer the closing of the file once it's opened.
	// Ensuring that if an error occurs during closing, then it can be passed back to the caller.
	defer func(f *os.File) {
		closeErr := f.Close()
		if closeErr != nil {
			err = errors.Join(err, closeErr)
		}
	}(f)

	encoder := toml.NewEncoder(f)
	if err := encoder.Encode(c); err != nil {
		return fmt.Errorf("could not encode execution context to file '%s': %w", path, err)
	}

	return nil
}