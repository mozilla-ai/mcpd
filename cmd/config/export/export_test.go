package export

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mozilla-ai/mcpd/v2/internal/cmd"
	"github.com/mozilla-ai/mcpd/v2/internal/flags"
)

const (
	// testFileConfig is the name of the test input file to use in place of .mcpd.toml
	testFileConfig = "config.test.toml"

	// testFileSecrets is the name of the test input file which to use in place of ~/.config/mcpd/secrets.dev.toml
	testFileSecrets = "secrets.test.toml"

	// testFileContext is the name of the test output file which represents the expected secrets.prod.toml
	testFileContext = "context.test.toml"

	// testFileContract is the name of the test output file which represents the expected .env (etc.)
	// NOTE: This may require reviewing as more formats are supported for contract export.
	testFileContract = "contract.test.env"
)

// testDataPaths represents the file paths to particular input files used by mcpd
// and output file paths mcpd should target when exporting contract and context data
// from the cmd: mcpd config export.
type testDataPaths struct {
	configFile   string
	secretsFile  string
	contextFile  string
	contractFile string
}

// dataPathsForTest returns the testDataPaths for a test relative to the supplied testdataDir.
// testdataDir should usually reference a temporary directory created via t.TempDir().
func dataPathsForTest(t *testing.T, testdataDir string) testDataPaths {
	t.Helper()

	testdataPath := filepath.Join("testdata", testdataDir)

	paths := testDataPaths{
		configFile:   filepath.Join(testdataPath, testFileConfig),
		secretsFile:  filepath.Join(testdataPath, testFileSecrets),
		contextFile:  filepath.Join(testdataPath, testFileContext),
		contractFile: filepath.Join(testdataPath, testFileContract),
	}

	// Verify testdata files exist
	require.FileExists(t, paths.configFile)
	require.FileExists(t, paths.secretsFile)
	require.FileExists(t, paths.contextFile)
	require.FileExists(t, paths.contractFile)

	return paths
}

// overrideFlagsForTest temporarily overrides global flags.ConfigFile and flags.RuntimeFile
// for the duration of the test, automatically restoring original values on test completion.
func overrideFlagsForTest(t *testing.T, configFile string, runtimeFile string) {
	t.Helper()

	prevConfig := flags.ConfigFile
	prevRuntime := flags.RuntimeFile

	flags.ConfigFile = configFile
	flags.RuntimeFile = runtimeFile

	t.Cleanup(func() {
		flags.ConfigFile = prevConfig
		flags.RuntimeFile = prevRuntime
	})
}

func TestExportCommand_Integration(t *testing.T) {
	tests := []struct {
		name        string
		testdataDir string
	}{
		{
			name:        "basic export with all configuration types",
			testdataDir: "basic_export",
		},
		{
			name:        "multiple servers",
			testdataDir: "multiple_servers",
		},
		{
			name:        "server with hyphens in name",
			testdataDir: "github_server_hyphens",
		},
		{
			name:        "only required configuration (no runtime secrets)",
			testdataDir: "minimal_server_required_only",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			// Get testdata file paths
			paths := dataPathsForTest(t, tc.testdataDir)
			// Override global flags for test
			overrideFlagsForTest(t, paths.configFile, paths.secretsFile)
			// Configure the location we are exporting to.
			contextOutput := filepath.Join(tmpDir, "exported.context.toml")
			contractOutput := filepath.Join(tmpDir, "exported.env")

			exportCmd, err := NewCmd(&cmd.BaseCmd{})
			require.NoError(t, err)

			// Set command-specific flags
			require.NoError(t, exportCmd.Flags().Set("context-output", contextOutput))
			require.NoError(t, exportCmd.Flags().Set("contract-output", contractOutput))

			// Execute command
			err = exportCmd.RunE(exportCmd, []string{})
			require.NoError(t, err)

			// Verify output files exist
			require.FileExists(t, contextOutput)
			require.FileExists(t, contractOutput)

			// Verify context output content
			contextContent, err := os.ReadFile(contextOutput)
			require.NoError(t, err)
			expectedContextContent, err := os.ReadFile(paths.contextFile)
			require.NoError(t, err)
			require.Equal(
				t,
				strings.TrimSpace(string(expectedContextContent)),
				strings.TrimSpace(string(contextContent)),
			)

			// Verify contract output content
			contractContent, err := os.ReadFile(contractOutput)
			require.NoError(t, err)
			expectedContractContent, err := os.ReadFile(paths.contractFile)
			require.NoError(t, err)

			// Parse both files into sorted slices for comparison
			actualLines := strings.Split(strings.TrimSpace(string(contractContent)), "\n")
			expectedLines := strings.Split(strings.TrimSpace(string(expectedContractContent)), "\n")

			// Both should be already sorted, but verify they match exactly
			require.Equal(t, expectedLines, actualLines, "contract file content should match expected")
		})
	}
}

