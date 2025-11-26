package plugins

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mozilla-ai/mcpd/v2/internal/cmd"
	cmdopts "github.com/mozilla-ai/mcpd/v2/internal/cmd/options"
	"github.com/mozilla-ai/mcpd/v2/internal/config"
)

func TestNewMoveCmd(t *testing.T) {
	t.Parallel()

	base := &cmd.BaseCmd{}
	c, err := NewMoveCmd(base)
	require.NoError(t, err)
	require.NotNil(t, c)

	require.Equal(t, "move", c.Use)
	require.Contains(t, c.Short, "Move a plugin")
	require.NotNil(t, c.RunE)
}

func TestMoveCmd_ToCategory(t *testing.T) {
	t.Parallel()

	loader := newMockLoaderFromFile(t)

	// Setup: Add a plugin to authentication category.
	cfgModifier, err := loader.Load("ignored")
	require.NoError(t, err)
	cfg := cfgModifier.(*config.Config)

	_, err = cfg.UpsertPlugin(config.CategoryAuthentication, config.PluginEntry{
		Name:  "jwt-auth",
		Flows: []config.Flow{config.FlowRequest},
	})
	require.NoError(t, err)

	base := &cmd.BaseCmd{}
	moveCmd, err := NewMoveCmd(base, cmdopts.WithConfigLoader(loader))
	require.NoError(t, err)

	err = moveCmd.Flags().Set(flagCategory, "authentication")
	require.NoError(t, err)
	err = moveCmd.Flags().Set(flagName, "jwt-auth")
	require.NoError(t, err)
	err = moveCmd.Flags().Set(flagToCategory, "audit")
	require.NoError(t, err)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	moveCmd.SetOut(&stdout)
	moveCmd.SetErr(&stderr)

	err = executeCmd(t, moveCmd, []string{})
	require.NoError(t, err)
	require.Empty(t, stderr.String())
	require.Contains(t, stdout.String(), "Plugin 'jwt-auth' moved")

	// Verify plugin was moved.
	cfgModifier, err = loader.Load("ignored")
	require.NoError(t, err)
	cfg = cfgModifier.(*config.Config)

	authPlugins := cfg.Plugins.ListPlugins(config.CategoryAuthentication)
	require.Empty(t, authPlugins)

	auditPlugins := cfg.Plugins.ListPlugins(config.CategoryAudit)
	require.Len(t, auditPlugins, 1)
	require.Equal(t, "jwt-auth", auditPlugins[0].Name)
}

func TestMoveCmd_Before(t *testing.T) {
	t.Parallel()

	loader := newMockLoaderFromFile(t)

	// Setup: Add two plugins to authentication category.
	cfgModifier, err := loader.Load("ignored")
	require.NoError(t, err)
	cfg := cfgModifier.(*config.Config)

	_, err = cfg.UpsertPlugin(config.CategoryAuthentication, config.PluginEntry{
		Name:  "plugin-a",
		Flows: []config.Flow{config.FlowRequest},
	})
	require.NoError(t, err)
	_, err = cfg.UpsertPlugin(config.CategoryAuthentication, config.PluginEntry{
		Name:  "plugin-b",
		Flows: []config.Flow{config.FlowRequest},
	})
	require.NoError(t, err)

	base := &cmd.BaseCmd{}
	moveCmd, err := NewMoveCmd(base, cmdopts.WithConfigLoader(loader))
	require.NoError(t, err)

	err = moveCmd.Flags().Set(flagCategory, "authentication")
	require.NoError(t, err)
	err = moveCmd.Flags().Set(flagName, "plugin-b")
	require.NoError(t, err)
	err = moveCmd.Flags().Set(flagBefore, "plugin-a")
	require.NoError(t, err)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	moveCmd.SetOut(&stdout)
	moveCmd.SetErr(&stderr)

	err = executeCmd(t, moveCmd, []string{})
	require.NoError(t, err)
	require.Empty(t, stderr.String())
	require.Contains(t, stdout.String(), "Plugin 'plugin-b' moved")

	// Verify order: plugin-b should now be before plugin-a.
	cfgModifier, err = loader.Load("ignored")
	require.NoError(t, err)
	cfg = cfgModifier.(*config.Config)

	authPlugins := cfg.Plugins.ListPlugins(config.CategoryAuthentication)
	require.Len(t, authPlugins, 2)
	require.Equal(t, "plugin-b", authPlugins[0].Name)
	require.Equal(t, "plugin-a", authPlugins[1].Name)
}

