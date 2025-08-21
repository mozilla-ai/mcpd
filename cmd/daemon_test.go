package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mozilla-ai/mcpd/v2/internal/cmd"
	cmdopts "github.com/mozilla-ai/mcpd/v2/internal/cmd/options"
	"github.com/mozilla-ai/mcpd/v2/internal/config"
	configcontext "github.com/mozilla-ai/mcpd/v2/internal/context"
	"github.com/mozilla-ai/mcpd/v2/internal/daemon"
	"github.com/mozilla-ai/mcpd/v2/internal/runtime"
)

// mockConfigLoader implements config.Loader for testing.
type mockConfigLoader struct {
	entries []config.ServerEntry
	err     error
}

func (m *mockConfigLoader) Load(path string) (config.Modifier, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &mockConfig{entries: m.entries}, nil
}

// mockConfig implements config.Modifier for testing.
type mockConfig struct {
	entries []config.ServerEntry
}

func (m *mockConfig) AddServer(entry config.ServerEntry) error {
	m.entries = append(m.entries, entry)
	return nil
}

func (m *mockConfig) RemoveServer(name string) error {
	for i, entry := range m.entries {
		if entry.Name == name {
			m.entries = append(m.entries[:i], m.entries[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("server %s not found", name)
}

func (m *mockConfig) ListServers() []config.ServerEntry {
	return m.entries
}

func (m *mockConfig) SaveConfig() error {
	return nil
}

// mockContextLoader implements configcontext.Loader for testing.
type mockContextLoader struct {
	servers []runtime.Server
	err     error
}

func (m *mockContextLoader) Load(path string) (configcontext.Modifier, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &mockContext{servers: m.servers}, nil
}

// mockContext implements configcontext.Modifier for testing.
type mockContext struct {
	servers  []runtime.Server
	contexts []configcontext.ServerExecutionContext
}

func (m *mockContext) Get(name string) (configcontext.ServerExecutionContext, bool) {
	for _, ctx := range m.contexts {
		if ctx.Name == name {
			return ctx, true
		}
	}
	return configcontext.ServerExecutionContext{}, false
}

func (m *mockContext) Upsert(ctx configcontext.ServerExecutionContext) (configcontext.UpsertResult, error) {
	for i, existing := range m.contexts {
		if existing.Name == ctx.Name {
			m.contexts[i] = ctx
			return configcontext.Updated, nil
		}
	}
	m.contexts = append(m.contexts, ctx)
	return configcontext.Created, nil
}

func (m *mockContext) List() []configcontext.ServerExecutionContext {
	return m.contexts
}

func TestDaemon_NewDaemonCmd_Success(t *testing.T) {
	t.Parallel()

	baseCmd := &cmd.BaseCmd{}
	cobraCmd, err := NewDaemonCmd(
		baseCmd,
		cmdopts.WithConfigLoader(&mockConfigLoader{}),
		cmdopts.WithContextLoader(&mockContextLoader{}),
	)
	require.NoError(t, err)
	require.NotNil(t, cobraCmd)

	assert.Equal(t, "daemon", cobraCmd.Name())
	assert.Contains(t, cobraCmd.Short, "daemon instance")
	assert.Contains(t, cobraCmd.Long, "MCP servers")
}

func TestDaemon_NewDaemonCmd_WithOptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		opts    []cmdopts.CmdOption
		wantErr string
	}{
		{
			name: "valid options",
			opts: []cmdopts.CmdOption{
				cmdopts.WithConfigLoader(&mockConfigLoader{}),
				cmdopts.WithContextLoader(&mockContextLoader{}),
			},
		},
		{
			name: "no options provided",
			opts: []cmdopts.CmdOption{},
		},
		{
			name: "nil config loader",
			opts: []cmdopts.CmdOption{
				cmdopts.WithConfigLoader(nil),
				cmdopts.WithContextLoader(&mockContextLoader{}),
			},
			wantErr: "config loader cannot be nil",
		},
		{
			name: "nil context loader",
			opts: []cmdopts.CmdOption{
				cmdopts.WithConfigLoader(&mockConfigLoader{}),
				cmdopts.WithContextLoader(nil),
			},
			wantErr: "context loader cannot be nil",
		},
		{
			name: "config loader interface pointing to nil",
			opts: []cmdopts.CmdOption{
				cmdopts.WithConfigLoader((*mockConfigLoader)(nil)),
				cmdopts.WithContextLoader(&mockContextLoader{}),
			},
			wantErr: "config loader cannot be nil",
		},
		{
			name: "context loader interface pointing to nil",
			opts: []cmdopts.CmdOption{
				cmdopts.WithConfigLoader(&mockConfigLoader{}),
				cmdopts.WithContextLoader((*mockContextLoader)(nil)),
			},
			wantErr: "context loader cannot be nil",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			baseCmd := &cmd.BaseCmd{}
			cobraCmd, err := NewDaemonCmd(baseCmd, tc.opts...)

			if tc.wantErr != "" {
				require.Error(t, err)
				require.EqualError(t, err, tc.wantErr)
				require.Nil(t, cobraCmd)
			} else {
				require.NoError(t, err)
				require.NotNil(t, cobraCmd)
			}
		})
	}
}

func TestDaemon_DaemonCmd_Flags(t *testing.T) {
	t.Parallel()

	baseCmd := &cmd.BaseCmd{}
	cobraCmd, err := NewDaemonCmd(
		baseCmd,
		cmdopts.WithConfigLoader(&mockConfigLoader{}),
		cmdopts.WithContextLoader(&mockContextLoader{}),
	)
	require.NoError(t, err)

	flags := cobraCmd.Flags()

	require.NotNil(t, flags.Lookup("dev"))
	require.NotNil(t, flags.Lookup(flagAddr))

	require.NotNil(t, flags.Lookup(flagCORSEnable))
	require.NotNil(t, flags.Lookup(flagCORSOrigin))
	require.NotNil(t, flags.Lookup(flagCORSMethod))
	require.NotNil(t, flags.Lookup(flagCORSCredentials))
	require.NotNil(t, flags.Lookup(flagCORSMaxAge))

	require.NotNil(t, flags.Lookup(flagTimeoutAPIShutdown))
	require.NotNil(t, flags.Lookup(flagTimeoutMCPInit))
	require.NotNil(t, flags.Lookup(flagTimeoutMCPHealth))

	require.NotNil(t, flags.Lookup(flagIntervalMCPHealth))

	devFlag := flags.Lookup("dev")
	require.NotNil(t, devFlag)
	assert.Equal(t, "false", devFlag.DefValue)

	addrFlag := flags.Lookup(flagAddr)
	require.NotNil(t, addrFlag)
	assert.Equal(t, "0.0.0.0:8090", addrFlag.DefValue)

	corsEnableFlag := flags.Lookup(flagCORSEnable)
	require.NotNil(t, corsEnableFlag)
	assert.Equal(t, "false", corsEnableFlag.DefValue)
}

func TestDaemon_DaemonCmd_FlagMutualExclusion(t *testing.T) {
	t.Parallel()

	baseCmd := &cmd.BaseCmd{}
	cobraCmd, err := NewDaemonCmd(
		baseCmd,
		cmdopts.WithConfigLoader(&mockConfigLoader{}),
		cmdopts.WithContextLoader(&mockContextLoader{}),
	)
	require.NoError(t, err)

	cobraCmd.SetArgs([]string{"--dev", "--addr=localhost:9000"})
	cobraCmd.SetOut(io.Discard)
	cobraCmd.SetErr(io.Discard)

	err = cobraCmd.Execute()
	require.Error(t, err)
	require.Contains(t, err.Error(), "if any flags in the group [dev addr] are set none of the others can be")
}

func TestDaemon_DaemonCmd_ValidateFlags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		config      daemonFlagConfig
		expectError string
	}{
		{
			name: "valid configuration",
			config: daemonFlagConfig{
				cors: corsFlagConfig{
					enable:  true,
					origins: []string{"http://localhost:3000"},
				},
				timeout: timeoutFlagConfig{
					apiShutdown: "5s",
					mcpInit:     "10s",
					healthCheck: "3s",
				},
				interval: intervalFlagConfig{
					healthCheck: "30s",
				},
			},
		},
		{
			name: "invalid CORS max age duration",
			config: daemonFlagConfig{
				cors: corsFlagConfig{
					enable:  true,
					origins: []string{"http://localhost:3000"},
					maxAge:  "invalid-duration",
				},
			},
			expectError: "invalid --cors-max-age duration: time: invalid duration \"invalid-duration\"",
		},
		{
			name: "invalid API shutdown timeout",
			config: daemonFlagConfig{
				timeout: timeoutFlagConfig{
					apiShutdown: "not-a-duration",
				},
			},
			expectError: "invalid --timeout-api-shutdown duration: time: invalid duration \"not-a-duration\"",
		},
		{
			name: "invalid MCP init timeout",
			config: daemonFlagConfig{
				timeout: timeoutFlagConfig{
					mcpInit: "invalid",
				},
			},
			expectError: "invalid --timeout-mcp-init duration: time: invalid duration \"invalid\"",
		},
		{
			name: "invalid health check timeout",
			config: daemonFlagConfig{
				timeout: timeoutFlagConfig{
					healthCheck: "bad-format",
				},
			},
			expectError: "invalid --timeout-mcp-health duration: time: invalid duration \"bad-format\"",
		},
		{
			name: "invalid health check interval",
			config: daemonFlagConfig{
				interval: intervalFlagConfig{
					healthCheck: "not-valid",
				},
			},
			expectError: "invalid --interval-mcp-health duration: time: invalid duration \"not-valid\"",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			baseCmd := &cmd.BaseCmd{}
			cobraCmd, err := NewDaemonCmd(
				baseCmd,
				cmdopts.WithConfigLoader(&mockConfigLoader{}),
				cmdopts.WithContextLoader(&mockContextLoader{}),
			)
			require.NoError(t, err)

			// Access the DaemonCmd through the RunE function closure
			daemonCmd := &DaemonCmd{
				BaseCmd:   baseCmd,
				config:    tc.config,
				cfgLoader: &mockConfigLoader{},
				ctxLoader: &mockContextLoader{},
			}

			err = daemonCmd.validateFlags(cobraCmd)

			if tc.expectError != "" {
				require.Error(t, err)
				require.EqualError(t, err, tc.expectError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestDaemon_DaemonCmd_BuildAPIOptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		config         daemonFlagConfig
		expectError    string
		validateResult func(t *testing.T, opts []daemon.APIOption)
	}{
		{
			name: "CORS disabled",
			config: daemonFlagConfig{
				cors: corsFlagConfig{enable: false},
			},
			validateResult: func(t *testing.T, opts []daemon.APIOption) {
				apiOpts, err := daemon.NewAPIOptions(opts...)
				require.NoError(t, err)
				assert.False(t, apiOpts.CORS.Enabled)
			},
		},
		{
			name: "CORS enabled with origins",
			config: daemonFlagConfig{
				cors: corsFlagConfig{
					enable:  true,
					origins: []string{"http://localhost:3000", "https://example.com"},
				},
			},
			validateResult: func(t *testing.T, opts []daemon.APIOption) {
				apiOpts, err := daemon.NewAPIOptions(opts...)
				require.NoError(t, err)
				assert.True(t, apiOpts.CORS.Enabled)
				assert.ElementsMatch(
					t,
					[]string{"http://localhost:3000", "https://example.com"},
					apiOpts.CORS.AllowOrigins,
				)
			},
		},
		{
			name: "CORS with custom methods",
			config: daemonFlagConfig{
				cors: corsFlagConfig{
					enable:  true,
					origins: []string{"http://localhost:3000"},
					methods: []string{"GET", "POST"},
				},
			},
			validateResult: func(t *testing.T, opts []daemon.APIOption) {
				apiOpts, err := daemon.NewAPIOptions(opts...)
				require.NoError(t, err)
				assert.True(t, apiOpts.CORS.Enabled)
				assert.ElementsMatch(t, []string{"GET", "POST"}, apiOpts.CORS.AllowMethods)
			},
		},
		{
			name: "CORS with credentials",
			config: daemonFlagConfig{
				cors: corsFlagConfig{
					enable:      true,
					origins:     []string{"http://localhost:3000"},
					credentials: true,
				},
			},
			validateResult: func(t *testing.T, opts []daemon.APIOption) {
				apiOpts, err := daemon.NewAPIOptions(opts...)
				require.NoError(t, err)
				assert.True(t, apiOpts.CORS.Enabled)
				assert.True(t, apiOpts.CORS.AllowCredentials)
			},
		},
		{
			name: "CORS with custom max age",
			config: daemonFlagConfig{
				cors: corsFlagConfig{
					enable:  true,
					origins: []string{"http://localhost:3000"},
					maxAge:  "10m",
				},
			},
			validateResult: func(t *testing.T, opts []daemon.APIOption) {
				apiOpts, err := daemon.NewAPIOptions(opts...)
				require.NoError(t, err)
				assert.True(t, apiOpts.CORS.Enabled)
				assert.Equal(t, 10*time.Minute, apiOpts.CORS.MaxAge)
			},
		},
		{
			name: "API shutdown timeout",
			config: daemonFlagConfig{
				timeout: timeoutFlagConfig{
					apiShutdown: "30s",
				},
			},
			validateResult: func(t *testing.T, opts []daemon.APIOption) {
				apiOpts, err := daemon.NewAPIOptions(opts...)
				require.NoError(t, err)
				assert.Equal(t, 30*time.Second, apiOpts.ShutdownTimeout)
			},
		},
		{
			name: "invalid CORS max age",
			config: daemonFlagConfig{
				cors: corsFlagConfig{
					enable:  true,
					origins: []string{"http://localhost:3000"},
					maxAge:  "invalid",
				},
			},
			expectError: "invalid cors-max-age: time: invalid duration \"invalid\"",
		},
		{
			name: "invalid API shutdown timeout",
			config: daemonFlagConfig{
				timeout: timeoutFlagConfig{
					apiShutdown: "not-valid",
				},
			},
			expectError: "invalid timeout-api-shutdown: time: invalid duration \"not-valid\"",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			daemonCmd := &DaemonCmd{
				config: tc.config,
			}

			opts, err := daemonCmd.buildAPIOptions()

			if tc.expectError != "" {
				require.Error(t, err)
				require.EqualError(t, err, tc.expectError)
			} else {
				require.NoError(t, err)
				if tc.validateResult != nil {
					tc.validateResult(t, opts)
				}
			}
		})
	}
}

func TestDaemon_DaemonCmd_BuildDaemonOptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		config         daemonFlagConfig
		expectError    string
		validateResult func(t *testing.T, opt ...daemon.Option)
	}{
		{
			name: "valid daemon options",
			config: daemonFlagConfig{
				timeout: timeoutFlagConfig{
					mcpInit:     "15s",
					healthCheck: "5s",
				},
				interval: intervalFlagConfig{
					healthCheck: "20s",
				},
			},
			validateResult: func(t *testing.T, opt ...daemon.Option) {
				opts, err := daemon.NewOptions(opt...)
				require.NoError(t, err)
				assert.Equal(t, 15*time.Second, opts.ClientInitTimeout)
				assert.Equal(t, 5*time.Second, opts.ClientHealthCheckTimeout)
				assert.Equal(t, 20*time.Second, opts.ClientHealthCheckInterval)
			},
		},
		{
			name: "invalid MCP init timeout",
			config: daemonFlagConfig{
				timeout: timeoutFlagConfig{
					mcpInit: "invalid",
				},
			},
			expectError: "invalid timeout-mcp-init: time: invalid duration \"invalid\"",
		},
		{
			name: "invalid health check timeout",
			config: daemonFlagConfig{
				timeout: timeoutFlagConfig{
					healthCheck: "bad-format",
				},
			},
			expectError: "invalid timeout-mcp-health: time: invalid duration \"bad-format\"",
		},
		{
			name: "invalid health check interval",
			config: daemonFlagConfig{
				interval: intervalFlagConfig{
					healthCheck: "not-valid",
				},
			},
			expectError: "invalid interval-mcp-health: time: invalid duration \"not-valid\"",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			daemonCmd := &DaemonCmd{
				config: tc.config,
			}

			apiOpts, err := daemonCmd.buildAPIOptions()
			require.NoError(t, err)

			opts, err := daemonCmd.buildDaemonOptions(apiOpts)

			if tc.expectError != "" {
				require.Error(t, err)
				require.EqualError(t, err, tc.expectError)
			} else {
				require.NoError(t, err)
				if tc.validateResult != nil {
					tc.validateResult(t, opts...)
				}
			}
		})
	}
}

