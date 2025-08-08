package daemon

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mozilla-ai/mcpd/v2/internal/config"
	configcontext "github.com/mozilla-ai/mcpd/v2/internal/context"
)

type testLogMessage struct {
	name    string
	level   hclog.Level
	message string
	args    []any
}

type testLoggerSink struct {
	messages []*testLogMessage
}

func (cs *testLoggerSink) Accept(name string, level hclog.Level, msg string, args ...any) {
	lm := &testLogMessage{
		name:    name,
		level:   level,
		message: msg,
		args:    args,
	}
	cs.messages = append(cs.messages, lm)
}

func TestIsValidAddr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		addr    string
		wantErr bool
	}{
		{"valid localhost numeric port", "localhost:8080", false},
		{"valid empty host numeric port", ":8080", false},
		{"valid IP address", "127.0.0.1:80", false},
		{"valid IPv6 address", "[::1]:443", false},
		{"valid named port", "localhost:http", false},
		{"missing port", "localhost", true},
		{"invalid port string", "localhost:notaport", true},
		{"missing host and port", "", true},
		{"missing host with invalid port", ":!@#", true},
		{"host only colon", "host:", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := IsValidAddr(tc.addr)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestNormalizeLogLevel(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		inputLevel    string
		expectedLevel hclog.Level
	}{
		// Standard hclog levels
		{
			name:          "Standard info level",
			inputLevel:    "info",
			expectedLevel: hclog.Info,
		},
		{
			name:          "Standard warn level",
			inputLevel:    "warn",
			expectedLevel: hclog.Warn,
		},
		{
			name:          "Standard debug level",
			inputLevel:    "debug",
			expectedLevel: hclog.Debug,
		},
		{
			name:          "Standard error level",
			inputLevel:    "error",
			expectedLevel: hclog.Error,
		},
		{
			name:          "Standard trace level",
			inputLevel:    "trace",
			expectedLevel: hclog.Trace,
		},
		{
			name:          "Off level",
			inputLevel:    "off",
			expectedLevel: hclog.Off,
		},
		// Normalized levels
		{
			name:          "Python's 'warning' should be Warn",
			inputLevel:    "warning",
			expectedLevel: hclog.Warn,
		},
		{
			name:          "'fatal' should be Error",
			inputLevel:    "fatal",
			expectedLevel: hclog.Error,
		},
		{
			name:          "'critical' should be Error",
			inputLevel:    "critical",
			expectedLevel: hclog.Error,
		},

		// Case-insensitivity tests
		{
			name:          "Uppercase 'INFO'",
			inputLevel:    "INFO",
			expectedLevel: hclog.Info,
		},
		{
			name:          "Mixed case 'WaRnInG'",
			inputLevel:    "WaRnInG",
			expectedLevel: hclog.Warn,
		},
		{
			name:          "Mixed case 'CrItiCaL'",
			inputLevel:    "CrItiCaL",
			expectedLevel: hclog.Error,
		},

		// Whitespace tests
		{
			name:          "Level with leading/trailing spaces",
			inputLevel:    "  debug  ",
			expectedLevel: hclog.Debug,
		},
		{
			name:          "Normalized level with spaces",
			inputLevel:    "\twarning \n",
			expectedLevel: hclog.Warn,
		},

		// Invalid input tests
		{
			name:          "Invalid level string",
			inputLevel:    "invalid-level",
			expectedLevel: hclog.NoLevel,
		},
		{
			name:          "Empty string input",
			inputLevel:    "",
			expectedLevel: hclog.NoLevel,
		},
		{
			name:          "Just whitespace",
			inputLevel:    "   ",
			expectedLevel: hclog.NoLevel,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			actualLevel := normalizeLogLevel(tc.inputLevel)
			assert.Equal(t, tc.expectedLevel, actualLevel)
		})
	}
}

