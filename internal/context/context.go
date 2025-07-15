package context

import (
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
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
		cfg = NewExecutionContextConfig()
	}

	// Update the file path to allow saving later.
	cfg.filePath = path

	return &cfg, nil
}

// ExecutionContextConfig stores execution context data for all configured MCP servers.
type ExecutionContextConfig struct {
	Servers  map[string]ServerExecutionContext `toml:"servers"`
	filePath string                            `toml:"-"`
}

// NewExecutionContextConfig returns a newly initialized ExecutionContextConfig.
func NewExecutionContextConfig() ExecutionContextConfig {
	return ExecutionContextConfig{
		Servers: map[string]ServerExecutionContext{},
	}
}

func (c *ExecutionContextConfig) AddServer(ec ServerExecutionContext) error {
	c.Servers[ec.Name] = ec

	// TODO: Any kind of required validation.

	if err := c.saveConfig(); err != nil {
		return fmt.Errorf("failed to add server, error saving execution context config: %w", err)
	}

	return nil
}

func (c *ExecutionContextConfig) RemoveServer(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("server name cannot be empty")
	}

	updated := maps.Clone(c.Servers)
	if _, ok := updated[name]; !ok {
		return fmt.Errorf("server '%s' not found in execution context config", name)
	}
	c.Servers = updated
	delete(updated, name)

	// TODO: Any kind of required validation.

	if err := c.saveConfig(); err != nil {
		return fmt.Errorf("failed to remove server, error saving execution context config: %w", err)
	}

	return nil
}

func (c *ExecutionContextConfig) ListServers() map[string]ServerExecutionContext {
	if len(c.Servers) == 0 {
		return map[string]ServerExecutionContext{}
	}

	return maps.Clone(c.Servers)
}

// loadExecutionContextConfig loads a runtime execution context file from disk, using the specified path.
func loadExecutionContextConfig(path string) (ExecutionContextConfig, error) { // TODO: unexport
	var cfg ExecutionContextConfig

	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, fmt.Errorf("execution context file '%s' does not exist: %w", path, err)
		}

		return cfg, fmt.Errorf("could not stat execution context file '%s': %w", path, err)
	}

	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return cfg, fmt.Errorf("execution context file '%s' could not be parsed: %w", path, err)
	}

	// Manually set the name field for each ServerExecutionContext.
	for name, server := range cfg.Servers {
		server.Name = name
		cfg.Servers[name] = server
	}

	return cfg, nil
}

func (c *ExecutionContextConfig) saveConfig() error {
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