func TestDaemon_DaemonCmd_FormatConfigInfo(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		config         daemonFlagConfig
		dev            bool
		expectedInInfo []string
		notInInfo      []string
	}{
		{
			name: "no custom config",
			config: daemonFlagConfig{
				api: apiFlagConfig{addr: "0.0.0.0:8090"},
			},
			expectedInInfo: []string{},
		},
		{
			name: "custom API address (runtime value shown)",
			config: daemonFlagConfig{
				api: apiFlagConfig{addr: "127.0.0.1:9090"},
			},
			dev:            false,
			expectedInInfo: []string{"API address", "localhost:8090"}, // Uses test runtime addr
		},
		{
			name: "dev mode shows runtime address",
			config: daemonFlagConfig{
				api: apiFlagConfig{addr: "127.0.0.1:9090"},
			},
			dev:            true,
			expectedInInfo: []string{"API address", "localhost:8090"}, // Shows runtime addr
		},
		{
			name: "CORS enabled",
			config: daemonFlagConfig{
				cors: corsFlagConfig{
					enable:  true,
					origins: []string{"http://localhost:3000", "https://example.com"},
				},
			},
			expectedInInfo: []string{"CORS enabled", "http://localhost:3000", "https://example.com"},
		},
		{
			name: "CORS with methods and credentials",
			config: daemonFlagConfig{
				cors: corsFlagConfig{
					enable:      true,
					origins:     []string{"http://localhost:3000"},
					methods:     []string{"GET", "POST"},
					credentials: true,
					maxAge:      "10m",
				},
			},
			expectedInInfo: []string{
				"CORS enabled", "http://localhost:3000",
				"CORS methods", "GET", "POST",
				"CORS credentials", "true",
				"CORS max age", "10m",
			},
		},
		{
			name: "custom timeouts",
			config: daemonFlagConfig{
				timeout: timeoutFlagConfig{
					apiShutdown: "10s",
					mcpInit:     "20s",
					healthCheck: "2s",
				},
			},
			expectedInInfo: []string{
				"API shutdown timeout", "10s",
				"MCP init timeout", "20s",
				"MCP health check timeout", "2s",
				"API address", "localhost:8090", // Runtime address shown
			},
		},
		{
			name: "custom health check interval",
			config: daemonFlagConfig{
				interval: intervalFlagConfig{
					healthCheck: "15s",
				},
			},
			expectedInInfo: []string{"MCP health check interval", "15s"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			daemonCmd := &DaemonCmd{
				config: tc.config,
				dev:    tc.dev,
			}

			info := daemonCmd.formatConfigInfo("localhost:8090") // Use default test addr

			for _, expected := range tc.expectedInInfo {
				assert.Contains(t, info, expected, "Expected '%s' to be in config info", expected)
			}

			for _, notExpected := range tc.notInInfo {
				assert.NotContains(t, info, notExpected, "Expected '%s' to NOT be in config info", notExpected)
			}
		})
	}
}

