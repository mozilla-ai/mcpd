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
	for _, source := range []string{"home", "xdg", "runtime override", "short runtime override"} {
		t.Run(source, func(t *testing.T) {
			t.Chdir(t.TempDir())
			t.Setenv(files.EnvVarXDGConfigHome, "")
			t.Setenv(files.EnvVarXDGCacheHome, "")
			t.Setenv(flags.EnvRuntimeFile, "")
			if source == "xdg" {
				t.Setenv(files.EnvVarXDGConfigHome, filepath.Join(t.TempDir(), "custom-config"))
				t.Setenv(files.EnvVarXDGCacheHome, filepath.Join(t.TempDir(), "custom-cache"))
			}
			if source == "runtime override" {
				t.Setenv(flags.EnvRuntimeFile, filepath.Join(t.TempDir(), "custom-secrets.toml"))
			}

			if source == "short runtime override" {
				t.Setenv(flags.EnvRuntimeFile, "a")
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
			root, err = cmd.NewRootCmd(&cmd.RootCmd{BaseCmd: &internalcmd.BaseCmd{}})
			require.NoError(t, err)
			require.Equal(t, runtimePath, root.PersistentFlags().Lookup(flags.FlagNameRuntimeFile).DefValue)
			require.Contains(t, root.PersistentFlags().FlagUsages(), strconv.Quote(runtimePath))
			search, _, err = root.Find([]string{"search"})
			require.NoError(t, err)
			require.Equal(t, cachePath, search.Flags().Lookup("cache-dir").DefValue)
			require.Contains(t, search.Flags().FlagUsages(), strconv.Quote(cachePath))
		})
	}
}
