package flags

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"

	"github.com/mozilla-ai/mcpd/v2/internal/context"
)

func TestConfig_InitConfigFile_EnvVars(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected string
	}{
		{
			name:     "env var value with extra white space",
			value:    "  /custom/path/config.toml  ",
			expected: "/custom/path/config.toml",
		},
		{
			name:     "env var missing",
			value:    "", // Implementation uses os.Getenv which returns an empty string when missing.
			expected: DefaultConfigFile,
		},
		{
			name:     "env var only white space",
			value:    "   ",
			expected: DefaultConfigFile,
		},
		{
			name:     "env var empty string",
			value:    "",
			expected: DefaultConfigFile,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(EnvVarConfigFile, tc.value)
			t.Cleanup(func() {
				// Reset global variable
				ConfigFile = ""
			})

			fs := pflag.NewFlagSet("test", pflag.ContinueOnError)

			// Call init func.
			initConfigFile(fs)

			require.Equal(t, tc.expected, ConfigFile)
			flag := fs.Lookup(FlagNameConfigFile)
			require.NotNil(t, flag)
			require.Equal(t, tc.expected, flag.Value.String())
		})
	}
}

func TestConfig_InitRuntimeVarsFile_EnvVars(t *testing.T) {
	home, err := os.UserHomeDir()
	require.NoError(t, err)

	tests := []struct {
		name     string
		value    string
		expected string
	}{
		{
			name:     "env var value with extra white space",
			value:    "  /custom/path/config.toml  ",
			expected: "/custom/path/config.toml",
		},
		{
			name:     "env var missing",
			value:    "", // Implementation uses os.Getenv which returns an empty string when missing.
			expected: filepath.Join(home, ".config", "mcpd", DefaultRuntimeVarsFile),
		},
		{
			name:     "env var only white space",
			value:    "   ",
			expected: filepath.Join(home, ".config", "mcpd", DefaultRuntimeVarsFile),
		},
		{
			name:     "env var empty string",
			value:    "",
			expected: filepath.Join(home, ".config", "mcpd", DefaultRuntimeVarsFile),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(EnvRuntimeFile, tc.value)
			t.Cleanup(func() {
				// Reset global variable
				RuntimeFile = ""
			})

			fs := pflag.NewFlagSet("test", pflag.ContinueOnError)

			// Call init func.
			err := initRuntimeVarsFile(fs)
			require.NoError(t, err)

			require.Equal(t, tc.expected, RuntimeFile)
			flag := fs.Lookup(FlagNameRuntimeFile)
			require.NotNil(t, flag)
			require.Equal(t, tc.expected, flag.Value.String())
		})
	}
}

func TestConfig_InitLogger_EnvVars(t *testing.T) {
	tests := []struct {
		name          string
		logPathValue  string
		logLevelValue string
		expectedPath  string
		expectedLevel string
	}{
		{
			name:          "both env vars set with extra whitespace",
			logPathValue:  "  /var/log/mcpd.log  ",
			logLevelValue: "  debug  ",
			expectedPath:  "/var/log/mcpd.log",
			expectedLevel: "DEBUG",
		},
		{
			name:          "env vars set to only whitespace",
			logPathValue:  "   ",
			logLevelValue: "   ",
			expectedPath:  DefaultLogPath,
			expectedLevel: DefaultLogLevel,
		},
		{
			name:          "no env vars set",
			logPathValue:  "", // Implementation uses os.Getenv which returns an empty string when missing.
			logLevelValue: "", // Implementation uses os.Getenv which returns an empty string when missing.
			expectedPath:  DefaultLogPath,
			expectedLevel: DefaultLogLevel,
		},
		{
			name:          "env var empty strings",
			logPathValue:  "",
			logLevelValue: "",
			expectedPath:  DefaultLogPath,
			expectedLevel: DefaultLogLevel,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(EnvVarLogPath, tc.logPathValue)
			t.Setenv(EnvVarLogLevel, tc.logLevelValue)
			t.Cleanup(func() {
				LogPath = ""
				LogLevel = ""
			})

			fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
			initLogger(fs)

			require.Equal(t, tc.expectedPath, LogPath)
			require.Equal(t, tc.expectedLevel, LogLevel)

			pathFlag := fs.Lookup(FlagNameLogPath)
			require.NotNil(t, pathFlag)
			require.Equal(t, tc.expectedPath, pathFlag.Value.String())

			levelFlag := fs.Lookup(FlagNameLogLevel)
			require.NotNil(t, levelFlag)
			require.Equal(t, tc.expectedLevel, levelFlag.Value.String())
		})
	}
}

