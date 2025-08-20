package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"github.com/mozilla-ai/mcpd/v2/internal/cmd"
	cmdopts "github.com/mozilla-ai/mcpd/v2/internal/cmd/options"
	"github.com/mozilla-ai/mcpd/v2/internal/cmd/output"
	"github.com/mozilla-ai/mcpd/v2/internal/packages"
	"github.com/mozilla-ai/mcpd/v2/internal/registry/options"
	"github.com/mozilla-ai/mcpd/v2/internal/runtime"
)

// fakeRegistryMultiple supports returning multiple packages
type fakeRegistryMultiple struct {
	packages []packages.Server
	err      error
}

func (f *fakeRegistryMultiple) Resolve(_ string, _ ...options.ResolveOption) (packages.Server, error) {
	if len(f.packages) > 0 {
		return f.packages[0], f.err
	}
	return packages.Server{}, f.err
}

func (f *fakeRegistryMultiple) Search(
	_ string,
	_ map[string]string,
	_ ...options.SearchOption,
) ([]packages.Server, error) {
	return f.packages, f.err
}

func (f *fakeRegistryMultiple) ID() string {
	return "fake-multiple"
}

// testServer creates a packages.Server with sensible defaults for testing.
func testServer(t *testing.T) packages.Server {
	t.Helper()
	return packages.Server{
		ID:          "test-server",
		Name:        "Test Server",
		Description: "A test server",
		License:     "MIT",
		Source:      "mozilla-ai",
		Tools: []packages.Tool{
			{Name: "test_tool"},
		},
		Installations: packages.Installations{
			runtime.UVX: packages.Installation{
				Runtime: "test-server",
			},
		},
	}
}

func TestSearchCmd_Filters(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setters    func(c *SearchCmd)
		wantFilter map[string]string
	}{
		{
			name:       "nothing set",
			setters:    func(c *SearchCmd) {},
			wantFilter: map[string]string{},
		},
		{
			name: "version and runtime",
			setters: func(c *SearchCmd) {
				c.Version = "v1.2.3"
				c.Runtime = "npx"
			},
			wantFilter: map[string]string{
				"version": "v1.2.3",
				"runtime": "npx",
			},
		},
		{
			name: "tools, tags, categories",
			setters: func(c *SearchCmd) {
				c.Tools = []string{"t1", "t2"}
				c.Tags = []string{"a", "b"}
				c.Categories = []string{"x"}
			},
			wantFilter: map[string]string{
				"tools":      "t1,t2",
				"tags":       "a,b",
				"categories": "x",
			},
		},
		{
			name: "license only",
			setters: func(c *SearchCmd) {
				c.License = "MIT"
			},
			wantFilter: map[string]string{
				"license": "MIT",
			},
		},
		{
			name: "official",
			setters: func(c *SearchCmd) {
				c.IsOfficial = true
				c.License = "MIT"
			},
			wantFilter: map[string]string{
				"isOfficial": "true",
				"license":    "MIT",
			},
		},
		{
			name: "official missing when not true",
			setters: func(c *SearchCmd) {
				c.IsOfficial = false
				c.License = "MIT"
			},
			wantFilter: map[string]string{
				"license": "MIT",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			c := &SearchCmd{}
			tc.setters(c)
			got := c.filters()
			require.Equal(t, tc.wantFilter, got)
		})
	}
}

func TestSearchCmd_NewSearchCmd_Defaults(t *testing.T) {
	t.Parallel()

	base := &cmd.BaseCmd{}
	cmdCobra, err := NewSearchCmd(base)
	require.NoError(t, err)

	// It should be a *cobra.Command
	require.IsType(t, &cobra.Command{}, cmdCobra)

	// Default Format should be text
	flag := cmdCobra.Flags().Lookup("format")
	require.NotNil(t, flag)
	// The default value is stored in SearchCmd.Format, but we can inspect the usage text
	usage := flag.Usage
	require.Contains(t, usage, string(cmd.FormatText))
}