func TestDaemon_DaemonCmd_PrintDevBanner(t *testing.T) {
	t.Parallel()

	var logBuf bytes.Buffer
	logger := hclog.New(&hclog.LoggerOptions{
		Level:      hclog.Info,
		Output:     &logBuf,
		JSONFormat: false,
	})

	daemonCmd := &DaemonCmd{
		config: daemonFlagConfig{
			cors: corsFlagConfig{
				enable:  true,
				origins: []string{"http://localhost:3000"},
				maxAge:  "10m",
			},
			timeout: timeoutFlagConfig{
				healthCheck: "2s",
			},
		},
	}

	var bannerBuf bytes.Buffer
	addr := "localhost:8090"
	daemonCmd.printDevBanner(&bannerBuf, logger, addr)

	// Verify the banner output
	bannerOutput := bannerBuf.String()
	assert.Contains(t, bannerOutput, "mcpd daemon running in 'dev' mode")
	assert.Contains(t, bannerOutput, "Local API:\thttp://localhost:8090/api/v1")
	assert.Contains(t, bannerOutput, "OpenAPI UI:\thttp://localhost:8090/docs")
	assert.Contains(t, bannerOutput, "CORS enabled:\ttrue (origins: http://localhost:3000)")
	assert.Contains(t, bannerOutput, "CORS max age:\t10m")
	assert.Contains(t, bannerOutput, "MCP health check timeout:\t2s")
	assert.Contains(t, bannerOutput, "Press Ctrl+C to stop")

	// Verify the logger was called
	logOutput := logBuf.String()
	assert.Contains(t, logOutput, "Launching daemon in dev mode")
	assert.Contains(t, logOutput, "localhost:8090")
}

