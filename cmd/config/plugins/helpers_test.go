package plugins

import (
	"testing"

	"github.com/spf13/cobra"

	"github.com/mozilla-ai/mcpd/internal/config"
)

// executeCmd runs both PreRunE and RunE hooks for the command.
func executeCmd(t *testing.T, cobraCmd *cobra.Command, args []string) error {
	t.Helper()

	// Run PreRunE if it exists.
	if cobraCmd.PreRunE != nil {
		if err := cobraCmd.PreRunE(cobraCmd, args); err != nil {
			return err
		}
	}

	// Run RunE.
	if cobraCmd.RunE != nil {
		return cobraCmd.RunE(cobraCmd, args)
	}

	return nil
}

// mockLoader is a mock config.Loader for testing.
type mockLoader struct {
	cfg *config.Config
	err error
}

func (m *mockLoader) Load(_ string) (config.Modifier, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.cfg, nil
}