func TestConfig_ConfigFile_Precedence(t *testing.T) {
	tests := []struct {
		name        string
		envValue    string
		cmdLineArgs []string
		expected    string
	}{
		{
			name:        "flag takes precedence over everything",
			envValue:    "/env/path/config.toml",
			cmdLineArgs: []string{"--" + FlagNameConfigFile, "/flag/path/config.toml"},
			expected:    "/flag/path/config.toml",
		},
		{
			name:        "env var takes precedence over default value",
			envValue:    "/env/only/path.toml",
			cmdLineArgs: nil,
			expected:    "/env/only/path.toml",
		},
		{
			name:        "default used when no flag and no env var set",
			envValue:    "",
			cmdLineArgs: nil,
			expected:    DefaultConfigFile,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Cleanup(func() {
				ConfigFile = ""
			})

			t.Setenv(EnvVarConfigFile, tc.envValue)

			fs := pflag.NewFlagSet("test", pflag.ContinueOnError)

			initConfigFile(fs)
			err := fs.Parse(tc.cmdLineArgs)

			require.NoError(t, err)
			require.Equal(t, tc.expected, ConfigFile)
			flag := fs.Lookup(FlagNameConfigFile)
			require.NotNil(t, flag)
			require.Equal(t, tc.expected, flag.Value.String())
		})
	}
}

func TestConfig_RuntimeVarsFile_Precedence(t *testing.T) {
	tests := []struct {
		name        string
		envValue    string
		cmdLineArgs []string
		expected    string
	}{
		{
			name:        "flag takes precedence over everything",
			envValue:    "/env/path/runtime.toml",
			cmdLineArgs: []string{"--" + FlagNameRuntimeFile, "/flag/path/runtime.toml"},
			expected:    "/flag/path/runtime.toml",
		},
		{
			name:        "env var takes precedence over resolved default path",
			envValue:    "/env/only/runtime.toml",
			cmdLineArgs: nil,
			expected:    "/env/only/runtime.toml",
		},
		{
			name:        "default used when no flag and no env var set",
			envValue:    "",
			cmdLineArgs: nil,
			expected: func() string {
				dir, err := context.UserSpecificConfigDir()
				require.NoError(t, err)
				return filepath.Join(dir, DefaultRuntimeVarsFile)
			}(),
		},
		{
			name:        "env var set to default triggers resolved path",
			envValue:    DefaultRuntimeVarsFile,
			cmdLineArgs: nil,
			expected: func() string {
				dir, err := context.UserSpecificConfigDir()
				require.NoError(t, err)
				return filepath.Join(dir, DefaultRuntimeVarsFile)
			}(),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			originalXDGConfigHome := os.Getenv(context.EnvVarXDGConfigHome)
			t.Cleanup(func() {
				// Reset flag vars.
				RuntimeFile = ""

				// Reset env state
				require.NoError(t, os.Setenv(context.EnvVarXDGConfigHome, originalXDGConfigHome))
			})
			// Clear XDG_CONFIG_HOME to ensure it cannot cause side effects in the test results.
			t.Setenv(context.EnvVarXDGConfigHome, "")
			t.Setenv(EnvRuntimeFile, tc.envValue)

			fs := pflag.NewFlagSet("test", pflag.ContinueOnError)

			err := initRuntimeVarsFile(fs)
			require.NoError(t, err)

			err = fs.Parse(tc.cmdLineArgs)
			require.NoError(t, err)

			require.Equal(t, tc.expected, RuntimeFile)
			flag := fs.Lookup(FlagNameRuntimeFile)
			require.NotNil(t, flag)
			require.Equal(t, tc.expected, flag.Value.String())
		})
	}
}

func TestConfig_LoggerFlags_Precedence(t *testing.T) {
	tests := []struct {
		name          string
		envLogPath    string
		envLogLevel   string
		cmdLineArgs   []string
		expectedPath  string
		expectedLevel string
	}{
		{
			name:          "flags take precedence over env vars",
			envLogPath:    "/env/log/path.log",
			envLogLevel:   "WARN",
			cmdLineArgs:   []string{"--" + FlagNameLogPath, "/flag/log/path.log", "--" + FlagNameLogLevel, "DEBUG"},
			expectedPath:  "/flag/log/path.log",
			expectedLevel: "DEBUG",
		},
		{
			name:          "env vars used when flags not set",
			envLogPath:    "/env/only/path.log",
			envLogLevel:   "INFO",
			cmdLineArgs:   nil,
			expectedPath:  "/env/only/path.log",
			expectedLevel: "INFO",
		},
		{
			name:          "defaults used when neither flags nor env vars set",
			envLogPath:    "",
			envLogLevel:   "",
			cmdLineArgs:   nil,
			expectedPath:  DefaultLogPath,
			expectedLevel: DefaultLogLevel,
		},
		{
			name:          "env var whitespace triggers default fallback",
			envLogPath:    "   ",
			envLogLevel:   "   ",
			cmdLineArgs:   nil,
			expectedPath:  DefaultLogPath,
			expectedLevel: DefaultLogLevel,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			originalXDGConfigHome := os.Getenv(context.EnvVarXDGConfigHome)
			t.Cleanup(func() {
				// Reset flag vars.
				ConfigFile = ""
				RuntimeFile = ""
				LogPath = ""
				LogLevel = ""

				// Reset env state.
				require.NoError(t, os.Setenv(context.EnvVarXDGConfigHome, originalXDGConfigHome))
			})
			// Clear XDG_CONFIG_HOME to ensure it cannot cause side effects in the test results.
			t.Setenv(context.EnvVarXDGConfigHome, "")

			t.Setenv(EnvVarLogPath, tc.envLogPath)
			t.Setenv(EnvVarLogLevel, tc.envLogLevel)

			fs := pflag.NewFlagSet("test", pflag.ContinueOnError)

			initLogger(fs)
			err := fs.Parse(tc.cmdLineArgs)

			require.NoError(t, err)
			require.Equal(t, tc.expectedPath, LogPath)
			require.Equal(t, tc.expectedLevel, LogLevel)

			pathFlag := fs.Lookup(FlagNameLogPath)
			require.NotNil(t, pathFlag)
			require.Equal(t, tc.expectedPath, pathFlag.Value.String())

			levelFlag := fs.Lookup(FlagNameLogLevel)
			require.NotNil(t, levelFlag)
			require.Equal(t, tc.expectedLevel, levelFlag.Value.String())
		})
	}
}

