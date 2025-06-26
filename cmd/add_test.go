package cmd

import (
	"bytes"
	"errors"
	"testing"

	"github.com/mozilla-ai/mcpd/v2/internal/printer"

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

func (f *fakeConfig) RemoveServer(name string) error {
	return nil
}

func (f *fakeConfig) ListServers() []config.ServerEntry {
	return []config.ServerEntry{f.entry}
}

type fakeLoader struct {
	cfg *fakeConfig
	err error
}

func (f *fakeLoader) Load(path string) (config.Modifier, error) {
	return f.cfg, f.err
}

type fakePrinter struct {
	printed packages.Package
	opts    []printer.PackagePrinterOption
	err     error
}

func (f *fakePrinter) PrintPackage(pkg packages.Package) error {
	f.printed = pkg
	return f.err
}

func (f *fakePrinter) SetOptions(opt ...printer.PackagePrinterOption) error {
	opts := make([]printer.PackagePrinterOption, 0, len(opt))
	for i, o := range opt {
		opts[i] = o
	}
	f.opts = opts

	return nil
}

type fakeRegistry struct {
	pkg packages.Package
	err error
}

func (f *fakeRegistry) Resolve(_ string, _ ...options.ResolveOption) (packages.Package, error) {
	return f.pkg, f.err
}

func (f *fakeRegistry) Search(_ string, _ map[string]string, _ ...options.SearchOption) ([]packages.Package, error) {
	return []packages.Package{f.pkg}, f.err
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

func TestAddCmd_Success(t *testing.T) {
	cfg := &fakeConfig{}
	pkg := packages.Package{
		ID:      "server1",
		Name:    "Server1",
		Tools:   []string{"toolA", "toolB"},
		Version: "1.2.3",
		InstallationDetails: map[runtime.Runtime]packages.Installation{
			runtime.UVX: {
				Package:     "mcp-server-1",
				Recommended: true,
			},
		},
	}
	buf := new(bytes.Buffer)

	cmdObj, err := NewAddCmd(
		&cmd.BaseCmd{},
		cmdopts.WithConfigLoader(&fakeLoader{cfg: cfg}),
		cmdopts.WithRegistryBuilder(&fakeBuilder{reg: &fakeRegistry{pkg: pkg}}),
		cmdopts.WithPrinter(&fakePrinter{}),
	)
	require.NoError(t, err)
	require.NotNil(t, cmdObj)

	cmdObj.SetOut(buf)
	cmdObj.SetArgs([]string{"mcp-server-1", "--version=1.2.3", "--tool=toolA", "--runtime=uvx"})

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
		cmdopts.WithPrinter(&fakePrinter{}),
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
		cmdopts.WithPrinter(&fakePrinter{}),
	)
	require.NoError(t, err)

	cmdObj.SetArgs([]string{"server1"})
	err = cmdObj.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "registry error")
}

func TestAddCmd_BasicServerAdd(t *testing.T) {
	output := &bytes.Buffer{}

	pkg := packages.Package{
		ID:      "testserver",
		Name:    "testserver",
		Version: "latest",
		Tools:   []string{"tool1", "tool2", "tool3"},
		InstallationDetails: map[runtime.Runtime]packages.Installation{
			"uvx": {
				Package:     "mcp-server-testserver",
				Recommended: true,
			},
		},
	}

	cfg := &fakeConfig{}
	fp := &fakePrinter{}
	cmdObj, err := NewAddCmd(
		&cmd.BaseCmd{},
		cmdopts.WithConfigLoader(&fakeLoader{cfg: cfg}),
		cmdopts.WithRegistryBuilder(&fakeBuilder{reg: &fakeRegistry{pkg: pkg}}),
		cmdopts.WithPrinter(fp),
	)
	require.NoError(t, err)

	cmdObj.SetOut(output)
	cmdObj.SetErr(output)
	cmdObj.SetArgs([]string{"testserver"})

	// Run the command
	err = cmdObj.Execute()
	require.NoError(t, err)

	// Output assertions
	outStr := output.String()
	assert.Contains(t, outStr, "✓ Added server 'testserver'")
	assert.Contains(t, outStr, "version: latest")

	// Config assertions
	require.True(t, cfg.addCalled)
	assert.Equal(t, "testserver", cfg.entry.Name)
	assert.Equal(t, "uvx::mcp-server-testserver@latest", cfg.entry.Package)
	assert.ElementsMatch(t, []string{"tool1", "tool2", "tool3"}, cfg.entry.Tools)
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
		name                 string
		installations        map[runtime.Runtime]packages.Installation
		supportedRuntimes    []runtime.Runtime
		pkgName              string
		pkgID                string
		availableTools       []string
		requestedTools       []string
		requestedRuntime     runtime.Runtime
		isErrorExpected      bool
		expectedErrorMessage string
		expectedPackageValue string
	}{
		{
			name: "Recommended runtime present",
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
			expectedPackageValue: "uvx::mcp-server-time@latest",
		},
		{
			name: "Fallback to supported priority",
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
			expectedPackageValue: "docker::mcp/time@latest",
		},
		{
			name: "Missing runtime-specific package name should error",
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
			isErrorExpected:      true,
			expectedErrorMessage: "installation package name is missing for runtime 'docker'",
		},
		{
			name: "Requested tool not found",
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
			isErrorExpected:      true,
			expectedErrorMessage: "error matching requested tools: none of the requested values were found",
		},
		{
			name: "No supported runtime found",
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
			isErrorExpected:      true,
			expectedErrorMessage: "error selecting runtime from available installations: no supported runtimes found",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			pkg := packages.Package{
				ID:                  tc.pkgID,
				Name:                tc.pkgName,
				Tools:               tc.availableTools,
				InstallationDetails: tc.installations,
			}

			entry, err := parseServerEntry(pkg, tc.requestedRuntime, tc.requestedTools, tc.supportedRuntimes)

			if tc.isErrorExpected {
				require.Error(t, err)
				require.EqualError(t, err, tc.expectedErrorMessage)
			} else {
				require.NoError(t, err)
				expected := config.ServerEntry{
					Name:    tc.pkgID,
					Package: tc.expectedPackageValue,
					Tools:   tc.requestedTools,
				}
				require.Equal(t, expected, entry)
			}
		})
	}
}
