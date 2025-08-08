package mcpm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mozilla-ai/mcpd/v2/internal/packages"
	"github.com/mozilla-ai/mcpd/v2/internal/runtime"
)

func TestCLIArgParser_PositionalArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		args           []string
		schema         Arguments
		expectedResult map[string]packages.ArgumentMetadata
	}{
		{
			name: "obsidian-mcp style positional args",
			args: []string{
				"-y",
				"obsidian-mcp",
				"${OBSIDIAN_VAULT_PATH}",
				"${OBSIDIAN_VAULT_PATH2}",
			},
			schema: Arguments{
				"OBSIDIAN_VAULT_PATH": {
					Description: "Path to your Obsidian vault",
					Required:    true,
				},
				"OBSIDIAN_VAULT_PATH2": {
					Description: "Path to your second Obsidian vault",
					Required:    false,
				},
			},
			expectedResult: map[string]packages.ArgumentMetadata{
				"OBSIDIAN_VAULT_PATH": {
					Name:         "OBSIDIAN_VAULT_PATH",
					VariableType: packages.VariableTypeArgPositional,
					Position:     intPtr(1),
					Required:     true,
					Description:  "Path to your Obsidian vault",
				},
				"OBSIDIAN_VAULT_PATH2": {
					Name:         "OBSIDIAN_VAULT_PATH2",
					VariableType: packages.VariableTypeArgPositional,
					Position:     intPtr(2),
					Required:     false,
					Description:  "Path to your second Obsidian vault",
				},
			},
		},
		{
			name: "mixed positional and flag args",
			args: []string{
				"-y",
				"some-package",
				"${INPUT_FILE}",
				"--config=${CONFIG_PATH}",
				"${OUTPUT_FILE}",
				"--verbose",
			},
			schema: Arguments{
				"INPUT_FILE": {
					Description: "Input file path",
					Required:    true,
				},
				"OUTPUT_FILE": {
					Description: "Output file path",
					Required:    true,
				},
				"CONFIG_PATH": {
					Description: "Config file path",
					Required:    false,
				},
				"--verbose": {
					Description: "Enable verbose output",
					Required:    false,
				},
			},
			expectedResult: map[string]packages.ArgumentMetadata{
				"INPUT_FILE": {
					Name:         "INPUT_FILE",
					VariableType: packages.VariableTypeArgPositional,
					Position:     intPtr(1),
					Required:     true,
					Description:  "Input file path",
				},
				"OUTPUT_FILE": {
					Name:         "OUTPUT_FILE",
					VariableType: packages.VariableTypeArgPositional,
					Position:     intPtr(2),
					Required:     true,
					Description:  "Output file path",
				},
				"--config": {
					Name:         "--config",
					VariableType: packages.VariableTypeArg,
					Required:     false,
					Description:  "Config file path",
				},
				"--verbose": {
					Name:         "--verbose",
					VariableType: packages.VariableTypeArgBool,
					Required:     false,
					Description:  "Enable verbose output",
				},
			},
		},
		{
			name: "positional args without placeholders are ignored",
			args: []string{
				"-y",
				"package-name",
				"literal-value",
				"${ACTUAL_ARG}",
			},
			schema: Arguments{
				"ACTUAL_ARG": {
					Description: "An actual argument",
					Required:    true,
				},
			},
			expectedResult: map[string]packages.ArgumentMetadata{
				"ACTUAL_ARG": {
					Name:         "ACTUAL_ARG",
					VariableType: packages.VariableTypeArgPositional,
					Position:     intPtr(1), // First positional with placeholder
					Required:     true,
					Description:  "An actual argument",
				},
			},
		},
		{
			name: "flag args with placeholders",
			args: []string{
				"--token=${API_TOKEN}",
				"--config",
				"${CONFIG_FILE}",
				"--debug",
			},
			schema: Arguments{
				"API_TOKEN": {
					Description: "API authentication token",
					Required:    true,
				},
				"CONFIG_FILE": {
					Description: "Configuration file path",
					Required:    false,
				},
				"--debug": {
					Description: "Enable debug mode",
					Required:    false,
				},
			},
			expectedResult: map[string]packages.ArgumentMetadata{
				"--token": {
					Name:         "--token",
					VariableType: packages.VariableTypeArg,
					Required:     true,
					Description:  "API authentication token",
				},
				"--config": {
					Name:         "--config",
					VariableType: packages.VariableTypeArg,
					Required:     false,
					Description:  "Configuration file path",
				},
				"--debug": {
					Name:         "--debug",
					VariableType: packages.VariableTypeArgBool,
					Required:     false,
					Description:  "Enable debug mode",
				},
			},
		},
		{
			name: "no arguments to parse",
			args: []string{
				"-y",
				"simple-package",
			},
			schema:         Arguments{},
			expectedResult: map[string]packages.ArgumentMetadata{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			spec := runtime.Specs()[runtime.NPX]
			parser := NewCLIArgParser(tc.schema, spec)
			result := parser.Parse(tc.args)

			require.Equal(t, len(tc.expectedResult), len(result),
				"Expected %d results but got %d", len(tc.expectedResult), len(result))

			for key, expected := range tc.expectedResult {
				actual, exists := result[key]
				assert.True(t, exists, "Expected key %s not found in result", key)
				if exists {
					assert.Equal(t, expected.Name, actual.Name, "Name mismatch for %s", key)
					assert.Equal(t, expected.VariableType, actual.VariableType, "VariableType mismatch for %s", key)
					assert.Equal(t, expected.Required, actual.Required, "Required mismatch for %s", key)
					assert.Equal(t, expected.Description, actual.Description, "Description mismatch for %s", key)

					// Check position for positional args
					if expected.VariableType == packages.VariableTypeArgPositional {
						require.NotNil(t, actual.Position, "Position should not be nil for positional arg %s", key)
						if expected.Position != nil {
							assert.Equal(t, *expected.Position, *actual.Position, "Position mismatch for %s", key)
						}
					}
				}
			}
		})
	}
}