func TestMoveCmd_InvalidFlagCombinations(t *testing.T) {
	t.Parallel()

	loader := newMockLoaderFromFile(t)

	// Setup: Add two plugins to authentication category.
	cfgModifier, err := loader.Load("ignored")
	require.NoError(t, err)
	cfg := cfgModifier.(*config.Config)
	_, err = cfg.UpsertPlugin(config.CategoryAuthentication, config.PluginEntry{
		Name:  "plugin-a",
		Flows: []config.Flow{config.FlowRequest},
	})
	require.NoError(t, err)
	_, err = cfg.UpsertPlugin(config.CategoryAuthentication, config.PluginEntry{
		Name:  "plugin-b",
		Flows: []config.Flow{config.FlowRequest},
	})
	require.NoError(t, err)

	tests := []struct {
		name  string
		args  map[string]string
		error string
	}{
		{
			name:  "before and after",
			args:  map[string]string{flagBefore: "plugin-b", flagAfter: "plugin-c"},
			error: "if any flags in the group [after before] are set none of the others can be; [after before] were all set",
		},
		{
			name:  "before and position",
			args:  map[string]string{flagBefore: "plugin-b", flagPosition: "2"},
			error: "if any flags in the group [before position] are set none of the others can be; [before position] were all set",
		},
		{
			name:  "after and position",
			args:  map[string]string{flagAfter: "plugin-b", flagPosition: "3"},
			error: "if any flags in the group [after position] are set none of the others can be; [after position] were all set",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			base := &cmd.BaseCmd{}
			moveCmd, err := NewMoveCmd(base, cmdopts.WithConfigLoader(loader))
			require.NoError(t, err)

			name := "plugin-a"
			category := "authentication"
			argFmt := "--%s=%s"

			// Set cmd line args and flags (Cobra checks raw args, our validation checks flags).
			err = moveCmd.Flags().Set(flagCategory, category)
			require.NoError(t, err)
			err = moveCmd.Flags().Set(flagName, name)
			require.NoError(t, err)

			args := make([]string, 0, len(tc.args)+2) // Manually add category and name
			args = append(args, fmt.Sprintf(argFmt, flagCategory, category))
			args = append(args, fmt.Sprintf(argFmt, flagName, name))
			for k, v := range tc.args {
				args = append(args, fmt.Sprintf(argFmt, k, v))
			}

			moveCmd.SetArgs(args)
			err = moveCmd.Execute()
			require.Error(t, err)
			require.EqualError(t, err, tc.error)
		})
	}
}

func TestMoveCmd_InvalidPositionValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		position string
		error    string
	}{
		{
			name:     "position zero",
			position: "0",
			error:    "invalid 'position' flag value (must be a positive integer or -1 for end)",
		},
		{
			name:     "position negative",
			position: "-2",
			error:    "invalid 'position' flag value (must be a positive integer or -1 for end)",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			loader := newMockLoaderFromFile(t)

			base := &cmd.BaseCmd{}
			moveCmd, err := NewMoveCmd(base, cmdopts.WithConfigLoader(loader))
			require.NoError(t, err)

			err = moveCmd.Flags().Set(flagCategory, "authentication")
			require.NoError(t, err)
			err = moveCmd.Flags().Set(flagName, "plugin-a")
			require.NoError(t, err)
			err = moveCmd.Flags().Set(flagPosition, tc.position)
			require.NoError(t, err)

			err = executeCmd(t, moveCmd, []string{})
			require.Error(t, err)
			require.ErrorContains(t, err, tc.error)
		})
	}
}