func TestExportCommand_Errors(t *testing.T) {
	tests := []struct {
		name          string
		mcpdConfig    string
		secretsConfig string
		expectedError string
	}{
		{
			name: "no servers in config",
			mcpdConfig: `# Empty config file
`,
			secretsConfig: `[servers]
# Empty secrets
`,
			expectedError: "export error, no servers defined in runtime config",
		},
		{
			name: "malformed config file",
			mcpdConfig: `[[servers]
name = "test-server" # Missing closing bracket
`,
			secretsConfig: `[servers]
`,
			expectedError: "failed to load config",
		},
		{
			name: "malformed secrets file",
			mcpdConfig: `[[servers]]
name = "test-server"
package = "uvx::test@latest"
`,
			secretsConfig: `[servers
# Missing closing bracket
`,
			expectedError: "failed to load execution context config",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			// Create input files
			configFile := filepath.Join(tmpDir, "test.mcpd.toml")
			secretsFile := filepath.Join(tmpDir, "secrets.test.toml")
			require.NoError(t, os.WriteFile(configFile, []byte(tc.mcpdConfig), 0o644))
			require.NoError(t, os.WriteFile(secretsFile, []byte(tc.secretsConfig), 0o644))

			// Set target files for export.
			contextOutput := filepath.Join(tmpDir, "exported.context.toml")
			contractOutput := filepath.Join(tmpDir, "exported.env")

			// Override global flags for test
			overrideFlagsForTest(t, configFile, secretsFile)

			// Create command
			exportCmd, err := NewCmd(&cmd.BaseCmd{})
			require.NoError(t, err)

			// Set command-specific flags
			require.NoError(t, exportCmd.Flags().Set("context-output", contextOutput))
			require.NoError(t, exportCmd.Flags().Set("contract-output", contractOutput))

			// Execute command - should fail
			err = exportCmd.RunE(exportCmd, []string{})
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.expectedError)

			// Verify output files were not created
			require.NoFileExists(t, contextOutput)
			require.NoFileExists(t, contractOutput)
		})
	}
}

func TestWriteDotenvFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		data          map[string]string
		expectedLines []string
	}{
		{
			name:          "empty data",
			data:          map[string]string{},
			expectedLines: []string{},
		},
		{
			name: "single entry",
			data: map[string]string{
				"KEY": "value",
			},
			expectedLines: []string{
				"KEY=value",
			},
		},
		{
			name: "multiple entries (should be sorted)",
			data: map[string]string{
				"Z_KEY": "z_value",
				"A_KEY": "a_value",
				"M_KEY": "m_value",
			},
			expectedLines: []string{
				"A_KEY=a_value",
				"M_KEY=m_value",
				"Z_KEY=z_value",
			},
		},
		{
			name: "values with newlines (should be escaped)",
			data: map[string]string{
				"MULTILINE": "line1\nline2\nline3",
				"SINGLE":    "normal",
			},
			expectedLines: []string{
				"MULTILINE=line1\\nline2\\nline3",
				"SINGLE=normal",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tmpDir := t.TempDir()
			filePath := filepath.Join(tmpDir, "test.env")

			err := writeDotenvFile(filePath, tc.data)
			require.NoError(t, err)

			if len(tc.expectedLines) == 0 {
				// File should be empty
				content, err := os.ReadFile(filePath)
				require.NoError(t, err)
				require.Empty(t, strings.TrimSpace(string(content)))
			} else {
				// Verify content matches expected lines
				content, err := os.ReadFile(filePath)
				require.NoError(t, err)
				lines := strings.Split(strings.TrimSpace(string(content)), "\n")
				require.Equal(t, tc.expectedLines, lines)
			}
		})
	}
}