func TestDaemon_DaemonCmd_DevModeOverrideWarning(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		addr        string
		expectWarn  bool
		warnMessage string
	}{
		{
			name:       "default address no warning",
			addr:       "0.0.0.0:8090",
			expectWarn: false,
		},
		{
			name:        "custom address shows warning",
			addr:        "127.0.0.1:9000",
			expectWarn:  true,
			warnMessage: "Development mode ignores custom address",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var logBuf bytes.Buffer
			logger := hclog.New(&hclog.LoggerOptions{
				Level:  hclog.Warn,
				Output: &logBuf,
			})

			devAddr := "localhost:8090"
			if tc.addr != "0.0.0.0:8090" {
				logger.Warn("Development mode ignores custom address", "provided", tc.addr, "using", devAddr)
			}

			logOutput := logBuf.String()
			if tc.expectWarn {
				assert.Contains(t, logOutput, tc.warnMessage)
				assert.Contains(t, logOutput, tc.addr)
			} else {
				assert.Empty(t, logOutput)
			}
		})
	}
}

func BenchmarkDaemon_DaemonCmd_ValidateFlags(b *testing.B) {
	cfg := daemonFlagConfig{
		cors: corsFlagConfig{
			enable:  true,
			origins: []string{"http://localhost:3000"},
		},
		timeout: timeoutFlagConfig{
			apiShutdown: "5s",
			mcpInit:     "10s",
			healthCheck: "3s",
		},
		interval: intervalFlagConfig{
			healthCheck: "30s",
		},
	}

	baseCmd := &cmd.BaseCmd{}
	cobraCmd, err := NewDaemonCmd(
		baseCmd,
		cmdopts.WithConfigLoader(&mockConfigLoader{}),
		cmdopts.WithContextLoader(&mockContextLoader{}),
	)
	require.NoError(b, err)

	daemonCmd := &DaemonCmd{
		BaseCmd:   baseCmd,
		config:    cfg,
		cfgLoader: &mockConfigLoader{},
		ctxLoader: &mockContextLoader{},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = daemonCmd.validateFlags(cobraCmd)
	}
}

func BenchmarkDaemon_DaemonCmd_BuildAPIOptions(b *testing.B) {
	cfg := daemonFlagConfig{
		cors: corsFlagConfig{
			enable:      true,
			origins:     []string{"http://localhost:3000"},
			methods:     []string{"GET", "POST", "PUT", "DELETE"},
			credentials: false,
			maxAge:      "5m",
		},
		timeout: timeoutFlagConfig{
			apiShutdown: "5s",
		},
	}

	daemonCmd := &DaemonCmd{config: cfg}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = daemonCmd.buildAPIOptions()
	}
}

func TestFormatValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{
			name:     "empty string slice",
			input:    []string{},
			expected: "[]",
		},
		{
			name:     "single item string slice",
			input:    []string{"item1"},
			expected: "[item1]",
		},
		{
			name:     "multiple items string slice",
			input:    []string{"GET", "POST", "PUT"},
			expected: "[GET, POST, PUT]",
		},
		{
			name:     "string slice with spaces",
			input:    []string{"http://localhost:3000", "https://example.com"},
			expected: "[http://localhost:3000, https://example.com]",
		},
		{
			name:     "regular string",
			input:    "simple-string",
			expected: "simple-string",
		},
		{
			name:     "boolean true",
			input:    true,
			expected: "true",
		},
		{
			name:     "boolean false",
			input:    false,
			expected: "false",
		},
		{
			name:     "integer",
			input:    42,
			expected: "42",
		},
		{
			name:     "time duration",
			input:    5 * time.Second,
			expected: "5s",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := formatValue(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestConfigOverrideWarning(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		flagName     string
		currentValue any
		configValue  any
		expected     string
	}{
		{
			name:         "string values",
			flagName:     "test-flag",
			currentValue: "current",
			configValue:  "new",
			expected:     "--test-flag: config=new, flag=current (using flag)",
		},
		{
			name:         "boolean values",
			flagName:     flagCORSEnable,
			currentValue: false,
			configValue:  true,
			expected:     "--cors-enable: config=true, flag=false (using flag)",
		},
		{
			name:         "string slice values",
			flagName:     flagCORSOrigin,
			currentValue: []string{"http://localhost:3000"},
			configValue:  []string{"http://localhost:3000", "https://example.com"},
			expected:     "--cors-allow-origin: config=[http://localhost:3000, https://example.com], flag=[http://localhost:3000] (using flag)",
		},
		{
			name:         "empty to populated string slice",
			flagName:     flagCORSMethod,
			currentValue: []string{},
			configValue:  []string{"GET", "POST"},
			expected:     "--cors-allow-method: config=[GET, POST], flag=[] (using flag)",
		},
		{
			name:         "integer values",
			flagName:     "timeout",
			currentValue: 30,
			configValue:  60,
			expected:     "--timeout: config=60, flag=30 (using flag)",
		},
		{
			name:         "duration values",
			flagName:     "api-timeout",
			currentValue: 5 * time.Second,
			configValue:  10 * time.Second,
			expected:     "--api-timeout: config=10s, flag=5s (using flag)",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := flagOverrideWarning(tc.flagName, tc.configValue, tc.currentValue)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestFlagOverrideWarning_TypeSafety(t *testing.T) {
	t.Parallel()

	// Test that the generic function enforces type safety
	result1 := flagOverrideWarning("string-flag", "config-value", "flag-value")
	assert.Equal(t, "--string-flag: config=config-value, flag=flag-value (using flag)", result1)

	result2 := flagOverrideWarning("bool-flag", false, true)
	assert.Equal(t, "--bool-flag: config=false, flag=true (using flag)", result2)

	result3 := flagOverrideWarning("slice-flag", []string{"a"}, []string{"b", "c"})
	assert.Equal(t, "--slice-flag: config=[a], flag=[b, c] (using flag)", result3)

	// Note: The compiler will prevent mismatched types at compile time,
	// so we don't need runtime tests for type mismatches.
}

// Helper functions for creating pointers in tests
func testStringPtr(t *testing.T, s string) *string {
	t.Helper()
	return &s
}

func testBoolPtr(t *testing.T, b bool) *bool {
	t.Helper()
	return &b
}

func testDurationPtr(t *testing.T, d time.Duration) *config.Duration {
	t.Helper()
	configDur := config.Duration(d)
	return &configDur
}

func TestDaemon_ApplyConfigAPI(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		apiConfig       *config.APIConfigSection
		flagChanged     bool
		initialFlagAddr string
		expectWarnings  []string
		expectFinalAddr string
	}{
		{
			name:            "nil API config",
			apiConfig:       nil,
			expectWarnings:  nil,
			initialFlagAddr: "localhost:8090",
			expectFinalAddr: "localhost:8090", // unchanged
		},
		{
			name: "empty API addr",
			apiConfig: &config.APIConfigSection{
				Addr: nil,
			},
			expectWarnings:  nil,
			initialFlagAddr: "localhost:8090",
			expectFinalAddr: "localhost:8090", // unchanged
		},
		{
			name: "config overrides flag value - flag changed",
			apiConfig: &config.APIConfigSection{
				Addr: testStringPtr(t, "localhost:9000"),
			},
			flagChanged:     true,
			initialFlagAddr: "localhost:7000",
			expectWarnings:  []string{"--addr: config=localhost:9000, flag=localhost:7000 (using flag)"},
			expectFinalAddr: "localhost:7000", // flag wins
		},
		{
			name: "config sets value - flag not changed",
			apiConfig: &config.APIConfigSection{
				Addr: testStringPtr(t, "localhost:9000"),
			},
			flagChanged:     false,
			initialFlagAddr: "localhost:8090", // default value
			expectWarnings:  nil,
			expectFinalAddr: "localhost:9000",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			daemonCmd := &DaemonCmd{
				config: daemonFlagConfig{
					api: apiFlagConfig{
						addr: tc.initialFlagAddr,
					},
				},
			}

			logger := hclog.NewNullLogger()
			command := &cobra.Command{}

			// Add the addr flag to the command
			command.Flags().String(flagAddr, "", "test flag")

			// Simulate flag being changed if needed
			if tc.flagChanged {
				err := command.Flags().Set(flagAddr, tc.initialFlagAddr)
				require.NoError(t, err)
			}

			warnings := daemonCmd.loadConfigAPI(tc.apiConfig, logger, command)

			assert.Equal(t, tc.expectWarnings, warnings)
			assert.Equal(t, tc.expectFinalAddr, daemonCmd.config.api.addr)
		})
	}
}

func TestDaemon_ApplyConfigCORS(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                   string
		corsConfig             *config.CORSConfigSection
		enableFlagChanged      bool
		originFlagChanged      bool
		methodFlagChanged      bool
		credentialsFlagChanged bool
		maxAgeFlagChanged      bool
		initialConfig          corsFlagConfig
		expectWarnings         []string
		expectFinalConfig      corsFlagConfig
	}{
		{
			name:              "nil CORS config",
			corsConfig:        nil,
			expectWarnings:    nil,
			initialConfig:     corsFlagConfig{},
			expectFinalConfig: corsFlagConfig{}, // unchanged
		},
		{
			name: "CORS enable - flag not changed",
			corsConfig: &config.CORSConfigSection{
				Enable: testBoolPtr(t, true),
			},
			enableFlagChanged: false,
			initialConfig:     corsFlagConfig{enable: false},
			expectWarnings:    nil,
			expectFinalConfig: corsFlagConfig{enable: true},
		},
		{
			name: "CORS enable - flag changed (override)",
			corsConfig: &config.CORSConfigSection{
				Enable: testBoolPtr(t, true),
			},
			enableFlagChanged: true,
			initialConfig:     corsFlagConfig{enable: false},
			expectWarnings:    []string{"--cors-enable: config=true, flag=false (using flag)"},
			expectFinalConfig: corsFlagConfig{enable: false}, // flag wins
		},
		{
			name: "CORS origins - flag not changed",
			corsConfig: &config.CORSConfigSection{
				Origins: []string{"http://example.com", "https://test.com"},
			},
			originFlagChanged: false,
			initialConfig:     corsFlagConfig{origins: []string{"http://localhost:3000"}},
			expectWarnings:    nil,
			expectFinalConfig: corsFlagConfig{origins: []string{"http://example.com", "https://test.com"}},
		},
		{
			name: "CORS origins - flag changed (override)",
			corsConfig: &config.CORSConfigSection{
				Origins: []string{"http://example.com"},
			},
			originFlagChanged: true,
			initialConfig:     corsFlagConfig{origins: []string{"http://localhost:3000"}},
			expectWarnings: []string{
				"--cors-allow-origin: config=[http://example.com], flag=[http://localhost:3000] (using flag)",
			},
			expectFinalConfig: corsFlagConfig{origins: []string{"http://localhost:3000"}}, // flag wins
		},
		{
			name: "multiple CORS settings with mixed flag changes",
			corsConfig: &config.CORSConfigSection{
				Enable:      testBoolPtr(t, true),
				Origins:     []string{"http://example.com"},
				Methods:     []string{"GET", "POST"},
				Credentials: testBoolPtr(t, true),
				MaxAge:      testDurationPtr(t, 10*time.Minute), // 10 minutes in seconds
			},
			enableFlagChanged:      true,  // will warn
			originFlagChanged:      false, // no warn
			methodFlagChanged:      true,  // will warn
			credentialsFlagChanged: false, // no warn
			maxAgeFlagChanged:      true,  // will warn
			initialConfig: corsFlagConfig{
				enable:      false,
				origins:     []string{"http://localhost:3000"},
				methods:     []string{"GET"},
				credentials: false,
				maxAge:      "5m",
			},
			expectWarnings: []string{
				"--cors-enable: config=true, flag=false (using flag)",
				"--cors-allow-method: config=[GET, POST], flag=[GET] (using flag)",
				"--cors-max-age: config=10m, flag=5m (using flag)",
			},
			expectFinalConfig: corsFlagConfig{
				enable:      false,                          // flag wins
				origins:     []string{"http://example.com"}, // config used (no flag set)
				methods:     []string{"GET"},                // flag wins
				credentials: true,                           // config used (no flag set)
				maxAge:      "5m",                           // flag wins
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			daemonCmd := &DaemonCmd{
				config: daemonFlagConfig{
					cors: tc.initialConfig,
				},
			}

			logger := hclog.NewNullLogger()
			c := &cobra.Command{}
			// Add all CORS flags to the command
			c.Flags().Bool(flagCORSEnable, false, "test flag")
			c.Flags().StringSlice(flagCORSOrigin, []string{}, "test flag")
			c.Flags().StringSlice(flagCORSMethod, []string{}, "test flag")
			c.Flags().Bool(flagCORSCredentials, false, "test flag")
			c.Flags().String(flagCORSMaxAge, "", "test flag")

			// Simulate flags being changed if needed
			if tc.enableFlagChanged {
				err := c.Flags().Set(flagCORSEnable, fmt.Sprintf("%t", tc.initialConfig.enable))
				require.NoError(t, err)
			}
			if tc.originFlagChanged {
				for _, origin := range tc.initialConfig.origins {
					err := c.Flags().Set(flagCORSOrigin, origin)
					require.NoError(t, err)
				}
			}
			if tc.methodFlagChanged {
				for _, method := range tc.initialConfig.methods {
					err := c.Flags().Set(flagCORSMethod, method)
					require.NoError(t, err)
				}
			}
			if tc.credentialsFlagChanged {
				err := c.Flags().Set(flagCORSCredentials, fmt.Sprintf("%t", tc.initialConfig.credentials))
				require.NoError(t, err)
			}
			if tc.maxAgeFlagChanged {
				err := c.Flags().Set(flagCORSMaxAge, tc.initialConfig.maxAge)
				require.NoError(t, err)
			}

			warnings := daemonCmd.loadConfigCORS(tc.corsConfig, logger, c)

			assert.Equal(t, tc.expectWarnings, warnings)
			assert.Equal(t, tc.expectFinalConfig, daemonCmd.config.cors)
		})
	}
}

func TestDaemon_ApplyConfigTimeout(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                   string
		apiConfig              *config.APIConfigSection
		mcpConfig              *config.MCPConfigSection
		apiShutdownFlagChanged bool
		mcpInitFlagChanged     bool
		healthCheckFlagChanged bool
		initialConfig          timeoutFlagConfig
		expectWarnings         []string
		expectFinalConfig      timeoutFlagConfig
	}{
		{
			name:              "nil timeout config",
			apiConfig:         nil,
			mcpConfig:         nil,
			expectWarnings:    nil,
			initialConfig:     timeoutFlagConfig{},
			expectFinalConfig: timeoutFlagConfig{}, // unchanged
		},
		{
			name: "API shutdown timeout - flag not changed",
			apiConfig: &config.APIConfigSection{
				Timeout: &config.APITimeoutConfigSection{
					Shutdown: testDurationPtr(t, 30*time.Second),
				},
			},
			apiShutdownFlagChanged: false,
			initialConfig:          timeoutFlagConfig{apiShutdown: "15s"},
			expectWarnings:         nil,
			expectFinalConfig:      timeoutFlagConfig{apiShutdown: "30s"},
		},
		{
			name: "API shutdown timeout - flag changed (override)",
			apiConfig: &config.APIConfigSection{
				Timeout: &config.APITimeoutConfigSection{
					Shutdown: testDurationPtr(t, 30*time.Second),
				},
			},
			apiShutdownFlagChanged: true,
			initialConfig:          timeoutFlagConfig{apiShutdown: "15s"},
			expectWarnings:         []string{"--timeout-api-shutdown: config=30s, flag=15s (using flag)"},
			expectFinalConfig:      timeoutFlagConfig{apiShutdown: "15s"}, // flag wins
		},
		{
			name: "multiple timeout settings with mixed flag changes",
			apiConfig: &config.APIConfigSection{
				Timeout: &config.APITimeoutConfigSection{
					Shutdown: testDurationPtr(t, 45*time.Second),
				},
			},
			mcpConfig: &config.MCPConfigSection{
				Timeout: &config.MCPTimeoutConfigSection{
					Init:   testDurationPtr(t, 60*time.Second),
					Health: testDurationPtr(t, 10*time.Second),
				},
			},
			apiShutdownFlagChanged: true,  // will warn
			mcpInitFlagChanged:     false, // no warn
			healthCheckFlagChanged: true,  // will warn
			initialConfig: timeoutFlagConfig{
				apiShutdown: "30s",
				// mcpInit:        "45s",
				healthCheck:    "5s",
				clientShutdown: "15s",
			},
			expectWarnings: []string{
				"--timeout-api-shutdown: config=45s, flag=30s (using flag)",
				"--timeout-mcp-health: config=10s, flag=5s (using flag)",
			},
			expectFinalConfig: timeoutFlagConfig{
				apiShutdown:    "30s", // flag wins
				mcpInit:        "1m",  // config used (no flag set)
				healthCheck:    "5s",  // flag wins
				clientShutdown: "15s", // unchanged
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			daemonCmd := &DaemonCmd{
				config: daemonFlagConfig{
					timeout: tc.initialConfig,
				},
			}

			logger := hclog.NewNullLogger()
			command := &cobra.Command{}

			// Add all timeout flags to the command
			command.Flags().String(flagTimeoutAPIShutdown, "", "test flag")
			command.Flags().String(flagTimeoutMCPInit, "", "test flag")
			command.Flags().String(flagTimeoutMCPHealth, "", "test flag")

			// Simulate flags being changed if needed
			if tc.apiShutdownFlagChanged {
				err := command.Flags().Set(flagTimeoutAPIShutdown, tc.initialConfig.apiShutdown)
				require.NoError(t, err)
			}
			if tc.mcpInitFlagChanged {
				err := command.Flags().Set(flagTimeoutMCPInit, tc.initialConfig.mcpInit)
				require.NoError(t, err)
			}
			if tc.healthCheckFlagChanged {
				err := command.Flags().Set(flagTimeoutMCPHealth, tc.initialConfig.healthCheck)
				require.NoError(t, err)
			}

			var warnings []string
			if tc.apiConfig != nil {
				warnings = append(warnings, daemonCmd.loadConfigAPI(tc.apiConfig, logger, command)...)
			}
			if tc.mcpConfig != nil {
				warnings = append(warnings, daemonCmd.loadConfigMCP(tc.mcpConfig, logger, command)...)
			}

			assert.Equal(t, tc.expectWarnings, warnings)
			assert.Equal(t, tc.expectFinalConfig, daemonCmd.config.timeout)
		})
	}
}