func TestParseAndLogMCPMessage(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		inputLine     string
		expectedLevel hclog.Level
		expectedMsg   string
		expectLog     bool
		loggerLevel   hclog.Level
	}{
		{
			name:          "Standard INFO message with three parts",
			inputLine:     "INFO:mcp.server.runner:This is an info message",
			expectedLevel: hclog.Info,
			expectedMsg:   "This is an info message",
			expectLog:     true,
			loggerLevel:   hclog.Trace,
		},
		{
			name:          "Standard WARNING message with two parts",
			inputLine:     "WARNING:This is a warning message",
			expectedLevel: hclog.Warn,
			expectedMsg:   "This is a warning message",
			expectLog:     true,
			loggerLevel:   hclog.Trace,
		},
		{
			name:          "Standard ERROR message with colons",
			inputLine:     "ERROR:mcp.server.runner:Error: something failed: exit 1",
			expectedLevel: hclog.Error,
			expectedMsg:   "Error: something failed: exit 1",
			expectLog:     true,
			loggerLevel:   hclog.Trace,
		},
		{
			name:          "DEBUG message logged when logger level is low enough",
			inputLine:     "DEBUG:This is a debug message",
			expectedLevel: hclog.Debug,
			expectedMsg:   "This is a debug message",
			expectLog:     true,
			loggerLevel:   hclog.Trace,
		},
		{
			name:          "DEBUG message ignored when logger level is too high",
			inputLine:     "DEBUG:This debug message should be ignored",
			expectedLevel: hclog.Debug,
			expectLog:     false,
			loggerLevel:   hclog.Info,
		},
		{
			name:          "Line without a valid level prefix (e.g. stack trace)",
			inputLine:     "  File \"/path/to/script.py\", line 123, in my_func",
			expectedLevel: hclog.Info,
			expectedMsg:   "File \"/path/to/script.py\", line 123, in my_func",
			expectLog:     true,
			loggerLevel:   hclog.Trace,
		},
		{
			name:          "Line with a non-standard but parsable level",
			inputLine:     "CRITICAL:A critical failure occurred",
			expectedLevel: hclog.Error,
			expectedMsg:   "A critical failure occurred",
			expectLog:     true,
			loggerLevel:   hclog.Trace,
		},
		{
			name:        "Empty line",
			inputLine:   "",
			expectLog:   false,
			loggerLevel: hclog.Trace,
		},
		{
			name:        "Line with only whitespace",
			inputLine:   "   \t\n",
			expectLog:   false,
			loggerLevel: hclog.Trace,
		},
		{
			name:          "Line that looks like a level but has no message",
			inputLine:     "INFO:",
			expectedLevel: hclog.Info,
			expectedMsg:   "",
			expectLog:     true,
			loggerLevel:   hclog.Trace,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Prepare a test logger to intercept the log calls.
			customSink := &testLoggerSink{}
			logger := hclog.NewInterceptLogger(&hclog.LoggerOptions{
				Name:  "test-logger",
				Level: tc.loggerLevel,
			})
			logger.RegisterSink(customSink)

			parseAndLogMCPMessage(logger, tc.inputLine)
			logs := customSink.messages

			if !tc.expectLog {
				assert.Empty(t, logs, "Expected no logs, but logs were generated")
				return
			}

			// If we get here, we expect exactly one log entry
			assert.Len(t, logs, 1, "Expected exactly one log entry")

			logEntry := logs[0]
			assert.Equal(t, tc.expectedLevel, logEntry.level, "Logged with incorrect level")
			assert.Equal(t, tc.expectedMsg, logEntry.message, "Logged with incorrect message")
		})
	}
}

// mockMCPClientWithBehavior is a test implementation that allows controlling closing behavior
type mockMCPClientWithBehavior struct {
	closeDelay  time.Duration
	closeError  error
	closeCalled bool
	closedAt    time.Time
	mu          sync.Mutex
}

func newMockMCPClientWithBehavior(closeDelay time.Duration, closeError error) *mockMCPClientWithBehavior {
	return &mockMCPClientWithBehavior{
		closeDelay: closeDelay,
		closeError: closeError,
	}
}

func (m *mockMCPClientWithBehavior) Close() error {
	m.mu.Lock()
	m.closeCalled = true
	m.mu.Unlock()

	if m.closeDelay > 0 {
		time.Sleep(m.closeDelay)
	}

	m.mu.Lock()
	m.closedAt = time.Now()
	m.mu.Unlock()

	return m.closeError
}

func (m *mockMCPClientWithBehavior) wasClosed() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.closeCalled
}

// Implement remaining MCPClient interface methods with minimal implementations
func (m *mockMCPClientWithBehavior) Initialize(
	ctx context.Context,
	request mcp.InitializeRequest,
) (*mcp.InitializeResult, error) {
	return nil, nil
}

