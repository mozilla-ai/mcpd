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

	"github.com/mozilla-ai/mcpd/v2/internal/perms"
)

const (
	// EnvVarXDGConfigHome is the XDG Base Directory env var name for config files.
	EnvVarXDGConfigHome = "XDG_CONFIG_HOME"

	// EnvVarXDGCacheHome is the XDG Base Directory env var name for cache file.
	EnvVarXDGCacheHome = "XDG_CACHE_HOME"
)

// ServerExecutionContext stores execution context data for an MCP server.
type ServerExecutionContext struct {
	Name string            `toml:"-"`
	Args []string          `toml:"args,omitempty"`
	Env  map[string]string `toml:"env,omitempty"`
}

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

	return true
}

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

func (s *ServerExecutionContext) IsEmpty() bool {
	return len(s.Args) == 0 && len(s.Env) == 0
}

type DefaultLoader struct{}

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

// ExecutionContextConfig stores execution context data for all configured MCP servers.
type ExecutionContextConfig struct {
	Servers  map[string]ServerExecutionContext `toml:"servers"`
	filePath string                            `toml:"-"`
}

// NewExecutionContextConfig returns a newly initialized ExecutionContextConfig.
func NewExecutionContextConfig(path string) *ExecutionContextConfig {
	return &ExecutionContextConfig{
		Servers:  map[string]ServerExecutionContext{},
		filePath: strings.TrimSpace(path),
	}
}

func (c *ExecutionContextConfig) List() []ServerExecutionContext {
	servers := slices.Collect(maps.Values(c.Servers))

	slices.SortFunc(servers, func(a, b ServerExecutionContext) int {
		return cmp.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	})

	return servers
}

func (c *ExecutionContextConfig) Get(name string) (ServerExecutionContext, bool) {
	name = strings.TrimSpace(name)
	if name == "" {
		return ServerExecutionContext{}, false
	}

	if srv, ok := c.Servers[name]; ok {
		return ServerExecutionContext{
			Name: name,
			Args: slices.Clone(srv.Args),
			Env:  maps.Clone(srv.Env),
		}, true
	}

	return ServerExecutionContext{}, false
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

		// Expand args.
		for i, arg := range server.Args {
			server.Args[i] = os.ExpandEnv(arg)
		}

		// Expand env vars.
		for k, v := range server.Env {
			server.Env[k] = os.ExpandEnv(v)
		}

		cfg.Servers[name] = server
	}

	return cfg, nil
}

// SaveConfig writes the ExecutionContextConfig to disk as a TOML file,
// creating parent directories and setting secure file permissions.
func (c *ExecutionContextConfig) SaveConfig() error {
	path := c.filePath
	if path == "" {
		return fmt.Errorf("config file path not present")
	}

	// Ensure the directory exists before creating the file.
	if err := EnsureSecureDir(filepath.Dir(path)); err != nil {
		return fmt.Errorf("could not ensure execution context directory exists: %w", err)
	}

	// owner: rw-, group: ---, others: ---
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, perms.SecureFile)
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

// userSpecificDir returns a user-specific directory following XDG Base Directory Specification.
// It respects the given environment variable, falling back to homeDir/dir/AppDirName() if not set.
// The envVar must have XDG_ prefix to follow the specification.
func userSpecificDir(envVar string, dir string) (string, error) {
	envVar = strings.TrimSpace(envVar)
	// Validate that the environment variable follows XDG naming convention.
	if !strings.HasPrefix(envVar, "XDG_") {
		return "", fmt.Errorf(
			"environment variable '%s' does not follow XDG Base Directory Specification",
			envVar,
		)
	}

	// If the relevant environment variable is present and configured, then use it.
	if ch, ok := os.LookupEnv(envVar); ok && strings.TrimSpace(ch) != "" {
		home := strings.TrimSpace(ch)
		return filepath.Join(home, AppDirName()), nil
	}

	// Attempt to locate the home directory for the current user and return the path that follows the spec.
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	return filepath.Join(homeDir, dir, AppDirName()), nil
}

// UserSpecificConfigDir returns the directory that should be used to store any user-specific configuration.
// It adheres to the XDG Base Directory Specification, respecting the XDG_CONFIG_HOME environment variable.
// When XDG_CONFIG_HOME is not set, it defaults to ~/.config/mcpd/
// See: https://specifications.freedesktop.org/basedir-spec/latest/
func UserSpecificConfigDir() (string, error) {
	return userSpecificDir(EnvVarXDGConfigHome, ".config")
}

// UserSpecificCacheDir returns the directory that should be used to store any user-specific cache files.
// It adheres to the XDG Base Directory Specification, respecting the XDG_CACHE_HOME environment variable.
// When XDG_CACHE_HOME is not set, it defaults to ~/.cache/mcpd/
// See: https://specifications.freedesktop.org/basedir-spec/latest/
func UserSpecificCacheDir() (string, error) {
	return userSpecificDir(EnvVarXDGCacheHome, ".cache")
}

// AppDirName returns the name of the application directory for use in user-specific operations where data is being written.
func AppDirName() string {
	return "mcpd"
}

// EnsureSecureDir creates a directory with secure permissions if it doesn't exist.
// Used for directories containing sensitive data like execution context.
func EnsureSecureDir(path string) error {
	if err := os.MkdirAll(path, perms.SecureDir); err != nil {
		return fmt.Errorf("could not ensure secure directory exists for '%s': %w", path, err)
	}
	return nil
}

// EnsureRegularDir creates a directory with standard permissions if it doesn't exist.
// Used for cache directories, data directories, and documentation.
func EnsureRegularDir(path string) error {
	if err := os.MkdirAll(path, perms.RegularDir); err != nil {
		return fmt.Errorf("could not ensure regular directory exists for '%s': %w", path, err)
	}
	return nil
}