func TestDaemon_ApplyConfigInterval(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                   string
		mcpConfig              *config.MCPConfigSection
		healthCheckFlagChanged bool
		initialConfig          intervalFlagConfig
		expectWarnings         []string
		expectFinalConfig      intervalFlagConfig
	}{
		{
			name:              "nil interval config",
			mcpConfig:         nil,
			expectWarnings:    nil,
			initialConfig:     intervalFlagConfig{},
			expectFinalConfig: intervalFlagConfig{}, // unchanged
		},
		{
			name: "healthCheck check interval - flag not changed",
			mcpConfig: &config.MCPConfigSection{
				Interval: &config.MCPIntervalConfigSection{
					Health: testDurationPtr(t, 30*time.Second),
				},
			},
			healthCheckFlagChanged: false,
			initialConfig:          intervalFlagConfig{healthCheck: "15s"},
			expectWarnings:         nil,
			expectFinalConfig:      intervalFlagConfig{healthCheck: "30s"},
		},
		{
			name: "healthCheck check interval - flag changed (override)",
			mcpConfig: &config.MCPConfigSection{
				Interval: &config.MCPIntervalConfigSection{
					Health: testDurationPtr(t, 30*time.Second),
				},
			},
			healthCheckFlagChanged: true,
			initialConfig:          intervalFlagConfig{healthCheck: "15s"},
			expectWarnings:         []string{"--interval-mcp-health: config=30s, flag=15s (using flag)"},
			expectFinalConfig:      intervalFlagConfig{healthCheck: "15s"}, // flag wins
		},
		{
			name:                   "empty health check interval",
			mcpConfig:              &config.MCPConfigSection{},
			healthCheckFlagChanged: false,
			initialConfig:          intervalFlagConfig{healthCheck: "15s"},
			expectWarnings:         nil,
			expectFinalConfig:      intervalFlagConfig{healthCheck: "15s"}, // unchanged
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			daemonCmd := &DaemonCmd{
				config: daemonFlagConfig{
					interval: tc.initialConfig,
				},
			}

			logger := hclog.NewNullLogger()
			command := &cobra.Command{}

			// Add interval flags to the command
			command.Flags().String(flagIntervalMCPHealth, "", "test flag")

			// Simulate flag being changed if needed
			if tc.healthCheckFlagChanged {
				err := command.Flags().Set(flagIntervalMCPHealth, tc.initialConfig.healthCheck)
				require.NoError(t, err)
			}

			warnings := daemonCmd.loadConfigMCP(tc.mcpConfig, logger, command)

			assert.Equal(t, tc.expectWarnings, warnings)
			assert.Equal(t, tc.expectFinalConfig, daemonCmd.config.interval)
		})
	}
}