func (m *mockMCPClientWithBehavior) Ping(ctx context.Context) error {
	return nil
}

func (m *mockMCPClientWithBehavior) ListResourcesByPage(
	ctx context.Context,
	request mcp.ListResourcesRequest,
) (*mcp.ListResourcesResult, error) {
	return nil, nil
}

func (m *mockMCPClientWithBehavior) ListResources(
	ctx context.Context,
	request mcp.ListResourcesRequest,
) (*mcp.ListResourcesResult, error) {
	return nil, nil
}

func (m *mockMCPClientWithBehavior) ListResourceTemplatesByPage(
	ctx context.Context,
	request mcp.ListResourceTemplatesRequest,
) (*mcp.ListResourceTemplatesResult, error) {
	return nil, nil
}

func (m *mockMCPClientWithBehavior) ListResourceTemplates(
	ctx context.Context,
	request mcp.ListResourceTemplatesRequest,
) (*mcp.ListResourceTemplatesResult, error) {
	return nil, nil
}

func (m *mockMCPClientWithBehavior) ReadResource(
	ctx context.Context,
	request mcp.ReadResourceRequest,
) (*mcp.ReadResourceResult, error) {
	return nil, nil
}

func (m *mockMCPClientWithBehavior) Subscribe(ctx context.Context, request mcp.SubscribeRequest) error {
	return nil
}

func (m *mockMCPClientWithBehavior) Unsubscribe(ctx context.Context, request mcp.UnsubscribeRequest) error {
	return nil
}

func (m *mockMCPClientWithBehavior) ListPromptsByPage(
	ctx context.Context,
	request mcp.ListPromptsRequest,
) (*mcp.ListPromptsResult, error) {
	return nil, nil
}

func (m *mockMCPClientWithBehavior) ListPrompts(
	ctx context.Context,
	request mcp.ListPromptsRequest,
) (*mcp.ListPromptsResult, error) {
	return nil, nil
}

func (m *mockMCPClientWithBehavior) GetPrompt(
	ctx context.Context,
	request mcp.GetPromptRequest,
) (*mcp.GetPromptResult, error) {
	return nil, nil
}

func (m *mockMCPClientWithBehavior) ListToolsByPage(
	ctx context.Context,
	request mcp.ListToolsRequest,
) (*mcp.ListToolsResult, error) {
	return nil, nil
}

func (m *mockMCPClientWithBehavior) ListTools(
	ctx context.Context,
	request mcp.ListToolsRequest,
) (*mcp.ListToolsResult, error) {
	return nil, nil
}

func (m *mockMCPClientWithBehavior) CallTool(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	return nil, nil
}

func (m *mockMCPClientWithBehavior) SetLevel(ctx context.Context, request mcp.SetLevelRequest) error {
	return nil
}

func (m *mockMCPClientWithBehavior) Complete(
	ctx context.Context,
	request mcp.CompleteRequest,
) (*mcp.CompleteResult, error) {
	return nil, nil
}

func (m *mockMCPClientWithBehavior) OnNotification(handler func(notification mcp.JSONRPCNotification)) {
}

// mockConfigModifier implements config.Modifier for testing
type mockConfigModifier struct{}

func (m *mockConfigModifier) AddServer(entry config.ServerEntry) error {
	return nil
}

func (m *mockConfigModifier) RemoveServer(name string) error {
	return nil
}

func (m *mockConfigModifier) ListServers() []config.ServerEntry {
	return []config.ServerEntry{}
}

// mockConfigLoader implements config.Loader for testing
type mockConfigLoader struct{}

func (m *mockConfigLoader) Load(path string) (config.Modifier, error) {
	return &mockConfigModifier{}, nil
}

// mockContextModifier implements configcontext.Modifier for testing
type mockContextModifier struct{}

func (m *mockContextModifier) Get(name string) (configcontext.ServerExecutionContext, bool) {
	return configcontext.ServerExecutionContext{}, false
}

func (m *mockContextModifier) Upsert(ctx configcontext.ServerExecutionContext) (configcontext.UpsertResult, error) {
	return configcontext.Created, nil
}