func TestMoveCmd_After(t *testing.T) {
	t.Parallel()

	loader := newMockLoaderFromFile(t)

	// Setup: Add three plugins.
	cfgModifier, err := loader.Load("ignored")
	require.NoError(t, err)
	cfg := cfgModifier.(*config.Config)

	_, err = cfg.UpsertPlugin(config.CategoryAuthentication, config.PluginEntry{
		Name:  "plugin-a",
		Flows: []config.Flow{config.FlowRequest},
	})
	require.NoError(t, err)

	_, err = cfg.UpsertPlugin(config.CategoryAuthentication, config.PluginEntry{
		Name:  "plugin-b",
		Flows: []config.Flow{config.FlowRequest},
	})
	require.NoError(t, err)

	_, err = cfg.UpsertPlugin(config.CategoryAuthentication, config.PluginEntry{
		Name:  "plugin-c",
		Flows: []config.Flow{config.FlowRequest},
	})
	require.NoError(t, err)

	base := &cmd.BaseCmd{}
	moveCmd, err := NewMoveCmd(base, cmdopts.WithConfigLoader(loader))
	require.NoError(t, err)

	// Move plugin-a after plugin-b.
	err = moveCmd.Flags().Set(flagCategory, "authentication")
	require.NoError(t, err)
	err = moveCmd.Flags().Set(flagName, "plugin-a")
	require.NoError(t, err)
	err = moveCmd.Flags().Set(flagAfter, "plugin-b")
	require.NoError(t, err)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	moveCmd.SetOut(&stdout)
	moveCmd.SetErr(&stderr)

	err = executeCmd(t, moveCmd, []string{})
	require.NoError(t, err)
	require.Empty(t, stderr.String())
	require.Contains(t, stdout.String(), "Plugin 'plugin-a' moved")

	// Verify order: plugin-b, plugin-a, plugin-c.
	cfgModifier, err = loader.Load("ignored")
	require.NoError(t, err)
	cfg = cfgModifier.(*config.Config)

	authPlugins := cfg.Plugins.ListPlugins(config.CategoryAuthentication)
	require.Len(t, authPlugins, 3)
	require.Equal(t, "plugin-b", authPlugins[0].Name)
	require.Equal(t, "plugin-a", authPlugins[1].Name)
	require.Equal(t, "plugin-c", authPlugins[2].Name)
}

func TestMoveCmd_ToPosition(t *testing.T) {
	t.Parallel()

	loader := newMockLoaderFromFile(t)

	// Setup: Add three plugins.
	cfgModifier, err := loader.Load("ignored")
	require.NoError(t, err)
	cfg := cfgModifier.(*config.Config)

	_, err = cfg.UpsertPlugin(config.CategoryAuthentication, config.PluginEntry{
		Name:  "plugin-a",
		Flows: []config.Flow{config.FlowRequest},
	})
	require.NoError(t, err)
	_, err = cfg.UpsertPlugin(config.CategoryAuthentication, config.PluginEntry{
		Name:  "plugin-b",
		Flows: []config.Flow{config.FlowRequest},
	})
	require.NoError(t, err)
	_, err = cfg.UpsertPlugin(config.CategoryAuthentication, config.PluginEntry{
		Name:  "plugin-c",
		Flows: []config.Flow{config.FlowRequest},
	})
	require.NoError(t, err)

	base := &cmd.BaseCmd{}
	moveCmd, err := NewMoveCmd(base, cmdopts.WithConfigLoader(loader))
	require.NoError(t, err)

	// Move plugin-c to position 1 (first).
	err = moveCmd.Flags().Set(flagCategory, "authentication")
	require.NoError(t, err)
	err = moveCmd.Flags().Set(flagName, "plugin-c")
	require.NoError(t, err)
	err = moveCmd.Flags().Set(flagPosition, "1")
	require.NoError(t, err)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	moveCmd.SetOut(&stdout)
	moveCmd.SetErr(&stderr)

	err = executeCmd(t, moveCmd, []string{})
	require.NoError(t, err)
	require.Empty(t, stderr.String())
	require.Contains(t, stdout.String(), "Plugin 'plugin-c' moved")

	// Verify order: plugin-c, plugin-a, plugin-b.
	cfgModifier, err = loader.Load("ignored")
	require.NoError(t, err)
	cfg = cfgModifier.(*config.Config)

	authPlugins := cfg.Plugins.ListPlugins(config.CategoryAuthentication)
	require.Len(t, authPlugins, 3)
	require.Equal(t, "plugin-c", authPlugins[0].Name)
	require.Equal(t, "plugin-a", authPlugins[1].Name)
	require.Equal(t, "plugin-b", authPlugins[2].Name)
}

