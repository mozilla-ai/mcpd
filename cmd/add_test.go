package cmd

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mozilla-ai/mcpd/v2/internal/cmd"
	cmdopts "github.com/mozilla-ai/mcpd/v2/internal/cmd/options"
	"github.com/mozilla-ai/mcpd/v2/internal/config"
	"github.com/mozilla-ai/mcpd/v2/internal/packages"
	"github.com/mozilla-ai/mcpd/v2/internal/registry"
	"github.com/mozilla-ai/mcpd/v2/internal/registry/options"
	"github.com/mozilla-ai/mcpd/v2/internal/runtime"
)

type fakeConfig struct {
	addCalled bool
	entry     config.ServerEntry
}

func (f *fakeConfig) AddServer(entry config.ServerEntry) error {
	f.addCalled = true
	f.entry = entry
	return nil
}

func (f *fakeConfig) RemoveServer(_ string) error {
	return nil
}

func (f *fakeConfig) ListServers() []config.ServerEntry {
	return []config.ServerEntry{f.entry}
}

type fakeLoader struct {
	cfg *fakeConfig
	err error
}

func (f *fakeLoader) Load(_ string) (config.Modifier, error) {
	return f.cfg, f.err
}

type fakeRegistry struct {
	pkg packages.Server
	err error
}

func (f *fakeRegistry) Resolve(_ string, _ ...options.ResolveOption) (packages.Server, error) {
	return f.pkg, f.err
}

func (f *fakeRegistry) Search(_ string, _ map[string]string, _ ...options.SearchOption) ([]packages.Server, error) {
	return []packages.Server{f.pkg}, f.err
}

func (f *fakeRegistry) ID() string {
	return "fake"
}

type fakeBuilder struct {
	reg registry.PackageProvider
	err error
}

func (f *fakeBuilder) Build() (registry.PackageProvider, error) {
	return f.reg, f.err
}

// testIntPtr returns a pointer to an int
func testIntPtr(t *testing.T, i int) *int {
	t.Helper()
	return &i
}

func TestAddCmd_Success(t *testing.T) {
	cfg := &fakeConfig{}
	pkg := packages.Server{
		ID:   "server1",
		Name: "Server1",
		Tools: []packages.Tool{
			{Name: "toolA"},
			{Name: "toolB"},
		},
		Installations: map[runtime.Runtime]packages.Installation{
			runtime.UVX: {
				Runtime:     "uvx",
				Package:     "mcp-server-1",
				Version:     "1.2.3",
				Recommended: true,
			},
		},
	}
	buf := new(bytes.Buffer)

	cmdObj, err := NewAddCmd(
		&cmd.BaseCmd{},
		cmdopts.WithConfigLoader(&fakeLoader{cfg: cfg}),
		cmdopts.WithRegistryBuilder(&fakeBuilder{reg: &fakeRegistry{pkg: pkg}}),
	)
	require.NoError(t, err)
	require.NotNil(t, cmdObj)

	cmdObj.SetOut(buf)
	cmdObj.SetArgs([]string{"server1", "--version=1.2.3", "--tool=toolA", "--runtime=uvx"})

	err = cmdObj.Execute()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "✓ Added server")
	assert.True(t, cfg.addCalled)
	assert.Equal(t, "server1", cfg.entry.Name)
	assert.Equal(t, "uvx::mcp-server-1@1.2.3", cfg.entry.Package)
}

func TestAddCmd_MissingArgs(t *testing.T) {
	cmdObj, err := NewAddCmd(&cmd.BaseCmd{},
		cmdopts.WithConfigLoader(&fakeLoader{}),
		cmdopts.WithRegistryBuilder(&fakeBuilder{}),
	)
	require.NoError(t, err)

	cmdObj.SetArgs([]string{}) // No arguments

	err = cmdObj.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "server name is required")
}

func TestAddCmd_RegistryFails(t *testing.T) {
	cmdObj, err := NewAddCmd(&cmd.BaseCmd{},
		cmdopts.WithConfigLoader(&fakeLoader{}),
		cmdopts.WithRegistryBuilder(&fakeBuilder{err: errors.New("registry error")}),
	)
	require.NoError(t, err)

	cmdObj.SetArgs([]string{"server1"})
	err = cmdObj.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "registry error")
}