func TestSearchCmd_Run_UnexpectedFormat(t *testing.T) {
	t.Parallel()
	// Create a SearchCmd with an unsupported format
	sc := &SearchCmd{Format: cmd.OutputFormat("bogus")}

	// We need a cobra.Command to satisfy the signature; out writer doesn't matter
	cmdCobra := &cobra.Command{RunE: sc.run}
	cmdCobra.SetOut(io.Discard)

	// Call run directly (args empty)
	err := sc.run(cmdCobra, []string{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "no handler for output format: bogus")
}

func TestSearchCmd_DefaultFormat(t *testing.T) {
	pkg := testServer(t)

	o := new(bytes.Buffer)
	cmdObj, err := NewSearchCmd(
		&cmd.BaseCmd{},
		cmdopts.WithRegistryBuilder(&fakeBuilder{reg: &fakeRegistry{pkg: pkg}}),
	)
	require.NoError(t, err)

	cmdObj.SetOut(o)
	cmdObj.SetArgs([]string{"test-server"})

	err = cmdObj.Execute()
	require.NoError(t, err)

	outStr := o.String()
	require.Contains(t, outStr, "🔎 Registry search results...")
	require.Contains(t, outStr, "🆔 test-server")
	require.Contains(t, outStr, "📦 Found 1 server")
}

func TestSearchCmd_TextFormat(t *testing.T) {
	pkg := testServer(t)

	o := new(bytes.Buffer)
	cmdObj, err := NewSearchCmd(
		&cmd.BaseCmd{},
		cmdopts.WithRegistryBuilder(&fakeBuilder{reg: &fakeRegistry{pkg: pkg}}),
	)
	require.NoError(t, err)

	cmdObj.SetOut(o)
	cmdObj.SetArgs([]string{"test-server", "--format=text"})

	err = cmdObj.Execute()
	require.NoError(t, err)

	outStr := o.String()
	require.Contains(t, outStr, "🔎 Registry search results...")
	require.Contains(t, outStr, "🆔 test-server")
	require.Contains(t, outStr, "📦 Found 1 server")
}

func TestSearchCmd_JSONFormat(t *testing.T) {
	pkg := testServer(t)

	o := new(bytes.Buffer)
	cmdObj, err := NewSearchCmd(
		&cmd.BaseCmd{},
		cmdopts.WithRegistryBuilder(&fakeBuilder{reg: &fakeRegistry{pkg: pkg}}),
	)
	require.NoError(t, err)

	cmdObj.SetOut(o)
	cmdObj.SetArgs([]string{"test-server", "--format=json"})

	err = cmdObj.Execute()
	require.NoError(t, err)

	var result output.ResultsPayload[packages.Server]
	err = json.Unmarshal(o.Bytes(), &result)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Results)
	require.Len(t, result.Results, 1)
	require.Equal(t, "test-server", result.Results[0].ID)
	require.Equal(t, "Test Server", result.Results[0].Name)
	require.Equal(t, "A test server", result.Results[0].Description)
	require.Equal(t, "MIT", result.Results[0].License)
	require.Equal(t, "mozilla-ai", result.Results[0].Source)
	require.Len(t, result.Results[0].Tools, 1)
	require.Equal(t, "test_tool", result.Results[0].Tools[0].Name)
}

func TestSearchCmd_JSONFormat_EmptyResults(t *testing.T) {
	o := new(bytes.Buffer)
	cmdObj, err := NewSearchCmd(
		&cmd.BaseCmd{},
		cmdopts.WithRegistryBuilder(&fakeBuilder{reg: &fakeRegistryMultiple{packages: []packages.Server{}}}),
	)
	require.NoError(t, err)

	cmdObj.SetOut(o)
	cmdObj.SetArgs([]string{"nonexistent", "--format=json"})

	err = cmdObj.Execute()
	require.NoError(t, err)

	result := struct {
		Results []packages.Server `json:"results"`
	}{}
	err = json.Unmarshal(o.Bytes(), &result)
	require.NoError(t, err)

	require.Empty(t, result.Results)
}

func TestSearchCmd_JSONFormat_MultipleResults(t *testing.T) {
	pkg1 := testServer(t)
	pkg1.ID = "server1"
	pkg1.Name = "Server 1"
	pkg1.Description = "First server"
	pkg1.Tools = []packages.Tool{{Name: "tool1"}}

	pkg2 := testServer(t)
	pkg2.ID = "server2"
	pkg2.Name = "Server 2"
	pkg2.Description = "Second server"
	pkg2.License = "Apache-2.0"
	pkg2.Tools = []packages.Tool{{Name: "tool2"}}
	pkg2.Installations = packages.Installations{
		runtime.Docker: packages.Installation{
			Runtime: "test-server",
		},
	}

	fakeReg := &fakeRegistryMultiple{packages: []packages.Server{pkg1, pkg2}}

	out := new(bytes.Buffer)
	cmdObj, err := NewSearchCmd(
		&cmd.BaseCmd{},
		cmdopts.WithRegistryBuilder(&fakeBuilder{reg: fakeReg}),
	)
	require.NoError(t, err)

	cmdObj.SetOut(out)
	cmdObj.SetArgs([]string{"server", "--format=json"})

	err = cmdObj.Execute()
	require.NoError(t, err)

	result := struct {
		Results []packages.Server `json:"results"`
	}{}
	err = json.Unmarshal(out.Bytes(), &result)
	require.NoError(t, err)

	require.Len(t, result.Results, 2)
	require.Equal(t, "server1", result.Results[0].ID)
	require.Equal(t, "server2", result.Results[1].ID)
}

