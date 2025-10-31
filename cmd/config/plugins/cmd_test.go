package plugins_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mozilla-ai/mcpd/v2/cmd/config/plugins"
	"github.com/mozilla-ai/mcpd/v2/internal/cmd"
)

func TestPlugins_NewCmd_Success(t *testing.T) {
	t.Parallel()

	baseCmd := &cmd.BaseCmd{}
	cobraCmd, err := plugins.NewCmd(baseCmd)

	require.NoError(t, err)
	require.NotNil(t, cobraCmd)
	require.Equal(t, "plugins", cobraCmd.Use)

	// Verify all subcommands are registered.
	commands := cobraCmd.Commands()
	require.Len(t, commands, 7)

	// Verify expected subcommands are present (Cobra sorts alphabetically).
	expectedCmds := map[string]bool{
		"add":      true,
		"get":      true,
		"list":     true,
		"move":     true,
		"remove":   true,
		"set":      true,
		"validate": true,
	}

	for _, command := range commands {
		// Extract command name (first word) from Use field to handle cases like "add <plugin-name>".
		cmdName := strings.Fields(command.Use)[0]
		require.True(t, expectedCmds[cmdName], "unexpected command: %s", command.Use)
		delete(expectedCmds, cmdName)
	}

	require.Empty(t, expectedCmds, "missing expected commands")
}