func TestMoveCmd_ToCategoryAndPosition(t *testing.T) {
	t.Parallel()

	loader := newMockLoaderFromFile(t)

	// Setup: Add plugin to auth and two plugins to audit.
	cfgModifier, err := loader.Load("ignored")
	require.NoError(t, err)
	cfg := cfgModifier.(*config.Config)

	_, err = cfg.UpsertPlugin(config.CategoryAuthentication, config.PluginEntry{
		Name:  "jwt-auth",
		Flows: []config.Flow{config.FlowRequest},
	})
	require.NoError(t, err)
	_, err = cfg.UpsertPlugin(config.CategoryAudit, config.PluginEntry{
		Name:  "audit-a",
		Flows: []config.Flow{config.FlowRequest},
	})
	require.NoError(t, err)
	_, err = cfg.UpsertPlugin(config.CategoryAudit, config.PluginEntry{
		Name:  "audit-b",
		Flows: []config.Flow{config.FlowRequest},
	})
	require.NoError(t, err)

	base := &cmd.BaseCmd{}
	moveCmd, err := NewMoveCmd(base, cmdopts.WithConfigLoader(loader))
	require.NoError(t, err)

	// Move jwt-auth to audit category at position 1 (first).
	err = moveCmd.Flags().Set(flagCategory, "authentication")
	require.NoError(t, err)
	err = moveCmd.Flags().Set(flagName, "jwt-auth")
	require.NoError(t, err)
	err = moveCmd.Flags().Set(flagToCategory, "audit")
	require.NoError(t, err)
	err = moveCmd.Flags().Set(flagPosition, "1")
	require.NoError(t, err)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	moveCmd.SetOut(&stdout)
	moveCmd.SetErr(&stderr)

	err = executeCmd(t, moveCmd, []string{})
	require.NoError(t, err)
	require.Empty(t, stderr.String())
	require.Contains(t, stdout.String(), "Plugin 'jwt-auth' moved")

	// Verify: auth is empty, audit has jwt-auth first.
	cfgModifier, err = loader.Load("ignored")
	require.NoError(t, err)
	cfg = cfgModifier.(*config.Config)

	authPlugins := cfg.Plugins.ListPlugins(config.CategoryAuthentication)
	require.Empty(t, authPlugins)

	auditPlugins := cfg.Plugins.ListPlugins(config.CategoryAudit)
	require.Len(t, auditPlugins, 3)
	require.Equal(t, "jwt-auth", auditPlugins[0].Name)
	require.Equal(t, "audit-a", auditPlugins[1].Name)
	require.Equal(t, "audit-b", auditPlugins[2].Name)
}

func TestMoveCmd_NoOperationSpecified(t *testing.T) {
	t.Parallel()

	loader := newMockLoaderFromFile(t)

	base := &cmd.BaseCmd{}
	moveCmd, err := NewMoveCmd(base, cmdopts.WithConfigLoader(loader))
	require.NoError(t, err)

	err = moveCmd.Flags().Set(flagCategory, "authentication")
	require.NoError(t, err)
	err = moveCmd.Flags().Set(flagName, "jwt-auth")
	require.NoError(t, err)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	moveCmd.SetOut(&stdout)
	moveCmd.SetErr(&stderr)

	err = executeCmd(t, moveCmd, []string{})
	require.Error(t, err)
	require.ErrorContains(t, err, "one of --to-category, --before, --after, or --position must be specified")
}

