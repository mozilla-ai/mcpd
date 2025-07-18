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
	// EnvVarConfigFile is the name of the environment variable that defines the config file location.
	EnvVarConfigFile = "MCPD_CONFIG_FILE"

	// EnvRuntimeFile is the name of the environment variable that defines the runtime config file location.
	EnvRuntimeFile = "MCPD_RUNTIME_FILE"

	// EnvVarLogPath is the name of the environment variable that defines the log file location.
	EnvVarLogPath = "MCPD_LOG_PATH"

	// EnvVarLogLevel is the name of the environment variable that defines the log level to use when a log file location is configured.
	EnvVarLogLevel = "MCPD_LOG_LEVEL"

	// DefaultConfigFile represents the default name of the config file.
	DefaultConfigFile = ".mcpd.toml"

	// DefaultRuntimeVarsFile represents the default name of the runtime execution context (secrets) config file.
	DefaultRuntimeVarsFile = "secrets.dev.toml"

	// DefaultLogPath represents the default log path (none).
	DefaultLogPath = ""

	// DefaultLogLevel represents the default log level to use when a log path is configured.
	DefaultLogLevel = "info"

	// FlagNameConfigFile is the name of the flag which represents the config file (.mcpd.toml).
	FlagNameConfigFile = "config-file"

	// FlagNameRuntimeFile is the name of the flag which represents the runtime execution context (secrets) file.
	FlagNameRuntimeFile = "runtime-file"

	// FlagNameLogPath is the name of the flag which represents the log file path.
	FlagNameLogPath = "log-path"

	// FlagNameLogLevel is the name of the flag which represents the log level.
	FlagNameLogLevel = "log-level"
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
