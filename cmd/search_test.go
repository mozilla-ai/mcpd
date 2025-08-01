package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mozilla-ai/mcpd/v2/internal/cmd"
	cmdopts "github.com/mozilla-ai/mcpd/v2/internal/cmd/options"
	"github.com/mozilla-ai/mcpd/v2/internal/cmd/output"
	"github.com/mozilla-ai/mcpd/v2/internal/packages"
	"github.com/mozilla-ai/mcpd/v2/internal/registry/options"
	"github.com/mozilla-ai/mcpd/v2/internal/runtime"
)

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
	pkg := packages.Package{
		ID:          "test-server",
		Name:        "Test Server",
		Description: "A test server",
		License:     "MIT",
		Source:      "mcpm",
		Tools: []packages.Tool{
			{Name: "test_tool"},
		},
		Runtimes: []runtime.Runtime{runtime.UVX},
	}

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
	assert.Contains(t, outStr, "🔎 Registry search results...")
	assert.Contains(t, outStr, "🆔 test-server")
	assert.Contains(t, outStr, "📦 Found 1 package")
}

func TestSearchCmd_TextFormat(t *testing.T) {
	pkg := packages.Package{
		ID:          "test-server",
		Name:        "Test Server",
		Description: "A test server",
		License:     "MIT",
		Source:      "mcpm",
		Tools: []packages.Tool{
			{Name: "test_tool"},
		},
		Runtimes: []runtime.Runtime{runtime.UVX},
	}

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
	assert.Contains(t, outStr, "🔎 Registry search results...")
	assert.Contains(t, outStr, "🆔 test-server")
	assert.Contains(t, outStr, "📦 Found 1 package")
}

func TestSearchCmd_JSONFormat(t *testing.T) {
	pkg := packages.Package{
		ID:          "test-server",
		Name:        "Test Server",
		Description: "A test server",
		License:     "MIT",
		Source:      "mcpm",
		Tools: []packages.Tool{
			{Name: "test_tool"},
		},
		Runtimes: []runtime.Runtime{runtime.UVX},
	}

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

	var result output.ResultsPayload[packages.Package]
	err = json.Unmarshal(o.Bytes(), &result)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Results)
	assert.Len(t, result.Results, 1)
	assert.Equal(t, "test-server", result.Results[0].ID)
	assert.Equal(t, "Test Server", result.Results[0].Name)
	assert.Equal(t, "A test server", result.Results[0].Description)
	assert.Equal(t, "MIT", result.Results[0].License)
	assert.Equal(t, "mcpm", result.Results[0].Source)
	assert.Len(t, result.Results[0].Tools, 1)
	assert.Equal(t, "test_tool", result.Results[0].Tools[0].Name)
}

func TestSearchCmd_JSONFormat_EmptyResults(t *testing.T) {
	o := new(bytes.Buffer)
	cmdObj, err := NewSearchCmd(
		&cmd.BaseCmd{},
		cmdopts.WithRegistryBuilder(&fakeBuilder{reg: &fakeRegistryMultiple{packages: []packages.Package{}}}),
	)
	require.NoError(t, err)

	cmdObj.SetOut(o)
	cmdObj.SetArgs([]string{"nonexistent", "--format=json"})

	err = cmdObj.Execute()
	require.NoError(t, err)

	result := struct {
		Results []packages.Package `json:"results"`
	}{}
	err = json.Unmarshal(o.Bytes(), &result)
	require.NoError(t, err)

	assert.Empty(t, result.Results)
}

func TestSearchCmd_JSONFormat_MultipleResults(t *testing.T) {
	pkg1 := packages.Package{
		ID:          "server1",
		Name:        "Server 1",
		Description: "First server",
		License:     "MIT",
		Source:      "mcpm",
		Tools: []packages.Tool{
			{Name: "tool1"},
		},
		Runtimes: []runtime.Runtime{runtime.UVX},
	}

	pkg2 := packages.Package{
		ID:          "server2",
		Name:        "Server 2",
		Description: "Second server",
		License:     "Apache-2.0",
		Source:      "mcpm",
		Tools: []packages.Tool{
			{Name: "tool2"},
		},
		Runtimes: []runtime.Runtime{runtime.Docker},
	}

	fakeReg := &fakeRegistryMultiple{packages: []packages.Package{pkg1, pkg2}}

	output := new(bytes.Buffer)
	cmdObj, err := NewSearchCmd(
		&cmd.BaseCmd{},
		cmdopts.WithRegistryBuilder(&fakeBuilder{reg: fakeReg}),
	)
	require.NoError(t, err)

	cmdObj.SetOut(output)
	cmdObj.SetArgs([]string{"server", "--format=json"})

	err = cmdObj.Execute()
	require.NoError(t, err)

	result := struct {
		Results []packages.Package `json:"results"`
	}{}
	err = json.Unmarshal(output.Bytes(), &result)
	require.NoError(t, err)

	assert.Len(t, result.Results, 2)
	assert.Equal(t, "server1", result.Results[0].ID)
	assert.Equal(t, "server2", result.Results[1].ID)
}

