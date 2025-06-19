package cmd

import (
	"bytes"
	"errors"
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
	err     error
}

func (f *fakePrinter) PrintPackage(pkg packages.Package) error {
	f.printed = pkg
	return f.err
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
		Name:    "server1",
		Tools:   []string{"toolA", "toolB"},
		Version: "1.2.3",
		InstallationDetails: map[runtime.Runtime]packages.Installation{
			runtime.UVX: {Recommended: true},
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
	cmdObj.SetArgs([]string{"server1", "--version=1.2.3", "--tool=toolA", "--runtime=uvx"})

	err = cmdObj.Execute()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "✓ Added server")
	assert.True(t, cfg.addCalled)
	assert.Equal(t, "server1", cfg.entry.Name)
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
		cmdopts.WithRegistryBuilder(&fakeBuilder{err: errors.New("fail")}),
		cmdopts.WithPrinter(&fakePrinter{}),
	)
	require.NoError(t, err)

	cmdObj.SetArgs([]string{"server1"})
	err = cmdObj.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "fail")
}

func TestAddCmd_BasicServerAdd(t *testing.T) {
	output := &bytes.Buffer{}

	pkg := packages.Package{
		ID:      "testserver",
		Name:    "testserver",
		Version: "latest",
		Tools:   []string{"tool1", "tool2", "tool3"},
		InstallationDetails: map[runtime.Runtime]packages.Installation{
			"uvx": {Recommended: true},
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
	assert.Equal(t, "uvx::testserver@latest", cfg.entry.Package)
	assert.ElementsMatch(t, []string{"tool1", "tool2", "tool3"}, cfg.entry.Tools)
}
