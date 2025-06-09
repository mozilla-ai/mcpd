package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mozilla-ai/mcpd-cli/v2/internal/context"
)

func NormalizeArgs(rawArgs []string) []string {
	var normalized []string
	for _, arg := range rawArgs {
		if strings.HasPrefix(arg, "--") {
			normalized = append(normalized, arg)
		} else {
			// Attach to previous arg if it's an --option expecting a value
			if len(normalized) > 0 {
				last := normalized[len(normalized)-1]
				if !strings.Contains(last, "=") {
					normalized[len(normalized)-1] = fmt.Sprintf("%s=%s", last, arg)
					continue
				}
			}
			// Or treat as positional
			normalized = append(normalized, arg)
		}
	}
	return normalized
}

func SetExecutionContextArgsEnv(serverName string, args []string, env map[string]string) error {
	cfgPath := defaultExecutionContextPath()

	// Load existing or create new config
	cfg := context.ExecutionContextConfig{}
	if _, err := os.Stat(cfgPath); err == nil {
		loaded, err := context.LoadExecutionContextConfig(cfgPath)
		if err != nil {
			return fmt.Errorf("failed to load execution context: %w", err)
		}
		cfg = loaded
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("error checking execution context path: %w", err)
	}

	if cfg.Servers == nil {
		cfg.Servers = make(map[string]context.ServerExecutionContext)
	}

	cfg.Servers[serverName] = context.ServerExecutionContext{
		Args: args,
		Env:  env,
	}

	if err := context.SaveExecutionContextConfig(cfgPath, cfg); err != nil {
		return fmt.Errorf("failed to save execution context: %w", err)
	}

	return nil
}

func defaultExecutionContextPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		// Fallback to current dir, last resort
		return "secrets.dev.toml"
	}
	return filepath.Join(home, ".mcpd", "secrets.dev.toml")
}