func TestSearchCmd_InvalidFormat(t *testing.T) {
	out := new(bytes.Buffer)
	cmdObj, err := NewSearchCmd(
		&cmd.BaseCmd{},
		cmdopts.WithRegistryBuilder(&fakeBuilder{reg: &fakeRegistry{}}),
	)
	require.NoError(t, err)

	cmdObj.SetOut(out)
	cmdObj.SetArgs([]string{"test-server", "--format=invalid"})

	err = cmdObj.Execute()
	require.Error(t, err)
	require.ErrorContains(t, err, "invalid argument \"invalid\"")
	require.ErrorContains(t, err, "must be one of json, text, yaml")
}

func TestSearchCmd_CaseInsensitiveFormat(t *testing.T) {
	pkg := testServer(t)

	testCases := []struct {
		name       string
		format     string
		expectJSON bool
		shouldFail bool
	}{
		{"uppercase JSON", "JSON", true, false},
		{"uppercase TEXT", "TEXT", false, false},
		{"mixed case Json", "Json", true, false},
		{"mixed case Text", "Text", false, false},
		{"with spaces json", "  json  ", true, false},
		{"with spaces text", "  text  ", false, false},
		{"invalid format", "XML", false, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			o := new(bytes.Buffer)
			cmdObj, err := NewSearchCmd(
				&cmd.BaseCmd{},
				cmdopts.WithRegistryBuilder(&fakeBuilder{reg: &fakeRegistry{pkg: pkg}}),
			)
			require.NoError(t, err)

			cmdObj.SetOut(o)
			cmdObj.SetArgs([]string{"test-server", fmt.Sprintf("--format=%s", tc.format)})

			err = cmdObj.Execute()

			if tc.shouldFail {
				require.Error(t, err)
				require.ErrorContains(t, err, "invalid argument")
				require.ErrorContains(t, err, "invalid format ")
				return
			}

			require.NoError(t, err)

			if tc.expectJSON {
				result := struct {
					Results []packages.Server `json:"results"`
				}{}
				err = json.Unmarshal(o.Bytes(), &result)
				require.NoError(t, err)
				require.Len(t, result.Results, 1)
				require.Equal(t, "test-server", result.Results[0].ID)
			} else {
				outStr := o.String()
				require.Contains(t, outStr, "🔎 Registry search results...")
				require.Contains(t, outStr, "🆔 test-server")
				require.Contains(t, outStr, "📦 Found 1 server")
			}
		})
	}
}

func TestSearchCmd_JSONFormat_RegistryError(t *testing.T) {
	o := new(bytes.Buffer)
	cmdObj, err := NewSearchCmd(
		&cmd.BaseCmd{},
		cmdopts.WithRegistryBuilder(&fakeBuilder{err: errors.New("registry build failed")}),
	)
	require.NoError(t, err)

	cmdObj.SetOut(o)
	cmdObj.SetArgs([]string{"test-server", "--format=json"})
	err = cmdObj.Execute()
	require.NoError(t, err)

	result := struct {
		Error string `json:"error"`
	}{}
	err = json.Unmarshal(o.Bytes(), &result)
	require.NoError(t, err)

	require.Equal(t, "registry build failed", result.Error)
}

func TestSearchCmd_JSONFormat_SearchError(t *testing.T) {
	o := new(bytes.Buffer)
	cmdObj, err := NewSearchCmd(
		&cmd.BaseCmd{},
		cmdopts.WithRegistryBuilder(&fakeBuilder{reg: &fakeRegistry{err: errors.New("search failed")}}),
	)
	require.NoError(t, err)

	cmdObj.SetOut(o)
	cmdObj.SetArgs([]string{"test-server", "--format=json"})

	err = cmdObj.Execute()
	require.NoError(t, err)

	result := struct {
		Error string `json:"error"`
	}{}
	err = json.Unmarshal(o.Bytes(), &result)
	require.NoError(t, err)

	require.Equal(t, "search failed", result.Error)
}

