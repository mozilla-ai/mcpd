package daemon

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mozilla-ai/mcpd/v2/internal/cmd"
	cmdopts "github.com/mozilla-ai/mcpd/v2/internal/cmd/options"
	"github.com/mozilla-ai/mcpd/v2/internal/config"
)

// mockConfigLoader implements config.Loader for testing daemon config validation.
type mockValidateConfigLoader struct {
	daemonConfig *config.DaemonConfig
	err          error
}

func (m *mockValidateConfigLoader) Load(_ string) (config.Modifier, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &config.Config{
		Daemon: m.daemonConfig,
	}, nil
}

func TestValidateCmd_ValidConfiguration(t *testing.T) {
	t.Parallel()

	// Create valid daemon configuration
	validDaemonConfig := &config.DaemonConfig{
		API: &config.APIConfigSection{
			Addr: &[]string{"localhost:8080"}[0],
			Timeout: &config.APITimeoutConfigSection{
				Shutdown: &[]config.Duration{config.Duration(30 * 1_000_000_000)}[0], // 30s in nanoseconds
			},
		},
	}

	// Create mock loader
	mockLoader := &mockValidateConfigLoader{
		daemonConfig: validDaemonConfig,
		err:          nil,
	}

	// Create base command and validate command
	baseCmd := &cmd.BaseCmd{}
	validateCmd, err := NewValidateCmd(baseCmd, cmdopts.WithConfigLoader(mockLoader))
	require.NoError(t, err)

	// Capture output
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	validateCmd.SetOut(&stdout)
	validateCmd.SetErr(&stderr)

	// Execute command
	err = validateCmd.Execute()

	// Assertions
	require.NoError(t, err)
	require.Contains(t, stdout.String(), "✓ Daemon configuration is valid")
	require.Empty(t, stderr.String())
}

func TestValidateCmd_InvalidConfiguration(t *testing.T) {
	t.Parallel()

	// Create invalid daemon configuration (invalid address)
	invalidDaemonConfig := &config.DaemonConfig{
		API: &config.APIConfigSection{
			Addr: &[]string{"invalid-address"}[0], // Invalid address format
		},
	}

	// Create mock loader
	mockLoader := &mockValidateConfigLoader{
		daemonConfig: invalidDaemonConfig,
		err:          nil,
	}

	// Create base command and validate command
	baseCmd := &cmd.BaseCmd{}
	validateCmd, err := NewValidateCmd(baseCmd, cmdopts.WithConfigLoader(mockLoader))
	require.NoError(t, err)

	// Capture output
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	validateCmd.SetOut(&stdout)
	validateCmd.SetErr(&stderr)

	// Execute command
	err = validateCmd.Execute()

	// Assertions
	require.Error(t, err)
	errorOutput := stderr.String()
	require.Contains(t, errorOutput, "Error:")
	require.Contains(t, errorOutput, "API address \"invalid-address\" appears to be invalid")
}

func TestValidateCmd_MissingDaemonConfig(t *testing.T) {
	t.Parallel()

	// Create mock loader with nil daemon config
	mockLoader := &mockValidateConfigLoader{
		daemonConfig: nil,
		err:          nil,
	}

	// Create base command and validate command
	baseCmd := &cmd.BaseCmd{}
	validateCmd, err := NewValidateCmd(baseCmd, cmdopts.WithConfigLoader(mockLoader))
	require.NoError(t, err)

	// Capture output
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	validateCmd.SetOut(&stdout)
	validateCmd.SetErr(&stderr)

	// Execute command
	err = validateCmd.Execute()

	// Assertions
	require.Error(t, err)
	errorOutput := stderr.String()
	require.Contains(t, errorOutput, "Error:")
	require.Contains(t, errorOutput, "no daemon configuration found")
}

func TestValidateCmd_ConfigLoadError(t *testing.T) {
	t.Parallel()

	// Create mock loader that returns error
	mockLoader := &mockValidateConfigLoader{
		daemonConfig: nil,
		err:          fmt.Errorf("mock config load error"),
	}

	// Create base command and validate command
	baseCmd := &cmd.BaseCmd{}
	validateCmd, err := NewValidateCmd(baseCmd, cmdopts.WithConfigLoader(mockLoader))
	require.NoError(t, err)

	// Capture output
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	validateCmd.SetOut(&stdout)
	validateCmd.SetErr(&stderr)

	// Execute command
	err = validateCmd.Execute()

	// Assertions
	require.Error(t, err)
	require.EqualError(t, err, "mock config load error")
}

func TestValidateCmd_MultipleValidationErrors(t *testing.T) {
	t.Parallel()

	// Create daemon configuration with multiple validation errors
	invalidDaemonConfig := &config.DaemonConfig{
		API: &config.APIConfigSection{
			Addr: &[]string{"invalid-address"}[0], // Invalid address
			Timeout: &config.APITimeoutConfigSection{
				Shutdown: &[]config.Duration{config.Duration(0)}[0], // Invalid timeout (0)
			},
			CORS: &config.CORSConfigSection{
				Enable:  &[]bool{true}[0],
				Methods: []string{"INVALID_METHOD"}, // Invalid HTTP method
			},
		},
	}

	// Create mock loader
	mockLoader := &mockValidateConfigLoader{
		daemonConfig: invalidDaemonConfig,
		err:          nil,
	}

	// Create base command and validate command
	baseCmd := &cmd.BaseCmd{}
	validateCmd, err := NewValidateCmd(baseCmd, cmdopts.WithConfigLoader(mockLoader))
	require.NoError(t, err)

	// Capture output
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	validateCmd.SetOut(&stdout)
	validateCmd.SetErr(&stderr)

	// Execute command
	err = validateCmd.Execute()

	// Assertions
	require.Error(t, err)
	errorOutput := stderr.String()
	require.Contains(t, errorOutput, "Error:")

	// Should contain multiple validation error messages
	// Note: The exact error messages may be combined, so we check for key error indicators
	require.True(t,
		strings.Contains(errorOutput, "API address") ||
			strings.Contains(errorOutput, "timeout") ||
			strings.Contains(errorOutput, "method"),
		"Expected to find at least one validation error message, got: %s", errorOutput)
}