func TestMoveCmd_SameCategoryError(t *testing.T) {
	t.Parallel()

	loader := newMockLoaderFromFile(t)

	base := &cmd.BaseCmd{}
	moveCmd, err := NewMoveCmd(base, cmdopts.WithConfigLoader(loader))
	require.NoError(t, err)

	err = moveCmd.Flags().Set(flagCategory, "authentication")
	require.NoError(t, err)
	err = moveCmd.Flags().Set(flagName, "jwt-auth")
	require.NoError(t, err)
	err = moveCmd.Flags().Set(flagToCategory, "authentication")
	require.NoError(t, err)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	moveCmd.SetOut(&stdout)
	moveCmd.SetErr(&stderr)

	err = executeCmd(t, moveCmd, []string{})
	require.Error(t, err)
	require.ErrorContains(t, err, "plugin is already in category")
}

func TestMoveCmd_PluginNotFound(t *testing.T) {
	t.Parallel()

	loader := newMockLoaderFromFile(t)

	// Setup: Add a different plugin.
	cfgModifier, err := loader.Load("ignored")
	require.NoError(t, err)
	cfg := cfgModifier.(*config.Config)

	_, err = cfg.UpsertPlugin(config.CategoryAuthentication, config.PluginEntry{
		Name:  "other-plugin",
		Flows: []config.Flow{config.FlowRequest},
	})
	require.NoError(t, err)

	base := &cmd.BaseCmd{}
	moveCmd, err := NewMoveCmd(base, cmdopts.WithConfigLoader(loader))
	require.NoError(t, err)

	err = moveCmd.Flags().Set(flagCategory, "authentication")
	require.NoError(t, err)
	err = moveCmd.Flags().Set(flagName, "nonexistent")
	require.NoError(t, err)
	err = moveCmd.Flags().Set(flagToCategory, "audit")
	require.NoError(t, err)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	moveCmd.SetOut(&stdout)
	moveCmd.SetErr(&stderr)

	err = executeCmd(t, moveCmd, []string{})
	require.Error(t, err)
	require.ErrorContains(t, err, "plugin 'nonexistent' not found")
}

func TestMoveCmd_TargetPluginNotFound(t *testing.T) {
	t.Parallel()

	loader := newMockLoaderFromFile(t)

	// Setup: Add two plugins (need at least 2 to test target not found).
	cfgModifier, err := loader.Load("ignored")
	require.NoError(t, err)
	cfg := cfgModifier.(*config.Config)

	_, err = cfg.UpsertPlugin(config.CategoryAuthentication, config.PluginEntry{
		Name:  "plugin-a",
		Flows: []config.Flow{config.FlowRequest},
	})
	require.NoError(t, err)
	_, err = cfg.UpsertPlugin(config.CategoryAuthentication, config.PluginEntry{
		Name:  "plugin-b",
		Flows: []config.Flow{config.FlowRequest},
	})
	require.NoError(t, err)

	base := &cmd.BaseCmd{}
	moveCmd, err := NewMoveCmd(base, cmdopts.WithConfigLoader(loader))
	require.NoError(t, err)

	err = moveCmd.Flags().Set(flagCategory, "authentication")
	require.NoError(t, err)
	err = moveCmd.Flags().Set(flagName, "plugin-a")
	require.NoError(t, err)
	err = moveCmd.Flags().Set(flagBefore, "nonexistent")
	require.NoError(t, err)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	moveCmd.SetOut(&stdout)
	moveCmd.SetErr(&stderr)

	err = executeCmd(t, moveCmd, []string{})
	require.Error(t, err)
	require.ErrorContains(t, err, "target plugin 'nonexistent' not found")
}

