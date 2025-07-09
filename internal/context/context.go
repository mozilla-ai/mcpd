package context

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// EnvVarXDGConfigHome is the XDG Base Directory env var name.
const EnvVarXDGConfigHome = "XDG_CONFIG_HOME"

// ServerExecutionContext stores execution context data for an MCP server.
type ServerExecutionContext struct {
	Args []string          `toml:"args,omitempty"`
	Env  map[string]string `toml:"env,omitempty"`
}

// ExecutionContextConfig stores execution context data for all configured MCP servers.
type ExecutionContextConfig struct {
	Servers map[string]ServerExecutionContext `toml:"servers"`
}

// NewExecutionContextConfig returns a newly initialized ExecutionContextConfig.
func NewExecutionContextConfig() ExecutionContextConfig {
	return ExecutionContextConfig{
		Servers: map[string]ServerExecutionContext{},
	}
}

// LoadOrInitExecutionContext loads a runtime execution context file from disk, using the specified path.
// If the file does not exist a newly initialized ExecutionContextConfig is returned.
func LoadOrInitExecutionContext(path string) (ExecutionContextConfig, error) {
	cfg, err := LoadExecutionContextConfig(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// Return a newly initialized execution context config since the file doesn't exist.
			return NewExecutionContextConfig(), nil
		}
		return ExecutionContextConfig{}, fmt.Errorf("failed to load execution context config: %w", err)
	}
	return cfg, nil
}

// LoadExecutionContextConfig loads a runtime execution context file from disk, using the specified path.
func LoadExecutionContextConfig(path string) (ExecutionContextConfig, error) {
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

	return cfg, nil
}

// SaveExecutionContextConfig saves a runtime execution context file to disk, using the specified path.
func SaveExecutionContextConfig(path string, cfg ExecutionContextConfig) (err error) {
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
	if err := encoder.Encode(cfg); err != nil {
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