func (m *mockContextModifier) List() []configcontext.ServerExecutionContext {
	return []configcontext.ServerExecutionContext{}
}

// mockContextLoader implements configcontext.Loader for testing
type mockContextLoader struct{}

func (m *mockContextLoader) Load(path string) (configcontext.Modifier, error) {
	return &mockContextModifier{}, nil
}

// Test that client closing happens with proper timeout handling
func TestDaemon_ClientClosingWithTimeout(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		setupClients    func(*Daemon)
		expectedTimeout bool
		expectedLogs    []string
		notExpectedLogs []string
	}{
		{
			name: "Fast close - completes immediately",
			setupClients: func(d *Daemon) {
				client := newMockMCPClientWithBehavior(0, nil)
				d.clientManager.Add("fast-server", client, []string{"tool1"})
			},
			expectedTimeout: false,
			expectedLogs:    []string{"Closing client fast-server", "Closed client fast-server"},
			notExpectedLogs: []string{"Timeout"},
		},
		{
			name: "Slow close - takes 2s (under 5s timeout)",
			setupClients: func(d *Daemon) {
				client := newMockMCPClientWithBehavior(2*time.Second, nil)
				d.clientManager.Add("slow-server", client, []string{"tool1"})
			},
			expectedTimeout: false,
			expectedLogs:    []string{"Closing client slow-server", "Closed client slow-server"},
			notExpectedLogs: []string{"Timeout"},
		},
		{
			name: "Timeout - takes 6s (exceeds 5s timeout)",
			setupClients: func(d *Daemon) {
				client := newMockMCPClientWithBehavior(6*time.Second, nil)
				d.clientManager.Add("timeout-server", client, []string{"tool1"})
			},
			expectedTimeout: true,
			expectedLogs:    []string{"Closing client timeout-server", "Timeout (5s) closing client timeout-server"},
			notExpectedLogs: []string{"Closed client timeout-server"}, // Should NOT see this due to timeout
		},
		{
			name: "Error on close but still closes quickly",
			setupClients: func(d *Daemon) {
				client := newMockMCPClientWithBehavior(100*time.Millisecond, errors.New("close error"))
				d.clientManager.Add("error-server", client, []string{"tool1"})
			},
			expectedTimeout: false,
			expectedLogs:    []string{"Closing client error-server", "Closed client error-server"},
			notExpectedLogs: []string{"Timeout"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Create a test logger with sink to capture logs
			customSink := &testLoggerSink{}
			logger := hclog.NewInterceptLogger(&hclog.LoggerOptions{
				Name:  "test-daemon",
				Level: hclog.Debug,
			})
			logger.RegisterSink(customSink)

			opts, err := NewDaemonOpts(logger, &mockConfigLoader{}, &mockContextLoader{})
			require.NoError(t, err)
			daemon, err := NewDaemon(":8080", opts)
			require.NoError(t, err)

			// Setup clients for this test case
			tc.setupClients(daemon)
			ctx, cancel := context.WithCancel(context.Background())
			daemonDone := make(chan error)
			go func() {
				daemonDone <- daemon.StartAndManage(ctx)
			}()
			time.Sleep(100 * time.Millisecond)
			startTime := time.Now()
			// Cancel context to trigger shutdown
			cancel()

			// Wait for daemon to finish with timeout
			select {
			case <-daemonDone:
				// Daemon finished
			case <-time.After(10 * time.Second):
				t.Fatal("Daemon did not shut down in time")
			}

			shutdownDuration := time.Since(startTime)

			// The timeout of the actual function (as best we know)
			standardTimeout := 5 * time.Second

			if tc.expectedTimeout {
				assert.Greater(
					t,
					shutdownDuration,
					standardTimeout-1*time.Second,
					"Should wait close to timeout duration",
				)
				assert.Less(
					t,
					shutdownDuration,
					standardTimeout+1*time.Second,
					"Should not wait much longer than timeout",
				)
			} else {
				assert.Less(t, shutdownDuration, 5*time.Second, "Should complete before timeout")
			}

			// Verify expected logs
			logs := customSink.messages
			for _, expectedLog := range tc.expectedLogs {
				found := false
				for _, log := range logs {
					if strings.Contains(log.message, expectedLog) {
						found = true
						break
					}
				}
				assert.True(t, found, "Expected log message not found: %s", expectedLog)
			}

			// Verify logs we should NOT see
			for _, notExpectedLog := range tc.notExpectedLogs {
				found := false
				for _, log := range logs {
					if strings.Contains(log.message, notExpectedLog) {
						found = true
						break
					}
				}
				assert.False(t, found, "Unexpected log message found: %s", notExpectedLog)
			}
		})
	}
}

