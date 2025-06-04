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
	if ConfigFile == "" {
		if env := strings.TrimSpace(os.Getenv(EnvVarConfigFile)); env != "" {
			ConfigFile = env
		} else {
			ConfigFile = DefaultConfigFile
		}
	}
	fs.StringVar(&ConfigFile, FlagNameConfigFile, ConfigFile, "path to config file")
}

func initLogger(fs *pflag.FlagSet) {
	if LogPath == "" {
		if env := strings.TrimSpace(os.Getenv(EnvVarLogPath)); env != "" {
			LogPath = env
		} else {
			LogPath = DefaultLogPath
		}
	}
	fs.StringVar(&LogPath, FlagNameLogPath, LogPath, "path to generated log file")

	if LogLevel == "" {
		if env := strings.TrimSpace(os.Getenv(EnvVarLogLevel)); env != "" {
			LogLevel = strings.ToLower(env)
		} else {
			LogLevel = DefaultLogLevel
		}
	}
	fs.StringVar(&LogLevel, FlagNameLogLevel, LogLevel, "log level for mcpd logs")
}