func TestExtractArgumentMetadata_RealObsidianCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		server         MCPServer
		expectedResult map[string]packages.ArgumentMetadata
	}{
		{
			name: "obsidian-mcp with positional args",
			server: MCPServer{
				Installations: map[string]Installation{
					"npm": {
						Type:    "npm",
						Command: "npx",
						Args: []string{
							"-y",
							"obsidian-mcp",
							"${OBSIDIAN_VAULT_PATH}",
							"${OBSIDIAN_VAULT_PATH2}",
						},
						Env: map[string]string{},
					},
				},
				Arguments: Arguments{
					"OBSIDIAN_VAULT_PATH": {
						Description: "Path to your Obsidian vault",
						Required:    true,
					},
					"OBSIDIAN_VAULT_PATH2": {
						Description: "Path to your second Obsidian vault",
						Required:    false,
					},
				},
			},
			expectedResult: map[string]packages.ArgumentMetadata{
				"OBSIDIAN_VAULT_PATH": {
					Name:         "OBSIDIAN_VAULT_PATH",
					VariableType: packages.VariableTypeArgPositional,
					Position:     intPtr(1),
					Required:     true,
					Description:  "Path to your Obsidian vault",
				},
				"OBSIDIAN_VAULT_PATH2": {
					Name:         "OBSIDIAN_VAULT_PATH2",
					VariableType: packages.VariableTypeArgPositional,
					Position:     intPtr(2),
					Required:     false,
					Description:  "Path to your second Obsidian vault",
				},
			},
		},
		{
			name: "mcp-obsidian with env var",
			server: MCPServer{
				Installations: map[string]Installation{
					"uvx": {
						Type:    "uvx",
						Command: "uvx",
						Args:    []string{"mcp-obsidian"},
						Env: map[string]string{
							"OBSIDIAN_API_KEY": "${OBSIDIAN_API_KEY}",
						},
					},
				},
				Arguments: Arguments{
					"OBSIDIAN_API_KEY": {
						Description: "Obsidian API key",
						Required:    true,
						Example:     "your-obsidian-api-key",
					},
				},
			},
			expectedResult: map[string]packages.ArgumentMetadata{
				"OBSIDIAN_API_KEY": {
					Name:         "OBSIDIAN_API_KEY",
					VariableType: packages.VariableTypeEnv,
					Required:     true,
					Description:  "Obsidian API key",
					Example:      "your-obsidian-api-key",
				},
			},
		},
		{
			name: "mixed env and positional",
			server: MCPServer{
				Installations: map[string]Installation{
					"npm": {
						Type:    "npm",
						Command: "npx",
						Args: []string{
							"-y",
							"some-server",
							"${INPUT_PATH}",
							"${OUTPUT_PATH}",
						},
						Env: map[string]string{
							"API_KEY": "${API_KEY}",
							"DEBUG":   "true",
						},
					},
				},
				Arguments: Arguments{
					"INPUT_PATH": {
						Description: "Input file path",
						Required:    true,
					},
					"OUTPUT_PATH": {
						Description: "Output file path",
						Required:    false,
					},
					"API_KEY": {
						Description: "API authentication key",
						Required:    true,
					},
				},
			},
			expectedResult: map[string]packages.ArgumentMetadata{
				"API_KEY": {
					Name:         "API_KEY",
					VariableType: packages.VariableTypeEnv,
					Required:     true,
					Description:  "API authentication key",
				},
				"DEBUG": {
					Name:         "DEBUG",
					VariableType: packages.VariableTypeEnv,
					Required:     false, // Not in schema
					Description:  "",
				},
				"INPUT_PATH": {
					Name:         "INPUT_PATH",
					VariableType: packages.VariableTypeArgPositional,
					Position:     intPtr(1),
					Required:     true,
					Description:  "Input file path",
				},
				"OUTPUT_PATH": {
					Name:         "OUTPUT_PATH",
					VariableType: packages.VariableTypeArgPositional,
					Position:     intPtr(2),
					Required:     false,
					Description:  "Output file path",
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			supported := map[runtime.Runtime]struct{}{
				runtime.NPX: {},
				runtime.UVX: {},
			}

			result := extractArgumentMetadata(tc.server, supported)

			require.Equal(t, len(tc.expectedResult), len(result),
				"Expected %d results but got %d", len(tc.expectedResult), len(result))

			for key, expected := range tc.expectedResult {
				actual, exists := result[key]
				assert.True(t, exists, "Expected key %s not found in result", key)
				if exists {
					assert.Equal(t, expected.Name, actual.Name, "Name mismatch for %s", key)
					assert.Equal(t, expected.VariableType, actual.VariableType, "VariableType mismatch for %s", key)
					assert.Equal(t, expected.Required, actual.Required, "Required mismatch for %s", key)
					assert.Equal(t, expected.Description, actual.Description, "Description mismatch for %s", key)
					assert.Equal(t, expected.Example, actual.Example, "Example mismatch for %s", key)

					// Check position for positional args
					if expected.VariableType == packages.VariableTypeArgPositional {
						require.NotNil(t, actual.Position, "Position should not be nil for positional arg %s", key)
						if expected.Position != nil {
							assert.Equal(t, *expected.Position, *actual.Position, "Position mismatch for %s", key)
						}
					} else {
						assert.Nil(t, actual.Position, "Position should be nil for non-positional arg %s", key)
					}
				}
			}
		})
	}
}