// Test that multiple clients close concurrently, not sequentially
func TestDaemon_MultipleClientsCloseConcurrently(t *testing.T) {
	t.Parallel()

	// Create a test logger with sink
	customSink := &testLoggerSink{}
	logger := hclog.NewInterceptLogger(&hclog.LoggerOptions{
		Name:  "test-daemon",
		Level: hclog.Debug,
	})
	logger.RegisterSink(customSink)

	opts, err := NewDaemonOpts(logger, &mockConfigLoader{}, &mockContextLoader{})
	require.NoError(t, err)
	daemon, err := NewDaemon(":8081", opts)
	require.NoError(t, err)

	// Add multiple clients with different delays
	clients := map[string]*mockMCPClientWithBehavior{
		"fast-1":    newMockMCPClientWithBehavior(100*time.Millisecond, nil),
		"slow-1":    newMockMCPClientWithBehavior(2*time.Second, nil),
		"slow-2":    newMockMCPClientWithBehavior(3*time.Second, nil),
		"timeout-1": newMockMCPClientWithBehavior(6*time.Second, nil), // Will timeout
	}

	for name, client := range clients {
		daemon.clientManager.Add(name, client, []string{"tool"})
	}

	ctx, cancel := context.WithCancel(context.Background())
	daemonDone := make(chan error)
	go func() {
		daemonDone <- daemon.StartAndManage(ctx)
	}()

	// Give daemon time to start
	time.Sleep(100 * time.Millisecond)

	// Record start time
	startTime := time.Now()
	cancel()

	// Wait for daemon to finish
	select {
	case <-daemonDone:
		// Daemon finished
	case <-time.After(10 * time.Second):
		t.Fatal("Daemon did not shut down in time")
	}

	shutdownDuration := time.Since(startTime)

	// The timeout of the actual function (as best we know)
	standardTimeout := 5 * time.Second

	// With concurrent closing, total time should be about 5s (timeout) not the sum of all delays
	// If they closed sequentially, it would take 100ms + 2s + 3s + 6s = 11.1s
	assert.Less(
		t,
		shutdownDuration,
		standardTimeout+2*time.Second,
		"Clients should close concurrently, not sequentially",
	)
	assert.Greater(t, shutdownDuration, standardTimeout-1*time.Second, "Should wait for timeout on slowest client")

	// Verify that non-timeout clients actually closed
	assert.True(t, clients["fast-1"].wasClosed(), "Fast client 1 should have closed")
	assert.True(t, clients["slow-1"].wasClosed(), "Slow client 1 should have closed")
	assert.True(t, clients["slow-2"].wasClosed(), "Slow client 2 should have closed")
	// timeout-1 may or may not have completed closing due to timeout
}

// TestDaemon_CloseAllClients_Direct tests the closeAllClients method directly
func TestDaemon_CloseAllClients_Direct(t *testing.T) {
	t.Parallel()

	// Create a test logger with sink to capture logs
	customSink := &testLoggerSink{}
	logger := hclog.NewInterceptLogger(&hclog.LoggerOptions{
		Name:  "test-daemon",
		Level: hclog.Debug,
	})
	logger.RegisterSink(customSink)

	opts, err := NewDaemonOpts(logger, &mockConfigLoader{}, &mockContextLoader{})
	require.NoError(t, err)
	daemon, err := NewDaemon(":8082", opts)
	require.NoError(t, err)

	fastClient := newMockMCPClientWithBehavior(100*time.Millisecond, nil)
	slowClient := newMockMCPClientWithBehavior(2*time.Second, nil)
	daemon.clientManager.Add("fast-client", fastClient, []string{"tool1"})
	daemon.clientManager.Add("slow-client", slowClient, []string{"tool2"})

	// Record start time
	startTime := time.Now()
	daemon.closeAllClients()

	// Should complete in ~2 seconds (slow client time), not 2.1 seconds (sum)
	elapsed := time.Since(startTime)
	assert.Less(t, elapsed, 3*time.Second, "Should complete in parallel")
	assert.Greater(t, elapsed, 1900*time.Millisecond, "Should wait for slowest client")

	// Verify clients were closed
	assert.True(t, fastClient.wasClosed(), "Fast client should be closed")
	assert.True(t, slowClient.wasClosed(), "Slow client should be closed")

	// Verify expected logs
	logs := customSink.messages
	found := 0
	for _, log := range logs {
		if log.message == "Shutting down MCP servers and client connections" ||
			log.message == "Closing client fast-client" ||
			log.message == "Closing client slow-client" ||
			log.message == "Closed client fast-client" ||
			log.message == "Closed client slow-client" {
			found++
		}
	}
	assert.GreaterOrEqual(t, found, 5, "Should have logged all expected messages")
}