func TestConfig_InitFlags(t *testing.T) {
	home, err := os.UserHomeDir()
	require.NoError(t, err)

	tests := []struct {
		name            string
		envConfig       string
		envRuntime      string
		envLogPath      string
		envLogLevel     string
		cmdLineArgs     []string
		expectedConfig  string
		expectedRuntime string
		expectedLogPath string
		expectedLogLvl  string
	}{
		{
			name:        "all flags take precedence over env and defaults",
			envConfig:   "/env/config.toml",
			envRuntime:  "/env/runtime.toml",
			envLogPath:  "/env/log/path.log",
			envLogLevel: "warn",
			cmdLineArgs: []string{
				"--" + FlagNameConfigFile, "/flag/config.toml",
				"--" + FlagNameRuntimeFile, "/flag/runtime.toml",
				"--" + FlagNameLogPath, "/flag/log.log",
				"--" + FlagNameLogLevel, "debug",
			},
			expectedConfig:  "/flag/config.toml",
			expectedRuntime: "/flag/runtime.toml",
			expectedLogPath: "/flag/log.log",
			expectedLogLvl:  "debug",
		},
		{
			name:            "env vars used when flags not set",
			envConfig:       "/env/only/config.toml",
			envRuntime:      "/env/only/runtime.toml",
			envLogPath:      "/env/only/log.log",
			envLogLevel:     "INFO",
			cmdLineArgs:     nil,
			expectedConfig:  "/env/only/config.toml",
			expectedRuntime: "/env/only/runtime.toml",
			expectedLogPath: "/env/only/log.log",
			expectedLogLvl:  "INFO",
		},
		{
			name:            "default values used when nothing set",
			envConfig:       "",
			envRuntime:      "",
			envLogPath:      "",
			envLogLevel:     "",
			cmdLineArgs:     nil,
			expectedConfig:  DefaultConfigFile,
			expectedRuntime: filepath.Join(home, ".config", "mcpd", DefaultRuntimeVarsFile),
			expectedLogPath: DefaultLogPath,
			expectedLogLvl:  DefaultLogLevel,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(EnvVarConfigFile, tc.envConfig)
			t.Setenv(EnvRuntimeFile, tc.envRuntime)
			t.Setenv(EnvVarLogPath, tc.envLogPath)
			t.Setenv(EnvVarLogLevel, tc.envLogLevel)

			t.Cleanup(func() {
				ConfigFile = ""
				RuntimeFile = ""
				LogPath = ""
				LogLevel = ""
			})

			fs := pflag.NewFlagSet("test", pflag.ContinueOnError)

			err := InitFlags(fs)
			require.NoError(t, err)

			err = fs.Parse(tc.cmdLineArgs)
			require.NoError(t, err)

			require.Equal(t, tc.expectedConfig, ConfigFile)
			require.Equal(t, tc.expectedRuntime, RuntimeFile)
			require.Equal(t, tc.expectedLogPath, LogPath)
			require.Equal(t, tc.expectedLogLvl, LogLevel)

			require.Equal(t, tc.expectedConfig, fs.Lookup(FlagNameConfigFile).Value.String())
			require.Equal(t, tc.expectedRuntime, fs.Lookup(FlagNameRuntimeFile).Value.String())
			require.Equal(t, tc.expectedLogPath, fs.Lookup(FlagNameLogPath).Value.String())
			require.Equal(t, tc.expectedLogLvl, fs.Lookup(FlagNameLogLevel).Value.String())
		})
	}
}
