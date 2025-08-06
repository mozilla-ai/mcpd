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
)

// EnvVarXDGConfigHome is the XDG Base Directory env var name.
const EnvVarXDGConfigHome = "XDG_CONFIG_HOME"

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

// loadExecutionContextConfig loads a runtime execution context file from disk, using the specified path.
func loadExecutionContextConfig(path string) (*ExecutionContextConfig, error) { // TODO: unexport
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

	// Manually set the name field for each ServerExecutionContext.
	for name, server := range cfg.Servers {
		server.Name = name
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

	// Ensure the directory exists before creating the file...
	// owner: rwx, group: r--, others: ---
	if err := os.MkdirAll(filepath.Dir(path), 0o740); err != nil {
		return fmt.Errorf("could not ensure execution context directory exists for '%s': %w", path, err)
	}

	// owner: rw-, group: ---, others: ---
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o600)
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

// UserSpecificConfigDir returns the directory that should be used to store any user-specific configuration.
// It adheres to the XDG Base Directory Specification, respecting the XDG_CONFIG_HOME environment variable.
// When XDG_CONFIG_HOME is not set, it defaults to ~/.config/mcpd/
// See: https://specifications.freedesktop.org/basedir-spec/latest/
func UserSpecificConfigDir() (string, error) {
	// If the relevant environment variable is present and configured, then use it.
	if ch, ok := os.LookupEnv(EnvVarXDGConfigHome); ok && strings.TrimSpace(ch) != "" {
		home := strings.TrimSpace(ch)
		return filepath.Join(home, AppDirName()), nil
	}

	// Attempt to locate the home directory for the current user and return the path that follows the spec.
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	return filepath.Join(homeDir, ".config", AppDirName()), nil
}

// AppDirName returns the name of the application directory for use in user-specific operations where data is being written.
func AppDirName() string {
	return "mcpd"
}