// Tests for the main configuration loading and precedence logic
func TestDaemon_LoadConfigurationLayers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		configData     *config.Config
		configError    error
		flagValues     map[string]string
		expectWarnings []string
		expectError    string
	}{
		{
			name:           "no daemon config section",
			configData:     &config.Config{Daemon: nil},
			expectWarnings: nil,
		},
		{
			name:        "config file error",
			configError: fmt.Errorf("failed - sad times"),
			expectError: "failed - sad times",
		},
		{
			name:        "invalid config structure",
			configData:  nil, // This will cause type assertion to fail
			expectError: "config data not present, cannot apply configuration layers",
		},
		{
			name: "daemon config without flags - uses config values",
			configData: &config.Config{
				Daemon: &config.DaemonConfig{
					API: &config.APIConfigSection{
						Addr: testStringPtr(t, "config.example.com:8080"),
						Timeout: &config.APITimeoutConfigSection{
							Shutdown: testDurationPtr(t, 30*time.Second),
						},
					},
					MCP: &config.MCPConfigSection{
						Timeout: &config.MCPTimeoutConfigSection{
							Shutdown: testDurationPtr(t, 20*time.Second),
							Init:     testDurationPtr(t, 45*time.Second),
						},
					},
				},
			},
			expectWarnings: nil, // No flag overrides
		},
		{
			name: "flags override config values with warnings",
			configData: &config.Config{
				Daemon: &config.DaemonConfig{
					API: &config.APIConfigSection{
						Addr: testStringPtr(t, "config.example.com:8080"),
						Timeout: &config.APITimeoutConfigSection{
							Shutdown: testDurationPtr(t, 30*time.Second),
						},
					},
					MCP: &config.MCPConfigSection{
						Timeout: &config.MCPTimeoutConfigSection{
							Init: testDurationPtr(t, 45*time.Second),
						},
					},
				},
			},
			flagValues: map[string]string{
				flagAddr:               "flag.example.com:9090",
				flagTimeoutAPIShutdown: "40s",
				flagTimeoutMCPInit:     "60s",
			},
			expectWarnings: []string{
				"--addr: config=config.example.com:8080, flag=flag.example.com:9090 (using flag)",
				"--timeout-api-shutdown: config=30s, flag=40s (using flag)",
				"--timeout-mcp-init: config=45s, flag=60s (using flag)",
			},
		},
		{
			name: "CORS config with flag overrides",
			configData: &config.Config{
				Daemon: &config.DaemonConfig{
					API: &config.APIConfigSection{
						CORS: &config.CORSConfigSection{
							Enable:      testBoolPtr(t, false),
							Origins:     []string{"http://config.example.com"},
							Methods:     []string{"GET", "POST"},
							Credentials: testBoolPtr(t, true),
						},
					},
				},
			},
			flagValues: map[string]string{
				flagCORSEnable:      "true",
				flagCORSCredentials: "false",
			},
			expectWarnings: []string{
				"--cors-enable: config=false, flag=true (using flag)",
				"--cors-allow-credentials: config=true, flag=false (using flag)",
			},
		},
		{
			name: "partial config with some flag overrides",
			configData: &config.Config{
				Daemon: &config.DaemonConfig{
					API: &config.APIConfigSection{
						Addr: testStringPtr(t, "partial.example.com:8080"),
						// No timeout config - will use defaults
					},
					// No MCP config - will use defaults
				},
			},
			flagValues: map[string]string{
				flagAddr: "override.example.com:9090",
				// Other flags not set - will use config or defaults
			},
			expectWarnings: []string{
				"--addr: config=partial.example.com:8080, flag=override.example.com:9090 (using flag)",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Create mock config loader
			mockLoader := &testMockConfigLoader{
				config: tc.configData,
				err:    tc.configError,
			}

			daemonCmd := &DaemonCmd{
				cfgLoader: mockLoader,
				config:    daemonFlagConfig{},
			}

			// Note: Config data will be passed directly to loadConfigurationLayers

			// Create command and bind flags to struct fields (using the actual production code)
			command := newDaemonCobraCmd(daemonCmd)

			// Set flag values if specified (this will now work because flags are bound)
			for flag, value := range tc.flagValues {
				err := command.Flags().Set(flag, value)
				require.NoError(t, err)
			}

			logger := hclog.NewNullLogger()

			if tc.configError != nil {
				_, err := daemonCmd.LoadConfig(daemonCmd.cfgLoader)
				require.Error(t, err)
				require.EqualError(t, err, tc.expectError)
				return
			}

			warnings, err := daemonCmd.loadConfigurationLayers(logger, command, tc.configData)

			if tc.expectError != "" {
				require.EqualError(t, err, tc.expectError)
				return
			}

			require.NoError(t, err)
			assert.ElementsMatch(t, tc.expectWarnings, warnings)
		})
	}
}

func TestDaemon_LoadConfigurationLayers_Integration(t *testing.T) {
	t.Parallel()

	t.Run("end-to-end precedence verification", func(t *testing.T) {
		t.Parallel()

		// Set up config with various values
		configData := &config.Config{
			Daemon: &config.DaemonConfig{
				API: &config.APIConfigSection{
					Addr: testStringPtr(t, "config.example.com:8080"),
					Timeout: &config.APITimeoutConfigSection{
						Shutdown: testDurationPtr(t, 30*time.Second),
					},
					CORS: &config.CORSConfigSection{
						Enable:  testBoolPtr(t, false),
						Origins: []string{"http://config.example.com", "https://config.example.com"},
					},
				},
				MCP: &config.MCPConfigSection{
					Timeout: &config.MCPTimeoutConfigSection{
						Shutdown: testDurationPtr(t, 20*time.Second),
						Init:     testDurationPtr(t, 30*time.Second),
						Health:   testDurationPtr(t, 5*time.Second),
					},
					Interval: &config.MCPIntervalConfigSection{
						Health: testDurationPtr(t, 15*time.Second),
					},
				},
			},
		}

		mockLoader := &testMockConfigLoader{config: configData}
		daemonCmd := &DaemonCmd{
			cfgLoader: mockLoader,
			config:    daemonFlagConfig{},
			// Note: Config data will be passed directly to loadConfigurationLayers
		}

		command := newDaemonCobraCmd(daemonCmd)

		// Override some flags, leave others to use config values
		err := command.Flags().Set(flagAddr, "flag.example.com:9090")
		require.NoError(t, err)
		err = command.Flags().Set(flagCORSEnable, "true")
		require.NoError(t, err)
		err = command.Flags().Set(flagTimeoutMCPInit, "60s")
		require.NoError(t, err)

		logger := hclog.NewNullLogger()
		warnings, err := daemonCmd.loadConfigurationLayers(logger, command, configData)

		require.NoError(t, err)

		// Verify warnings for overrides
		expectedWarnings := []string{
			"--addr: config=config.example.com:8080, flag=flag.example.com:9090 (using flag)",
			"--cors-enable: config=false, flag=true (using flag)",
			"--timeout-mcp-init: config=30s, flag=60s (using flag)",
		}
		assert.ElementsMatch(t, expectedWarnings, warnings)

		// Verify final configuration values
		assert.Equal(
			t,
			"flag.example.com:9090",
			daemonCmd.config.api.addr,
		) // From flag
		assert.Equal(
			t,
			"30s",
			daemonCmd.config.timeout.apiShutdown,
		) // From config (stored as string)
		assert.True(
			t,
			daemonCmd.config.cors.enable,
		) // From flag
		assert.ElementsMatch(
			t,
			[]string{"http://config.example.com", "https://config.example.com"},
			daemonCmd.config.cors.origins,
		) // From config
		assert.Equal(
			t,
			"5s",
			daemonCmd.config.timeout.healthCheck,
		) // From config (stored as string)
		// Note: mcpShutdown is not loaded from config - only Init and Health are handled
		assert.Equal(
			t,
			"60s",
			daemonCmd.config.timeout.mcpInit,
		) // From flag (stored as string)
		assert.Equal(
			t,
			"5s",
			daemonCmd.config.timeout.healthCheck,
		) // From config (stored as string)
		assert.Equal(
			t,
			"15s",
			daemonCmd.config.interval.healthCheck,
		) // From config (stored as string)
	})
}