func TestAddCmd_BasicServerAdd(t *testing.T) {
	o := &bytes.Buffer{}

	pkg := packages.Server{
		ID:   "testserver",
		Name: "testserver",
		Tools: []packages.Tool{
			{Name: "tool1"},
			{Name: "tool2"},
			{Name: "tool3"},
		},
		Installations: map[runtime.Runtime]packages.Installation{
			"uvx": {
				Runtime:     "uvx",
				Package:     "mcp-server-testserver",
				Version:     "latest",
				Recommended: true,
			},
		},
	}

	cfg := &fakeConfig{}
	cmdObj, err := NewAddCmd(
		&cmd.BaseCmd{},
		cmdopts.WithConfigLoader(&fakeLoader{cfg: cfg}),
		cmdopts.WithRegistryBuilder(&fakeBuilder{reg: &fakeRegistry{pkg: pkg}}),
	)
	require.NoError(t, err)

	cmdObj.SetOut(o)
	cmdObj.SetErr(o)
	cmdObj.SetArgs([]string{"testserver"})

	// Run the command
	err = cmdObj.Execute()
	require.NoError(t, err)

	// Output assertions
	outStr := o.String()
	assert.Contains(t, outStr, "✓ Added server 'testserver'")
	assert.Contains(t, outStr, "version: latest")

	// Config assertions
	require.True(t, cfg.addCalled)
	assert.Equal(t, "testserver", cfg.entry.Name)
	assert.Equal(t, "uvx::mcp-server-testserver@latest", cfg.entry.Package)
	assert.ElementsMatch(t, []string{"tool1", "tool2", "tool3"}, cfg.entry.Tools)
}