func TestSearchCmd_TextFormat_NoResults(t *testing.T) {
	fakeReg := &fakeRegistryMultiple{packages: []packages.Server{}}

	o := new(bytes.Buffer)
	cmdObj, err := NewSearchCmd(
		&cmd.BaseCmd{},
		cmdopts.WithRegistryBuilder(&fakeBuilder{reg: fakeReg}),
	)
	require.NoError(t, err)

	cmdObj.SetOut(o)
	cmdObj.SetArgs([]string{"nonexistent", "--format=text"})

	err = cmdObj.Execute()
	require.NoError(t, err)

	outStr := o.String()
	require.Contains(t, outStr, "No items found")
}

func TestSearchCmd_FlagsWithJSONFormat(t *testing.T) {
	pkg := testServer(t)

	o := new(bytes.Buffer)
	cmdObj, err := NewSearchCmd(
		&cmd.BaseCmd{},
		cmdopts.WithRegistryBuilder(&fakeBuilder{reg: &fakeRegistry{pkg: pkg}}),
	)
	require.NoError(t, err)

	cmdObj.SetOut(o)
	cmdObj.SetArgs([]string{"test-server", "--format=json", "--runtime=uvx", "--license=MIT"})

	err = cmdObj.Execute()
	require.NoError(t, err)

	result := struct {
		Results []packages.Server `json:"results"`
	}{}
	err = json.Unmarshal(o.Bytes(), &result)
	require.NoError(t, err)

	require.Len(t, result.Results, 1)
	require.Equal(t, "test-server", result.Results[0].ID)
}

func TestSearchCmd_WildcardSearch(t *testing.T) {
	pkg := testServer(t)

	o := new(bytes.Buffer)
	cmdObj, err := NewSearchCmd(
		&cmd.BaseCmd{},
		cmdopts.WithRegistryBuilder(&fakeBuilder{reg: &fakeRegistry{pkg: pkg}}),
	)
	require.NoError(t, err)

	cmdObj.SetOut(o)
	cmdObj.SetArgs([]string{"--format=json"}) // No search term, should use wildcard

	err = cmdObj.Execute()
	require.NoError(t, err)

	result := struct {
		Results []packages.Server `json:"results"`
	}{}
	err = json.Unmarshal(o.Bytes(), &result)
	require.NoError(t, err)

	require.Len(t, result.Results, 1)
	require.Equal(t, "test-server", result.Results[0].ID)
}

func TestParseOutputFormat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		input          string
		expectedFormat cmd.OutputFormat
		expectError    bool
	}{
		{"lowercase text", "text", cmd.FormatText, false},
		{"uppercase TEXT", "TEXT", cmd.FormatText, false},
		{"mixed case Text", "Text", cmd.FormatText, false},
		{"with spaces text", "  text  ", cmd.FormatText, false},
		{"lowercase json", "json", cmd.FormatJSON, false},
		{"uppercase JSON", "JSON", cmd.FormatJSON, false},
		{"mixed case Json", "Json", cmd.FormatJSON, false},
		{"with spaces json", "  json  ", cmd.FormatJSON, false},
		{"lowercase YAML", "yaml", cmd.FormatYAML, false},
		{"uppercase YAML", "YAML", cmd.FormatYAML, false},
		{"mixed case YAML", "Yaml", cmd.FormatYAML, false},
		{"with spaces YAML", "  yaml  ", cmd.FormatYAML, false},
		{"invalid format", "xml", "", true},
		{"empty string", "", "", true},
		{"only spaces", "   ", "", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cobraCmd, err := NewSearchCmd(&cmd.BaseCmd{})
			require.NoError(t, err)

			cobraCmd.SetArgs([]string{"--format", tc.input})
			err = cobraCmd.ParseFlags([]string{"--format", tc.input})
			// err = cobraCmd.Flags().Parse([]string{"--format", tc.input})

			if tc.expectError {
				require.Error(t, err)
				require.ErrorContains(t, err, "invalid argument")
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectedFormat.String(), cobraCmd.Flag("format").Value.String())
			}
		})
	}
}

