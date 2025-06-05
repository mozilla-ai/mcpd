package flags

import (
	"os"

	"github.com/spf13/pflag"
)

const (
	EnvVarConfigFile   = "MCPD_CONFIG_FILE"
	DefaultConfigFile  = ".mcpd.toml"
	FlagNameConfigFile = "config-file"
)

var ConfigFile string

func InitFlags(fs *pflag.FlagSet) {
	defaultVal := os.Getenv(EnvVarConfigFile)
	if defaultVal == "" {
		defaultVal = DefaultConfigFile
	}

	fs.StringVar(&ConfigFile, FlagNameConfigFile, defaultVal, "path to config file")
}