func TestAddCmd_ServerWithArguments(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                        string
		pkg                         packages.Server
		expectedRequiredEnvs        []string
		expectedRequiredPositionals []string
		expectedRequiredValues      []string
		expectedRequiredBools       []string
	}{
		{
			name: "server with all argument types",
			pkg: packages.Server{
				ID:   "github-server",
				Name: "GitHub Server",
				Tools: []packages.Tool{
					{Name: "create_repo"},
					{Name: "list_repos"},
				},
				Installations: map[runtime.Runtime]packages.Installation{
					runtime.UVX: {
						Runtime:     "uvx",
						Package:     "mcp-server-github",
						Version:     "1.0.0",
						Recommended: true,
					},
				},
				Arguments: packages.Arguments{
					"GITHUB_TOKEN": {VariableType: packages.VariableTypeEnv, Required: true},
					"DEBUG_MODE":   {VariableType: packages.VariableTypeEnv, Required: false},
					"--api-url":    {VariableType: packages.VariableTypeArg, Required: true},
					"--timeout":    {VariableType: packages.VariableTypeArg, Required: false},
					"--verbose":    {VariableType: packages.VariableTypeArgBool, Required: true},
					"--dry-run":    {VariableType: packages.VariableTypeArgBool, Required: false},
				},
			},
			expectedRequiredEnvs:   []string{"GITHUB_TOKEN"},
			expectedRequiredValues: []string{"--api-url"},
			expectedRequiredBools:  []string{"--verbose"},
		},
		{
			name: "server with only env vars",
			pkg: packages.Server{
				ID:   "db-server",
				Name: "Database Server",
				Tools: []packages.Tool{
					{Name: "query"},
				},
				Installations: map[runtime.Runtime]packages.Installation{
					runtime.UVX: {
						Runtime:     "uvx",
						Package:     "mcp-server-db",
						Version:     "2.0.0",
						Recommended: true,
					},
				},
				Arguments: packages.Arguments{
					"DB_HOST": {VariableType: packages.VariableTypeEnv, Required: true},
					"DB_PORT": {VariableType: packages.VariableTypeEnv, Required: true},
					"DB_NAME": {VariableType: packages.VariableTypeEnv, Required: false},
					"DB_USER": {VariableType: packages.VariableTypeEnv, Required: false},
				},
			},
			expectedRequiredEnvs: []string{"DB_HOST", "DB_PORT"},
		},
		{
			name: "server with mixed value and bool args",
			pkg: packages.Server{
				ID:   "api-server",
				Name: "API Server",
				Tools: []packages.Tool{
					{Name: "call_api"},
				},
				Installations: map[runtime.Runtime]packages.Installation{
					runtime.UVX: {
						Runtime:     "uvx",
						Package:     "mcp-server-api",
						Version:     "3.0.0",
						Recommended: true,
					},
				},
				Arguments: packages.Arguments{
					"--endpoint":     {VariableType: packages.VariableTypeArg, Required: true},
					"--api-key":      {VariableType: packages.VariableTypeArg, Required: true},
					"--format":       {VariableType: packages.VariableTypeArg, Required: false},
					"--enable-cache": {VariableType: packages.VariableTypeArgBool, Required: true},
					"--debug":        {VariableType: packages.VariableTypeArgBool, Required: false},
				},
			},
			expectedRequiredValues: []string{"--endpoint", "--api-key"},
			expectedRequiredBools:  []string{"--enable-cache"},
		},
		{
			name: "server with no required arguments",
			pkg: packages.Server{
				ID:   "simple-server",
				Name: "Simple Server",
				Tools: []packages.Tool{
					{Name: "hello"},
				},
				Installations: map[runtime.Runtime]packages.Installation{
					runtime.UVX: {
						Runtime:     "uvx",
						Package:     "mcp-server-simple",
						Version:     "1.0.0",
						Recommended: true,
					},
				},
				Arguments: packages.Arguments{
					"OPTIONAL_ENV":     {VariableType: packages.VariableTypeEnv, Required: false},
					"--optional-flag":  {VariableType: packages.VariableTypeArgBool, Required: false},
					"--optional-value": {VariableType: packages.VariableTypeArg, Required: false},
				},
			},
		},
		{
			name: "server all kinds of required args and envs",
			pkg: packages.Server{
				ID:   "simple-server",
				Name: "Simple Server",
				Tools: []packages.Tool{
					{
						Name: "hello",
					},
				},
				Installations: map[runtime.Runtime]packages.Installation{
					runtime.UVX: {
						Runtime:     "uvx",
						Package:     "mcp-server-simple",
						Version:     "1.0.0",
						Recommended: true,
					},
				},
				Arguments: packages.Arguments{
					"REQUIRED_ENV": {
						VariableType: packages.VariableTypeEnv,
						Required:     true,
					},
					"OPTIONAL_ENV": {
						VariableType: packages.VariableTypeEnv,
						Required:     false,
					},
					"RequiredPositional1": {
						VariableType: packages.VariableTypeArgPositional,
						Position:     testIntPtr(t, 1),
						Required:     true,
					},
					"OptionalPositional2": {
						VariableType: packages.VariableTypeArgPositional,
						Position:     testIntPtr(t, 2),
						Required:     false,
					},
					"--required-value": {
						VariableType: packages.VariableTypeArg,
						Required:     true,
					},
					"--optional-value": {
						VariableType: packages.VariableTypeArg,
						Required:     false,
					},
					"--required-flag": {
						VariableType: packages.VariableTypeArgBool,
						Required:     true,
					},
					"--optional-flag": {
						VariableType: packages.VariableTypeArgBool,
						Required:     false,
					},
				},
			},
			expectedRequiredEnvs:        []string{"REQUIRED_ENV"},
			expectedRequiredPositionals: []string{"RequiredPositional1"},
			expectedRequiredValues:      []string{"--required-value"},
			expectedRequiredBools:       []string{"--required-flag"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cfg := &fakeConfig{}
			cmdObj, err := NewAddCmd(
				&cmd.BaseCmd{},
				cmdopts.WithConfigLoader(&fakeLoader{cfg: cfg}),
				cmdopts.WithRegistryBuilder(&fakeBuilder{reg: &fakeRegistry{pkg: tc.pkg}}),
			)
			require.NoError(t, err)

			cmdObj.SetOut(io.Discard)
			cmdObj.SetErr(io.Discard)
			cmdObj.SetArgs([]string{tc.pkg.ID})

			err = cmdObj.Execute()
			require.NoError(t, err)

			// Verify config was called with correct arguments
			require.True(t, cfg.addCalled)
			assert.Equal(t, tc.pkg.ID, cfg.entry.Name)
			assert.ElementsMatch(t, tc.expectedRequiredEnvs, cfg.entry.RequiredEnvVars)
			assert.ElementsMatch(t, tc.expectedRequiredPositionals, cfg.entry.RequiredPositionalArgs)
			assert.ElementsMatch(t, tc.expectedRequiredValues, cfg.entry.RequiredValueArgs)
			assert.ElementsMatch(t, tc.expectedRequiredBools, cfg.entry.RequiredBoolArgs)
		})
	}
}