func TestCLIArgParser_EdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		args           []string
		schema         Arguments
		runtime        runtime.Runtime
		expectedResult map[string]packages.ArgumentMetadata
	}{
		{
			name: "ignored runtime flags",
			args: []string{
				"-y", // Should be ignored for NPX
				"package-name",
				"${USER_ARG}",
			},
			schema: Arguments{
				"USER_ARG": {
					Description: "User provided argument",
					Required:    true,
				},
			},
			runtime: runtime.NPX,
			expectedResult: map[string]packages.ArgumentMetadata{
				"USER_ARG": {
					Name:         "USER_ARG",
					VariableType: packages.VariableTypeArgPositional,
					Position:     intPtr(1),
					Required:     true,
					Description:  "User provided argument",
				},
			},
		},
		{
			name: "docker runtime specific flags",
			args: []string{
				"run",
				"--rm",
				"-i",
				"${IMAGE_NAME}",
			},
			schema: Arguments{
				"IMAGE_NAME": {
					Description: "Docker image name",
					Required:    true,
				},
			},
			runtime: runtime.Docker,
			expectedResult: map[string]packages.ArgumentMetadata{
				"IMAGE_NAME": {
					Name:         "IMAGE_NAME",
					VariableType: packages.VariableTypeArgPositional,
					Position:     intPtr(1),
					Required:     true,
					Description:  "Docker image name",
				},
			},
		},
		{
			name: "placeholders not in schema are ignored",
			args: []string{
				"${UNKNOWN_PLACEHOLDER}",
				"${KNOWN_ARG}",
			},
			schema: Arguments{
				"KNOWN_ARG": {
					Description: "Known argument",
					Required:    true,
				},
			},
			runtime: runtime.NPX,
			expectedResult: map[string]packages.ArgumentMetadata{
				"KNOWN_ARG": {
					Name:         "KNOWN_ARG",
					VariableType: packages.VariableTypeArgPositional,
					Position:     intPtr(1),
					Required:     true,
					Description:  "Known argument",
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			spec := runtime.Specs()[tc.runtime]
			parser := NewCLIArgParser(tc.schema, spec)
			result := parser.Parse(tc.args)

			require.Equal(t, len(tc.expectedResult), len(result),
				"Expected %d results but got %d", len(tc.expectedResult), len(result))

			for key, expected := range tc.expectedResult {
				actual, exists := result[key]
				assert.True(t, exists, "Expected key %s not found in result", key)
				if exists {
					assert.Equal(t, expected, actual, "Metadata mismatch for %s", key)
				}
			}
		})
	}
}

func intPtr(i int) *int {
	return &i
}