func TestSearchCmd_CacheTTL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		ttl           string
		expectedError string
	}{
		{
			name: "valid cache TTL",
			ttl:  "1h",
		},
		{
			name:          "invalid cache TTL",
			ttl:           "invalid",
			expectedError: "invalid cache TTL: time: invalid duration \"invalid\"",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			pkg := testServer(t)
			pkg.ID = "testserver"
			pkg.Name = "testserver"
			pkg.Tools = []packages.Tool{{Name: "tool1"}}
			pkg.Installations = map[runtime.Runtime]packages.Installation{
				runtime.UVX: {
					Runtime:     "uvx",
					Package:     "mcp-server-testserver",
					Version:     "latest",
					Recommended: true,
				},
			}

			cmdObj, err := NewSearchCmd(
				&cmd.BaseCmd{},
				cmdopts.WithRegistryBuilder(&fakeBuilder{reg: &fakeRegistry{pkg: pkg}}),
			)
			require.NoError(t, err)

			cmdObj.SetOut(io.Discard)
			cmdObj.SetErr(io.Discard)
			cmdObj.SetArgs([]string{"testserver", "--cache-ttl", tc.ttl})

			err = cmdObj.Execute()
			if tc.expectedError != "" {
				require.Error(t, err)
				require.EqualError(t, err, tc.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestSearchCmd_CacheFlagsWithTempDir(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		setupCmd func(t *testing.T, tempDir string) []string
	}{
		{
			name: "custom cache directory",
			setupCmd: func(t *testing.T, tempDir string) []string {
				return []string{"testserver", "--cache-dir", tempDir}
			},
		},
		{
			name: "both custom cache flags",
			setupCmd: func(t *testing.T, tempDir string) []string {
				return []string{"testserver", "--cache-dir", tempDir, "--cache-ttl", "30m"}
			},
		},
		{
			name: "cache disabled with custom settings",
			setupCmd: func(t *testing.T, tempDir string) []string {
				return []string{"testserver", "--no-cache", "--cache-dir", tempDir, "--cache-ttl", "2h"}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tempDir := t.TempDir()
			args := tc.setupCmd(t, tempDir)

			pkg := testServer(t)
			pkg.ID = "testserver"
			pkg.Name = "testserver"
			pkg.Tools = []packages.Tool{{Name: "tool1"}}
			pkg.Installations = map[runtime.Runtime]packages.Installation{
				runtime.UVX: {
					Runtime:     "uvx",
					Package:     "mcp-server-testserver",
					Version:     "latest",
					Recommended: true,
				},
			}

			cmdObj, err := NewSearchCmd(
				&cmd.BaseCmd{},
				cmdopts.WithRegistryBuilder(&fakeBuilder{reg: &fakeRegistry{pkg: pkg}}),
			)
			require.NoError(t, err)

			cmdObj.SetOut(io.Discard)
			cmdObj.SetErr(io.Discard)
			cmdObj.SetArgs(args)

			err = cmdObj.Execute()
			require.NoError(t, err)

			// tempDir is available here for any cache directory verification
		})
	}
}

func TestSearchCmd_NoCacheDirectoryCreatedWhenDisabled(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	cacheSubDir := filepath.Join(tempDir, "should-not-be-created")

	// Verify the cache directory doesn't exist initially.
	_, err := os.Stat(cacheSubDir)
	require.True(t, os.IsNotExist(err), "Cache directory should not exist initially")

	pkg := testServer(t)
	pkg.ID = "testserver"
	pkg.Name = "testserver"
	pkg.Tools = []packages.Tool{{Name: "tool1"}}
	pkg.Installations = map[runtime.Runtime]packages.Installation{
		runtime.UVX: {
			Runtime:     "uvx",
			Package:     "mcp-server-testserver",
			Version:     "latest",
			Recommended: true,
		},
	}

	cmdObj, err := NewSearchCmd(
		&cmd.BaseCmd{},
		cmdopts.WithRegistryBuilder(&fakeBuilder{reg: &fakeRegistry{pkg: pkg}}),
	)
	require.NoError(t, err)

	cmdObj.SetOut(io.Discard)
	cmdObj.SetErr(io.Discard)
	// Use --no-cache with custom cache directory - directory should NOT be created.
	cmdObj.SetArgs([]string{"testserver", "--no-cache", "--cache-dir", cacheSubDir})

	err = cmdObj.Execute()
	require.NoError(t, err)

	// Verify the cache directory was never created.
	_, err = os.Stat(cacheSubDir)
	require.True(t, os.IsNotExist(err), "Cache directory should not be created when --no-cache is used")
}