func TestSelectRuntime(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		installations    map[runtime.Runtime]packages.Installation
		requestedRuntime runtime.Runtime
		supported        []runtime.Runtime
		expectedRuntime  runtime.Runtime
		expectErr        bool
	}{
		{
			name: "selects recommended runtime",
			installations: map[runtime.Runtime]packages.Installation{
				runtime.UVX:    {Recommended: false},
				runtime.Docker: {Recommended: true},
			},
			supported:       []runtime.Runtime{runtime.UVX, runtime.Docker},
			expectedRuntime: runtime.Docker,
		},
		{
			name: "falls back to supported non-recommended runtime",
			installations: map[runtime.Runtime]packages.Installation{
				runtime.UVX: {Recommended: false},
			},
			supported:       []runtime.Runtime{runtime.UVX},
			expectedRuntime: runtime.UVX,
		},
		{
			name: "selects first supported runtime when none recommended",
			installations: map[runtime.Runtime]packages.Installation{
				runtime.Python: {Recommended: false},
				runtime.UVX:    {Recommended: false},
			},
			supported:       []runtime.Runtime{runtime.UVX, runtime.Python},
			expectedRuntime: runtime.UVX,
		},
		{
			name: "returns error when no supported runtimes",
			installations: map[runtime.Runtime]packages.Installation{
				runtime.UVX: {Recommended: true},
			},
			supported: []runtime.Runtime{runtime.Docker},
			expectErr: true,
		},
		{
			name:          "returns error when installations empty",
			installations: map[runtime.Runtime]packages.Installation{},
			supported:     []runtime.Runtime{runtime.Docker},
			expectErr:     true,
		},
		{
			name: "returns error when supported list is empty",
			installations: map[runtime.Runtime]packages.Installation{
				runtime.UVX: {Recommended: true},
			},
			supported: []runtime.Runtime{},
			expectErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := selectRuntime(tc.installations, tc.requestedRuntime, tc.supported)
			if tc.expectErr {
				require.Error(t, err)
				require.EqualError(t, err, "no supported runtimes found")
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expectedRuntime, got)
			}
		})
	}
}