func TestMoveCmd_ForceOverwrite(t *testing.T) {
	t.Parallel()

	loader := newMockLoaderFromFile(t)

	// Setup: Add plugin with same name in both categories.
	cfgModifier, err := loader.Load("ignored")
	require.NoError(t, err)
	cfg := cfgModifier.(*config.Config)

	_, err = cfg.UpsertPlugin(config.CategoryAuthentication, config.PluginEntry{
		Name:  "shared-name",
		Flows: []config.Flow{config.FlowRequest},
	})
	require.NoError(t, err)
	_, err = cfg.UpsertPlugin(config.CategoryAudit, config.PluginEntry{
		Name:  "shared-name",
		Flows: []config.Flow{config.FlowResponse},
	})
	require.NoError(t, err)

	base := &cmd.BaseCmd{}
	moveCmd, err := NewMoveCmd(base, cmdopts.WithConfigLoader(loader))
	require.NoError(t, err)

	// Move from auth to audit with force.
	err = moveCmd.Flags().Set(flagCategory, "authentication")
	require.NoError(t, err)
	err = moveCmd.Flags().Set(flagName, "shared-name")
	require.NoError(t, err)
	err = moveCmd.Flags().Set(flagToCategory, "audit")
	require.NoError(t, err)
	err = moveCmd.Flags().Set(flagForce, "true")
	require.NoError(t, err)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	moveCmd.SetOut(&stdout)
	moveCmd.SetErr(&stderr)

	err = executeCmd(t, moveCmd, []string{})
	require.NoError(t, err)
	require.Empty(t, stderr.String())
	require.Contains(t, stdout.String(), "Plugin 'shared-name' moved")

	// Verify: auth is empty, audit has plugin with FlowRequest (from auth).
	cfgModifier, err = loader.Load("ignored")
	require.NoError(t, err)
	cfg = cfgModifier.(*config.Config)

	authPlugins := cfg.Plugins.ListPlugins(config.CategoryAuthentication)
	require.Empty(t, authPlugins)

	auditPlugins := cfg.Plugins.ListPlugins(config.CategoryAudit)
	require.Len(t, auditPlugins, 1)
	require.Equal(t, "shared-name", auditPlugins[0].Name)
	require.Equal(t, []config.Flow{config.FlowRequest}, auditPlugins[0].Flows)
}

func TestMoveCmd_DuplicateWithoutForce(t *testing.T) {
	t.Parallel()

	loader := newMockLoaderFromFile(t)

	// Setup: Add plugin with same name in both categories.
	cfgModifier, err := loader.Load("ignored")
	require.NoError(t, err)
	cfg := cfgModifier.(*config.Config)

	_, err = cfg.UpsertPlugin(config.CategoryAuthentication, config.PluginEntry{
		Name:  "shared-name",
		Flows: []config.Flow{config.FlowRequest},
	})
	require.NoError(t, err)
	_, err = cfg.UpsertPlugin(config.CategoryAudit, config.PluginEntry{
		Name:  "shared-name",
		Flows: []config.Flow{config.FlowResponse},
	})
	require.NoError(t, err)

	base := &cmd.BaseCmd{}
	moveCmd, err := NewMoveCmd(base, cmdopts.WithConfigLoader(loader))
	require.NoError(t, err)

	// Move from auth to audit without force - should fail.
	err = moveCmd.Flags().Set(flagCategory, "authentication")
	require.NoError(t, err)
	err = moveCmd.Flags().Set(flagName, "shared-name")
	require.NoError(t, err)
	err = moveCmd.Flags().Set(flagToCategory, "audit")
	require.NoError(t, err)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	moveCmd.SetOut(&stdout)
	moveCmd.SetErr(&stderr)

	err = executeCmd(t, moveCmd, []string{})
	require.Error(t, err)
	require.ErrorContains(t, err, "plugin 'shared-name' already exists in category 'audit'")
}

