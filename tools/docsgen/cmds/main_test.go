//go:build docsgen_cli
// +build docsgen_cli

package main

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mozilla-ai/mcpd/cmd"
	internalcmd "github.com/mozilla-ai/mcpd/internal/cmd"
	"github.com/mozilla-ai/mcpd/internal/files"
	"github.com/mozilla-ai/mcpd/internal/flags"
)

func TestGeneratedDocsUsePortablePaths(t *testing.T) {
	for _, source := range []string{
		"home",
		"xdg",
		"runtime override",
		"short runtime override",
		"config override",
		"log override",
		"all overrides",
	} {
		t.Run(source, func(t *testing.T) {
			t.Chdir(t.TempDir())
			t.Setenv(files.EnvVarXDGConfigHome, "")
			t.Setenv(files.EnvVarXDGCacheHome, "")
			t.Setenv(flags.EnvRuntimeFile, "")
			t.Setenv(flags.EnvVarConfigFile, "")
			t.Setenv(flags.EnvVarLogPath, "")
			if source == "xdg" {
				t.Setenv(files.EnvVarXDGConfigHome, filepath.Join(t.TempDir(), "custom-config"))
				t.Setenv(files.EnvVarXDGCacheHome, filepath.Join(t.TempDir(), "custom-cache"))
			}
			if source == "runtime override" || source == "all overrides" {
				t.Setenv(flags.EnvRuntimeFile, filepath.Join(t.TempDir(), "custom-secrets.toml"))
			}

			if source == "short runtime override" {
				t.Setenv(flags.EnvRuntimeFile, "a")
			}

			if source == "config override" || source == "all overrides" {
				t.Setenv(flags.EnvVarConfigFile, filepath.Join(t.TempDir(), "custom-config.toml"))
			}
			if source == "log override" || source == "all overrides" {
				t.Setenv(flags.EnvVarLogPath, filepath.Join(t.TempDir(), "custom.log"))
			}

			root, err := cmd.NewRootCmd(&cmd.RootCmd{BaseCmd: &internalcmd.BaseCmd{}})
			require.NoError(t, err)
			runtimePath := root.PersistentFlags().Lookup(flags.FlagNameRuntimeFile).DefValue
			require.NotEmpty(t, runtimePath)

			search, _, err := root.Find([]string{"search"})
			require.NoError(t, err)
			cachePath := search.Flags().Lookup("cache-dir").DefValue
			require.NotEmpty(t, cachePath)

			runtimeEnv := os.Getenv(flags.EnvRuntimeFile)
			configEnv := os.Getenv(flags.EnvVarConfigFile)
			logEnv := os.Getenv(flags.EnvVarLogPath)
			main()

			pages, err := filepath.Glob("docs/commands/*.md")
			require.NoError(t, err)
			require.NotEmpty(t, pages)
			for _, page := range pages {
				content, err := os.ReadFile(page)
				require.NoError(t, err)
				require.NotContains(t, string(content), strconv.Quote(runtimePath), page)
				require.NotContains(t, string(content), strconv.Quote(cachePath), page)
				require.Contains(t, string(content), `default "~/.config/mcpd/secrets.dev.toml"`, page)
				require.Contains(t, string(content), `default ".mcpd.toml"`, page)
				if configEnv != "" {
					require.NotContains(t, string(content), strconv.Quote(configEnv), page)
				}
				if logEnv != "" {
					require.NotContains(t, string(content), strconv.Quote(logEnv), page)
				}
			}
			for _, name := range []string{"list", "set", "clear", "remove"} {
				content, err := os.ReadFile("docs/commands/mcpd_config_volumes_" + name + ".md")
				require.NoError(t, err)
				require.Contains(t, string(content), "(e.g. ~/.config/mcpd/secrets.dev.toml)")
			}

			for _, name := range []string{"mcpd_add", "mcpd_search", "mcpd_config_tools_list", "mcpd_config_tools_set"} {
				content, err := os.ReadFile("docs/commands/" + name + ".md")
				require.NoError(t, err)
				require.Contains(t, string(content), `default "~/.cache/mcpd/registries"`)
			}

			// Normal CLI help still resolves the caller's runtime path.
			t.Setenv(flags.EnvRuntimeFile, runtimeEnv)
			t.Setenv(flags.EnvVarConfigFile, configEnv)
			t.Setenv(flags.EnvVarLogPath, logEnv)
			root, err = cmd.NewRootCmd(&cmd.RootCmd{BaseCmd: &internalcmd.BaseCmd{}})
			require.NoError(t, err)
			require.Equal(t, runtimePath, root.PersistentFlags().Lookup(flags.FlagNameRuntimeFile).DefValue)
			require.Contains(t, root.PersistentFlags().FlagUsages(), strconv.Quote(runtimePath))
			if configEnv != "" {
				require.Equal(t, configEnv, root.PersistentFlags().Lookup(flags.FlagNameConfigFile).DefValue)
				require.Contains(t, root.PersistentFlags().FlagUsages(), strconv.Quote(configEnv))
			}
			if logEnv != "" {
				require.Equal(t, logEnv, root.PersistentFlags().Lookup(flags.FlagNameLogPath).DefValue)
				require.Contains(t, root.PersistentFlags().FlagUsages(), strconv.Quote(logEnv))
			}
			search, _, err = root.Find([]string{"search"})
			require.NoError(t, err)
			require.Equal(t, cachePath, search.Flags().Lookup("cache-dir").DefValue)
			require.Contains(t, search.Flags().FlagUsages(), strconv.Quote(cachePath))
		})
	}
}