func TestParseServerEntry(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                        string
		installations               map[runtime.Runtime]packages.Installation
		supportedRuntimes           []runtime.Runtime
		pkgName                     string
		pkgID                       string
		availableTools              []string
		requestedTools              []string
		requestedRuntime            runtime.Runtime
		arguments                   packages.Arguments
		isErrorExpected             bool
		expectedErrorMessage        string
		expectedPackageValue        string
		expectedRequiredEnvs        []string
		expectedRequiredPositionals []string
		expectedRequiredValues      []string
		expectedRequiredBools       []string
	}{
		{
			name: "basic server with no arguments",
			installations: map[runtime.Runtime]packages.Installation{
				runtime.UVX: {
					Package:     "mcp-server-time",
					Recommended: true,
				},
			},
			supportedRuntimes:    []runtime.Runtime{runtime.UVX, runtime.Docker},
			pkgName:              "time",
			pkgID:                "time",
			availableTools:       []string{"get_current_time", "convert_time"},
			requestedTools:       []string{"get_current_time"},
			arguments:            packages.Arguments{},
			expectedPackageValue: "uvx::mcp-server-time@latest",
		},
		{
			name: "server with all argument types",
			installations: map[runtime.Runtime]packages.Installation{
				runtime.UVX: {
					Package:     "mcp-server-github",
					Recommended: true,
				},
			},
			supportedRuntimes: []runtime.Runtime{runtime.UVX},
			pkgName:           "github",
			pkgID:             "github",
			availableTools:    []string{"create_repo", "list_repos"},
			requestedTools:    []string{"create_repo"},
			arguments: packages.Arguments{
				"GITHUB_TOKEN": {VariableType: packages.VariableTypeEnv, Required: true},
				"DEBUG_MODE":   {VariableType: packages.VariableTypeEnv, Required: false},
				"--api-url":    {VariableType: packages.VariableTypeArg, Required: true},
				"--timeout":    {VariableType: packages.VariableTypeArg, Required: false},
				"--verbose":    {VariableType: packages.VariableTypeArgBool, Required: true},
				"--dry-run":    {VariableType: packages.VariableTypeArgBool, Required: false},
			},
			expectedPackageValue:   "uvx::mcp-server-github@latest",
			expectedRequiredEnvs:   []string{"GITHUB_TOKEN"},
			expectedRequiredValues: []string{"--api-url"},
			expectedRequiredBools:  []string{"--verbose"},
		},
		{
			name: "server with only required env vars",
			installations: map[runtime.Runtime]packages.Installation{
				runtime.UVX: {
					Package:     "mcp-server-database",
					Recommended: true,
				},
			},
			supportedRuntimes: []runtime.Runtime{runtime.UVX},
			pkgName:           "database",
			pkgID:             "database",
			availableTools:    []string{"query", "insert"},
			requestedTools:    []string{"query"},
			arguments: packages.Arguments{
				"DB_HOST": {VariableType: packages.VariableTypeEnv, Required: true},
				"DB_PORT": {VariableType: packages.VariableTypeEnv, Required: true},
				"DB_NAME": {VariableType: packages.VariableTypeEnv, Required: false},
			},
			expectedPackageValue: "uvx::mcp-server-database@latest",
			expectedRequiredEnvs: []string{"DB_HOST", "DB_PORT"},
		},
		{
			name: "server with only required value args",
			installations: map[runtime.Runtime]packages.Installation{
				runtime.UVX: {
					Package:     "mcp-server-api",
					Recommended: true,
				},
			},
			supportedRuntimes: []runtime.Runtime{runtime.UVX},
			pkgName:           "api",
			pkgID:             "api",
			availableTools:    []string{"call", "status"},
			requestedTools:    []string{"call"},
			arguments: packages.Arguments{
				"--endpoint": {VariableType: packages.VariableTypeArg, Required: true},
				"--api-key":  {VariableType: packages.VariableTypeArg, Required: true},
				"--format":   {VariableType: packages.VariableTypeArg, Required: false},
			},
			expectedPackageValue:   "uvx::mcp-server-api@latest",
			expectedRequiredValues: []string{"--endpoint", "--api-key"},
		},
		{
			name: "server with positional and value args",
			installations: map[runtime.Runtime]packages.Installation{
				runtime.UVX: {
					Package:     "mcp-server-files",
					Recommended: true,
				},
			},
			supportedRuntimes: []runtime.Runtime{runtime.UVX},
			pkgName:           "files",
			pkgID:             "files",
			availableTools:    []string{"read", "write"},
			requestedTools:    []string{"read"},
			arguments: packages.Arguments{
				"path": {
					VariableType: packages.VariableTypeArgPositional,
					Position:     testIntPtr(t, 1),
					Required:     true,
				},
				"mode": {
					VariableType: packages.VariableTypeArgPositional,
					Position:     testIntPtr(t, 2),
					Required:     true,
				},
				"--format":   {VariableType: packages.VariableTypeArg, Required: true},
				"--encoding": {VariableType: packages.VariableTypeArg, Required: false},
			},
			expectedPackageValue:        "uvx::mcp-server-files@latest",
			expectedRequiredPositionals: []string{"path", "mode"},
			expectedRequiredValues:      []string{"--format"},
		},
		{
			name: "server with only required bool args",
			installations: map[runtime.Runtime]packages.Installation{
				runtime.UVX: {
					Package:     "mcp-server-logger",
					Recommended: true,
				},
			},
			supportedRuntimes: []runtime.Runtime{runtime.UVX},
			pkgName:           "logger",
			pkgID:             "logger",
			availableTools:    []string{"log", "clear"},
			requestedTools:    []string{"log"},
			arguments: packages.Arguments{
				"--enable-debug": {VariableType: packages.VariableTypeArgBool, Required: true},
				"--enable-trace": {VariableType: packages.VariableTypeArgBool, Required: true},
				"--quiet":        {VariableType: packages.VariableTypeArgBool, Required: false},
			},
			expectedPackageValue:  "uvx::mcp-server-logger@latest",
			expectedRequiredBools: []string{"--enable-debug", "--enable-trace"},
		},
		{
			name: "fallback to supported priority",
			installations: map[runtime.Runtime]packages.Installation{
				runtime.UVX: {
					Package:     "mcp-server-time",
					Recommended: false,
				},
				runtime.Docker: {
					Package:     "mcp/time",
					Recommended: false,
				},
			},
			supportedRuntimes:    []runtime.Runtime{runtime.Docker, runtime.UVX},
			pkgName:              "time",
			pkgID:                "time",
			availableTools:       []string{"get_current_time", "convert_time"},
			requestedTools:       []string{"get_current_time"},
			arguments:            packages.Arguments{},
			expectedPackageValue: "docker::mcp/time@latest",
		},
		{
			name: "missing runtime-specific package name should error",
			installations: map[runtime.Runtime]packages.Installation{
				runtime.Docker: {
					Package:     "", // This is bad.
					Recommended: true,
				},
			},
			supportedRuntimes:    []runtime.Runtime{runtime.Docker},
			pkgName:              "time",
			pkgID:                "time",
			availableTools:       []string{"convert_time"},
			requestedTools:       []string{"convert_time"},
			arguments:            packages.Arguments{},
			isErrorExpected:      true,
			expectedErrorMessage: "installation server name is missing for runtime 'docker'",
		},
		{
			name: "requested tool not found",
			installations: map[runtime.Runtime]packages.Installation{
				runtime.UVX: {
					Package:     "mcp-server-time",
					Recommended: true,
				},
			},
			supportedRuntimes:    []runtime.Runtime{runtime.UVX},
			pkgName:              "time",
			pkgID:                "time",
			availableTools:       []string{"get_current_time"},
			requestedTools:       []string{"missing_tool"},
			arguments:            packages.Arguments{},
			isErrorExpected:      true,
			expectedErrorMessage: "error matching requested tools: none of the requested values were found",
		},
		{
			name: "no supported runtime found",
			installations: map[runtime.Runtime]packages.Installation{
				"python": {
					Package:     "mcp_server_time",
					Recommended: true,
				},
			},
			supportedRuntimes:    []runtime.Runtime{runtime.UVX, runtime.Docker},
			pkgName:              "time",
			pkgID:                "time",
			availableTools:       []string{"get_current_time"},
			requestedTools:       []string{"get_current_time"},
			arguments:            packages.Arguments{},
			isErrorExpected:      true,
			expectedErrorMessage: "error selecting runtime from available installations: no supported runtimes found",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tools := make(packages.Tools, len(tc.availableTools))
			for i, tool := range tc.availableTools {
				tools[i] = packages.Tool{Name: tool}
			}

			pkg := packages.Server{
				ID:            tc.pkgID,
				Name:          tc.pkgName,
				Tools:         tools,
				Installations: tc.installations,
				Arguments:     tc.arguments,
			}

			opts := serverEntryOptions{
				Runtime:           tc.requestedRuntime,
				Tools:             tc.requestedTools,
				SupportedRuntimes: tc.supportedRuntimes,
				AllowDeprecated:   false,
			}
			entry, err := parseServerEntry(pkg, opts)

			if tc.isErrorExpected {
				require.Error(t, err)
				require.EqualError(t, err, tc.expectedErrorMessage)
			} else {
				require.NoError(t, err)
				expected := config.ServerEntry{
					Name:                   tc.pkgID,
					Package:                tc.expectedPackageValue,
					Tools:                  tc.requestedTools,
					RequiredEnvVars:        tc.expectedRequiredEnvs,
					RequiredPositionalArgs: tc.expectedRequiredPositionals,
					RequiredValueArgs:      tc.expectedRequiredValues,
					RequiredBoolArgs:       tc.expectedRequiredBools,
				}
				require.Equal(t, expected.Name, entry.Name)
				require.Equal(t, expected.Package, entry.Package)
				require.ElementsMatch(t, expected.Tools, entry.Tools)
				require.ElementsMatch(t, expected.RequiredEnvVars, entry.RequiredEnvVars)
				require.ElementsMatch(t, expected.RequiredValueArgs, entry.RequiredValueArgs)
				require.ElementsMatch(t, expected.RequiredBoolArgs, entry.RequiredBoolArgs)
			}
		})
	}
}