func TestSearchCmd_InvalidFormat(t *testing.T) {
	output := new(bytes.Buffer)
	cmdObj, err := NewSearchCmd(
		&cmd.BaseCmd{},
		cmdopts.WithRegistryBuilder(&fakeBuilder{reg: &fakeRegistry{}}),
	)
	require.NoError(t, err)

	cmdObj.SetOut(output)
	cmdObj.SetArgs([]string{"test-server", "--format=invalid"})

	err = cmdObj.Execute()
	require.Error(t, err)
	assert.ErrorContains(t, err, "invalid argument \"invalid\"")
	assert.ErrorContains(t, err, "must be one of json, text, yaml")
}

func TestSearchCmd_CaseInsensitiveFormat(t *testing.T) {
	pkg := packages.Package{
		ID:          "test-server",
		Name:        "Test Server",
		Description: "A test server",
		License:     "MIT",
		Source:      "mcpm",
		Tools: []packages.Tool{
			{Name: "test_tool"},
		},
		Runtimes: []runtime.Runtime{runtime.UVX},
	}

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
				assert.ErrorContains(t, err, "invalid argument")
				assert.ErrorContains(t, err, "invalid format ")
				return
			}

			require.NoError(t, err)

			if tc.expectJSON {
				result := struct {
					Results []packages.Package `json:"results"`
				}{}
				err = json.Unmarshal(o.Bytes(), &result)
				require.NoError(t, err)
				assert.Len(t, result.Results, 1)
				assert.Equal(t, "test-server", result.Results[0].ID)
			} else {
				outStr := o.String()
				assert.Contains(t, outStr, "🔎 Registry search results...")
				assert.Contains(t, outStr, "🆔 test-server")
				assert.Contains(t, outStr, "📦 Found 1 package")
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

	assert.Equal(t, "registry build failed", result.Error)
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

	assert.Equal(t, "search failed", result.Error)
}

func TestSearchCmd_TextFormat_NoResults(t *testing.T) {
	fakeReg := &fakeRegistryMultiple{packages: []packages.Package{}}

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
	assert.Contains(t, outStr, "No items found")
}

func TestSearchCmd_FlagsWithJSONFormat(t *testing.T) {
	pkg := packages.Package{
		ID:          "test-server",
		Name:        "Test Server",
		Description: "A test server",
		License:     "MIT",
		Source:      "mcpm",
		Tools: []packages.Tool{
			{Name: "test_tool"},
		},
		Runtimes: []runtime.Runtime{runtime.UVX},
	}

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
		Results []packages.Package `json:"results"`
	}{}
	err = json.Unmarshal(o.Bytes(), &result)
	require.NoError(t, err)

	assert.Len(t, result.Results, 1)
	assert.Equal(t, "test-server", result.Results[0].ID)
}

func TestSearchCmd_WildcardSearch(t *testing.T) {
	pkg := packages.Package{
		ID:          "test-server",
		Name:        "Test Server",
		Description: "A test server",
		License:     "MIT",
		Source:      "mcpm",
		Tools: []packages.Tool{
			{Name: "test_tool"},
		},
		Runtimes: []runtime.Runtime{runtime.UVX},
	}

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
		Results []packages.Package `json:"results"`
	}{}
	err = json.Unmarshal(o.Bytes(), &result)
	require.NoError(t, err)

	assert.Len(t, result.Results, 1)
	assert.Equal(t, "test-server", result.Results[0].ID)
}

// fakeRegistryMultiple supports returning multiple packages
type fakeRegistryMultiple struct {
	packages []packages.Package
	err      error
}

func (f *fakeRegistryMultiple) Resolve(_ string, _ ...options.ResolveOption) (packages.Package, error) {
	if len(f.packages) > 0 {
		return f.packages[0], f.err
	}
	return packages.Package{}, f.err
}

func (f *fakeRegistryMultiple) Search(
	_ string,
	_ map[string]string,
	_ ...options.SearchOption,
) ([]packages.Package, error) {
	return f.packages, f.err
}

func (f *fakeRegistryMultiple) ID() string {
	return "fake-multiple"
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
				assert.ErrorContains(t, err, "invalid argument")
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expectedFormat.String(), cobraCmd.Flag("format").Value.String())
			}
		})
	}
}