func TestMoveCmd_ToEnd(t *testing.T) {
	t.Parallel()

	loader := newMockLoaderFromFile(t)

	// Setup: Add three plugins.
	cfgModifier, err := loader.Load("ignored")
	require.NoError(t, err)
	cfg := cfgModifier.(*config.Config)

	_, err = cfg.UpsertPlugin(config.CategoryAuthentication, config.PluginEntry{
		Name:  "plugin-a",
		Flows: []config.Flow{config.FlowRequest},
	})
	require.NoError(t, err)
	_, err = cfg.UpsertPlugin(config.CategoryAuthentication, config.PluginEntry{
		Name:  "plugin-b",
		Flows: []config.Flow{config.FlowRequest},
	})
	require.NoError(t, err)
	_, err = cfg.UpsertPlugin(config.CategoryAuthentication, config.PluginEntry{
		Name:  "plugin-c",
		Flows: []config.Flow{config.FlowRequest},
	})
	require.NoError(t, err)

	base := &cmd.BaseCmd{}
	moveCmd, err := NewMoveCmd(base, cmdopts.WithConfigLoader(loader))
	require.NoError(t, err)

	// Move plugin-a to end using position -1.
	err = moveCmd.Flags().Set(flagCategory, "authentication")
	require.NoError(t, err)
	err = moveCmd.Flags().Set(flagName, "plugin-a")
	require.NoError(t, err)
	err = moveCmd.Flags().Set(flagPosition, "-1")
	require.NoError(t, err)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	moveCmd.SetOut(&stdout)
	moveCmd.SetErr(&stderr)

	err = executeCmd(t, moveCmd, []string{})
	require.NoError(t, err)
	require.Empty(t, stderr.String())
	require.Contains(t, stdout.String(), "Plugin 'plugin-a' moved")

	// Verify order: plugin-b, plugin-c, plugin-a.
	cfgModifier, err = loader.Load("ignored")
	require.NoError(t, err)
	cfg = cfgModifier.(*config.Config)

	authPlugins := cfg.Plugins.ListPlugins(config.CategoryAuthentication)
	require.Len(t, authPlugins, 3)
	require.Equal(t, "plugin-b", authPlugins[0].Name)
	require.Equal(t, "plugin-c", authPlugins[1].Name)
	require.Equal(t, "plugin-a", authPlugins[2].Name)
}

func TestMoveCmd_PrintsOrder(t *testing.T) {
	t.Parallel()

	loader := newMockLoaderFromFile(t)

	// Setup: Add two plugins.
	cfgModifier, err := loader.Load("ignored")
	require.NoError(t, err)
	cfg := cfgModifier.(*config.Config)

	_, err = cfg.UpsertPlugin(config.CategoryAuthentication, config.PluginEntry{
		Name:  "plugin-a",
		Flows: []config.Flow{config.FlowRequest},
	})
	require.NoError(t, err)
	_, err = cfg.UpsertPlugin(config.CategoryAuthentication, config.PluginEntry{
		Name:  "plugin-b",
		Flows: []config.Flow{config.FlowRequest},
	})
	require.NoError(t, err)

	base := &cmd.BaseCmd{}
	moveCmd, err := NewMoveCmd(base, cmdopts.WithConfigLoader(loader))
	require.NoError(t, err)

	err = moveCmd.Flags().Set(flagCategory, "authentication")
	require.NoError(t, err)
	err = moveCmd.Flags().Set(flagName, "plugin-b")
	require.NoError(t, err)
	err = moveCmd.Flags().Set(flagBefore, "plugin-a")
	require.NoError(t, err)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	moveCmd.SetOut(&stdout)
	moveCmd.SetErr(&stderr)

	err = executeCmd(t, moveCmd, []string{})
	require.NoError(t, err)
	require.Empty(t, stderr.String())

	// Verify output contains order information.
	output := stdout.String()
	require.Contains(t, output, "Order in 'authentication':")
	require.Contains(t, output, "1. plugin-b")
	require.Contains(t, output, "2. plugin-a")
}

func TestMoveCmd_NoPluginsConfigured(t *testing.T) {
	t.Parallel()

	loader := newMockLoaderFromFile(t)

	base := &cmd.BaseCmd{}
	moveCmd, err := NewMoveCmd(base, cmdopts.WithConfigLoader(loader))
	require.NoError(t, err)

	err = moveCmd.Flags().Set(flagCategory, "authentication")
	require.NoError(t, err)
	err = moveCmd.Flags().Set(flagName, "jwt-auth")
	require.NoError(t, err)
	err = moveCmd.Flags().Set(flagToCategory, "audit")
	require.NoError(t, err)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	moveCmd.SetOut(&stdout)
	moveCmd.SetErr(&stderr)

	err = executeCmd(t, moveCmd, []string{})
	require.Error(t, err)
	require.ErrorContains(t, err, "no plugins configured")
}