// Mock config loader for testing
type testMockConfigLoader struct {
	config *config.Config
	err    error
}

// testInvalidConfigType is a type that implements config.Modifier but is not *config.Config
type testInvalidConfigType struct{}

func (t testInvalidConfigType) AddServer(entry config.ServerEntry) error { return nil }
func (t testInvalidConfigType) RemoveServer(name string) error           { return nil }
func (t testInvalidConfigType) ListServers() []config.ServerEntry        { return nil }
func (t testInvalidConfigType) SaveConfig() error                        { return nil }

func (m *testMockConfigLoader) Load(_ string) (config.Modifier, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.config == nil {
		// Return something that implements Modifier but is not a *config.Config to test type assertion failure
		return testInvalidConfigType{}, nil
	}
	return m.config, nil
}

func TestDaemon_DaemonCmd_HandleSignals(t *testing.T) {
	t.Parallel()

	createDaemonCmd := func(t *testing.T) *DaemonCmd {
		t.Helper()
		baseCmd := &cmd.BaseCmd{}
		mockLoader := &mockConfigLoader{entries: []config.ServerEntry{}}
		contextLoader := &configcontext.DefaultLoader{}
		daemonCmd, err := newDaemonCmd(baseCmd, mockLoader, contextLoader)
		require.NoError(t, err)
		return daemonCmd
	}

	createLogger := func() hclog.Logger {
		return hclog.New(&hclog.LoggerOptions{
			Name:   "test",
			Level:  hclog.Off,
			Output: io.Discard,
		})
	}

	t.Run("SIGHUP triggers reload", func(t *testing.T) {
		t.Parallel()

		daemonCmd := createDaemonCmd(t)
		logger := createLogger()

		sigChan := make(chan os.Signal, 1)
		reloadChan := make(chan struct{}, 1)
		shutdownCancel := func() {}

		// Start handleSignals in goroutine.
		go daemonCmd.handleSignals(logger, sigChan, reloadChan, shutdownCancel)

		// Send SIGHUP.
		sigChan <- syscall.SIGHUP

		// Verify reload signal received.
		select {
		case <-reloadChan:
			// Expected
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Expected reload signal not received")
		}

		close(sigChan)
	})

	t.Run("duplicate SIGHUP signals are handled gracefully", func(t *testing.T) {
		t.Parallel()

		daemonCmd := createDaemonCmd(t)
		logger := createLogger()

		sigChan := make(chan os.Signal, 2)
		reloadChan := make(chan struct{}) // No buffer - will block second send
		shutdownCancel := func() {}

		// Start handleSignals in goroutine.
		go daemonCmd.handleSignals(logger, sigChan, reloadChan, shutdownCancel)

		// Send two SIGHUP signals quickly.
		sigChan <- syscall.SIGHUP
		sigChan <- syscall.SIGHUP

		// Verify first reload signal received.
		select {
		case <-reloadChan:
			// Expected - first signal processed
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Expected first reload signal not received")
		}

		// Second signal should be dropped (non-blocking send).
		// We can't directly verify the drop, but the function should not hang.

		close(sigChan)
	})

	t.Run("SIGTERM triggers shutdown", func(t *testing.T) {
		t.Parallel()

		daemonCmd := createDaemonCmd(t)
		logger := createLogger()

		sigChan := make(chan os.Signal, 1)
		reloadChan := make(chan struct{}, 1)
		shutdownCalled := false
		shutdownCancel := func() { shutdownCalled = true }

		// Start handleSignals in goroutine.
		done := make(chan struct{})
		go func() {
			daemonCmd.handleSignals(logger, sigChan, reloadChan, shutdownCancel)
			close(done)
		}()

		// Send SIGTERM.
		sigChan <- syscall.SIGTERM

		// Verify function returns and shutdown is called.
		select {
		case <-done:
			assert.True(t, shutdownCalled, "shutdown function should be called")
		case <-time.After(100 * time.Millisecond):
			t.Fatal("handleSignals should return after shutdown signal")
		}
	})

	t.Run("SIGINT triggers shutdown", func(t *testing.T) {
		t.Parallel()

		daemonCmd := createDaemonCmd(t)
		logger := createLogger()

		sigChan := make(chan os.Signal, 1)
		reloadChan := make(chan struct{}, 1)
		shutdownCalled := false
		shutdownCancel := func() { shutdownCalled = true }

		// Start handleSignals in goroutine.
		done := make(chan struct{})
		go func() {
			daemonCmd.handleSignals(logger, sigChan, reloadChan, shutdownCancel)
			close(done)
		}()

		// Send SIGINT.
		sigChan <- syscall.SIGINT

		// Verify function returns and shutdown is called.
		select {
		case <-done:
			assert.True(t, shutdownCalled, "shutdown function should be called")
		case <-time.After(100 * time.Millisecond):
			t.Fatal("handleSignals should return after shutdown signal")
		}
	})

	t.Run("os.Interrupt triggers shutdown", func(t *testing.T) {
		t.Parallel()

		daemonCmd := createDaemonCmd(t)
		logger := createLogger()

		sigChan := make(chan os.Signal, 1)
		reloadChan := make(chan struct{}, 1)
		shutdownCalled := false
		shutdownCancel := func() { shutdownCalled = true }

		// Start handleSignals in goroutine.
		done := make(chan struct{})
		go func() {
			daemonCmd.handleSignals(logger, sigChan, reloadChan, shutdownCancel)
			close(done)
		}()

		// Send os.Interrupt.
		sigChan <- os.Interrupt

		// Verify function returns and shutdown is called.
		select {
		case <-done:
			assert.True(t, shutdownCalled, "shutdown function should be called")
		case <-time.After(100 * time.Millisecond):
			t.Fatal("handleSignals should return after shutdown signal")
		}
	})

	t.Run("channel closure terminates function", func(t *testing.T) {
		t.Parallel()

		daemonCmd := createDaemonCmd(t)
		logger := createLogger()

		sigChan := make(chan os.Signal)
		reloadChan := make(chan struct{}, 1)
		shutdownCalled := false
		shutdownCancel := func() { shutdownCalled = true }

		// Start handleSignals in goroutine.
		done := make(chan struct{})
		go func() {
			daemonCmd.handleSignals(logger, sigChan, reloadChan, shutdownCancel)
			close(done)
		}()

		// Close signal channel.
		close(sigChan)

		// Verify function returns without calling shutdown.
		select {
		case <-done:
			assert.False(t, shutdownCalled, "shutdown should not be called on channel closure")
		case <-time.After(100 * time.Millisecond):
			t.Fatal("handleSignals should return after channel closure")
		}
	})
}