// TestDaemon_CloseAllClients_EmptyManager tests behavior with no clients
func TestDaemon_CloseAllClients_EmptyManager(t *testing.T) {
	t.Parallel()

	// Create a test logger with sink
	customSink := &testLoggerSink{}
	logger := hclog.NewInterceptLogger(&hclog.LoggerOptions{
		Name:  "test-daemon",
		Level: hclog.Debug,
	})
	logger.RegisterSink(customSink)

	opts, err := NewDaemonOpts(logger, &mockConfigLoader{}, &mockContextLoader{})
	require.NoError(t, err)
	daemon, err := NewDaemon(":8083", opts)
	require.NoError(t, err)

	// Don't add any clients

	// Record start time
	startTime := time.Now()
	daemon.closeAllClients()

	// Should complete very quickly
	elapsed := time.Since(startTime)
	assert.Less(t, elapsed, 100*time.Millisecond, "Should complete immediately with no clients")

	// Verify expected log
	logs := customSink.messages
	shutdownLogFound := false
	for _, log := range logs {
		if log.message == "Shutting down MCP servers and client connections" {
			shutdownLogFound = true
			break
		}
	}
	assert.True(t, shutdownLogFound, "Should have logged shutdown message")
}

// TestDaemon_CloseClientWithTimeout_Direct tests the closeClientWithTimeout method directly
func TestDaemon_CloseClientWithTimeout_Direct(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		clientDelay     time.Duration
		timeout         time.Duration
		expectTimeout   bool
		expectedMaxTime time.Duration
	}{
		{
			name:            "Fast close under timeout",
			clientDelay:     100 * time.Millisecond,
			timeout:         1 * time.Second,
			expectTimeout:   false,
			expectedMaxTime: 500 * time.Millisecond,
		},
		{
			name:            "Slow close exceeds timeout",
			clientDelay:     2 * time.Second,
			timeout:         500 * time.Millisecond,
			expectTimeout:   true,
			expectedMaxTime: 1 * time.Second, // timeout + buffer
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			customSink := &testLoggerSink{}
			logger := hclog.NewInterceptLogger(&hclog.LoggerOptions{
				Name:  "test-daemon",
				Level: hclog.Debug,
			})
			logger.RegisterSink(customSink)

			opts, err := NewDaemonOpts(logger, &mockConfigLoader{}, &mockContextLoader{})
			require.NoError(t, err)
			daemon, err := NewDaemon(":8084", opts)
			require.NoError(t, err)

			testClient := newMockMCPClientWithBehavior(tc.clientDelay, nil)

			// Record start time
			startTime := time.Now()
			daemon.closeClientWithTimeout("test-client", testClient, tc.timeout)
			elapsed := time.Since(startTime)

			assert.Less(t, elapsed, tc.expectedMaxTime, "Should not take longer than expected")

			// Verify logs based on expectation
			logs := customSink.messages
			if tc.expectTimeout {
				// Should have timeout warning
				found := false
				for _, log := range logs {
					if log.level == hclog.Warn && log.message != "" {
						found = true
						break
					}
				}
				assert.True(t, found, "Should have timeout warning log")
			} else {
				// Should have success logs
				found := 0
				for _, log := range logs {
					if log.message == "Closing client test-client" ||
						log.message == "Closed client test-client" {
						found++
					}
				}
				assert.Equal(t, 2, found, "Should have closing and closed messages")
			}
		})
	}
}
