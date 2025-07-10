package flags

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/pflag"

	"github.com/mozilla-ai/mcpd/v2/internal/context"
)

const (
	// Env vars
	EnvVarConfigFile = "MCPD_CONFIG_FILE"
	EnvRuntimeFile   = "MCPD_RUNTIME_FILE"
	EnvVarLogPath    = "MCPD_LOG_PATH"
	EnvVarLogLevel   = "MCPD_LOG_LEVEL"

	// Defaults
	DefaultConfigFile      = ".mcpd.toml"
	DefaultRuntimeVarsFile = "secrets.dev.toml"
	DefaultLogPath         = ""
	DefaultLogLevel        = "info"

	// Flag names
	FlagNameConfigFile  = "config-file"
	FlagNameRuntimeFile = "runtime-file"
	FlagNameLogPath     = "log-path"
	FlagNameLogLevel    = "log-level"
)

var (
	ConfigFile  string
	RuntimeFile string
	LogPath     string
	LogLevel    string
)

func InitFlags(fs *pflag.FlagSet) error {
	initConfigFile(fs)

	if err := initRuntimeVarsFile(fs); err != nil {
		return err
	}

	initLogger(fs)

	return nil
}

func initConfigFile(fs *pflag.FlagSet) {
	defaultConfigFile := strings.TrimSpace(os.Getenv(EnvVarConfigFile))
	if defaultConfigFile == "" {
		defaultConfigFile = DefaultConfigFile
	}
	fs.StringVar(&ConfigFile, FlagNameConfigFile, defaultConfigFile, "path to config file")
}

func initRuntimeVarsFile(fs *pflag.FlagSet) error {
	defaultRuntimeVarsFile := strings.TrimSpace(os.Getenv(EnvRuntimeFile))
	// When empty or matching the default value, resolve the correct folder.
	if defaultRuntimeVarsFile == "" || defaultRuntimeVarsFile == DefaultRuntimeVarsFile {
		dir, err := context.UserSpecificConfigDir()
		if err != nil {
			return fmt.Errorf("error configuring default value for runtime vars file: %w", err)
		}
		defaultRuntimeVarsFile = filepath.Join(dir, DefaultRuntimeVarsFile)
	}
	fs.StringVar(
		&RuntimeFile,
		FlagNameRuntimeFile,
		defaultRuntimeVarsFile,
		"path to runtime (execution context) file that contains env vars, and arguments for your MCP servers",
	)

	return nil
}

func initLogger(fs *pflag.FlagSet) {
	// NOTE: Consider splitting this into two separate functions if additional flags/logic are to be added.

	defaultLogPath := strings.TrimSpace(os.Getenv(EnvVarLogPath))
	if defaultLogPath == "" {
		defaultLogPath = DefaultLogPath
	}
	fs.StringVar(&LogPath, FlagNameLogPath, defaultLogPath, "log file path to use for log output")

	defaultLogLevel := strings.ToUpper(strings.TrimSpace(os.Getenv(EnvVarLogLevel)))
	if defaultLogLevel == "" {
		defaultLogLevel = DefaultLogLevel
	}
	fs.StringVar(&LogLevel, FlagNameLogLevel, defaultLogLevel, "log level for mcpd logs")
}
