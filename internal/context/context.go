package context

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

// ServerExecutionContext stores execution context data for an MCP server.
type ServerExecutionContext struct {
	Args []string          `toml:"args,omitempty"`
	Env  map[string]string `toml:"env,omitempty"`
}

// ExecutionContextConfig stores execution context data for all configured MCP servers.
type ExecutionContextConfig struct {
	Servers map[string]ServerExecutionContext `toml:"servers"`
}

// LoadExecutionContextConfig loads a secrets/dev context file from disk.
func LoadExecutionContextConfig(path string) (ExecutionContextConfig, error) {
	var cfg ExecutionContextConfig

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return cfg, fmt.Errorf("execution context file '%s' does not exist", path)
	}

	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return cfg, fmt.Errorf("execution context file '%s' could not be parsed: %w", path, err)
	}

	return cfg, nil
}

func SaveExecutionContextConfig(path string, cfg ExecutionContextConfig) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("could not create file '%s': %w", path, err)
	}
	defer f.Close()

	encoder := toml.NewEncoder(f)
	if err := encoder.Encode(cfg); err != nil {
		return fmt.Errorf("could not encode execution context to file '%s': %w", path, err)
	}

	return nil
}
