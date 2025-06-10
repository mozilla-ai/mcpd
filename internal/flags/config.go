package flags

import (
	"os"
	"strings"

	"github.com/spf13/pflag"
)

const (
	// Env vars
	EnvVarConfigFile = "MCPD_CONFIG_FILE"
	EnvVarLogPath    = "MCPD_LOG_PATH"
	EnvVarLogLevel   = "MCPD_LOG_LEVEL"

	// Defaults
	DefaultConfigFile = ".mcpd.toml"
	DefaultLogPath    = ""
	DefaultLogLevel   = "info"

	// Flag names
	FlagNameConfigFile = "config-file"
	FlagNameLogPath    = "log-path"
	FlagNameLogLevel   = "log-level"
)

var (
	ConfigFile string
	LogPath    string
	LogLevel   string
)

func InitFlags(fs *pflag.FlagSet) {
	initConfigFile(fs)
	initLogger(fs)
}

func initConfigFile(fs *pflag.FlagSet) {
	defaultConfigFile := strings.TrimSpace(os.Getenv(EnvVarConfigFile))
	if defaultConfigFile == "" {
		defaultConfigFile = DefaultConfigFile
	}
	fs.StringVar(&ConfigFile, FlagNameConfigFile, defaultConfigFile, "path to config file")
}

func initLogger(fs *pflag.FlagSet) {
	defaultLogPath := strings.TrimSpace(os.Getenv(EnvVarLogPath))
	if defaultLogPath == "" {
		LogPath = DefaultLogPath
	}
	fs.StringVar(&LogPath, FlagNameLogPath, defaultLogPath, "path to generated log file")

	defaultLogLevel := strings.ToLower(os.Getenv(EnvVarLogLevel))
	if defaultLogLevel == "" {
		LogLevel = DefaultLogLevel
	}
	fs.StringVar(&LogLevel, FlagNameLogLevel, defaultLogLevel, "log level for mcpd logs")
}
