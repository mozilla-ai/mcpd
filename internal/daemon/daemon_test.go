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

	"github.com/mozilla-ai/mcpd/internal/config"
	configcontext "github.com/mozilla-ai/mcpd/internal/context"
	"github.com/mozilla-ai/mcpd/internal/runtime"
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

// mockMCPClientWithStuckPing simulates an MCP client with a Ping method that blocks
// for a fixed duration without respecting context cancellation.
type mockMCPClientWithStuckPing struct {
	pingDelay    time.Duration
	pingStarted  chan struct{}
	pingFinished chan struct{}
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

func (cs *testLoggerSink) Accept(name string, level hclog.Level, msg string, args ...any) {
	lm := &testLogMessage{
		name:    name,
		level:   level,
		message: msg,
		args:    args,
	}
	cs.messages = append(cs.messages, lm)
}

func newMockMCPClientWithStuckPing(delay time.Duration) *mockMCPClientWithStuckPing {
	return &mockMCPClientWithStuckPing{
		pingDelay:    delay,
		pingStarted:  make(chan struct{}),
		pingFinished: make(chan struct{}),
	}
}

func (m *mockMCPClientWithStuckPing) Ping(ctx context.Context) error {
	close(m.pingStarted)
	// Ignore context and block for the full duration.
	time.Sleep(m.pingDelay)
	close(m.pingFinished)
	return nil
}

func (m *mockMCPClientWithStuckPing) Close() error {
	return nil
}

func (m *mockMCPClientWithStuckPing) Initialize(
	ctx context.Context,
	request mcp.InitializeRequest,
) (*mcp.InitializeResult, error) {
	return nil, nil
}

func (m *mockMCPClientWithStuckPing) ListResourcesByPage(
	ctx context.Context,
	request mcp.ListResourcesRequest,
) (*mcp.ListResourcesResult, error) {
	return nil, nil
}

func (m *mockMCPClientWithStuckPing) ListResources(
	ctx context.Context,
	request mcp.ListResourcesRequest,
) (*mcp.ListResourcesResult, error) {
	return nil, nil
}

func (m *mockMCPClientWithStuckPing) ListResourceTemplatesByPage(
	ctx context.Context,
	request mcp.ListResourceTemplatesRequest,
) (*mcp.ListResourceTemplatesResult, error) {
	return nil, nil
}

func (m *mockMCPClientWithStuckPing) ListResourceTemplates(
	ctx context.Context,
	request mcp.ListResourceTemplatesRequest,
) (*mcp.ListResourceTemplatesResult, error) {
	return nil, nil
}

func (m *mockMCPClientWithStuckPing) ReadResource(
	ctx context.Context,
	request mcp.ReadResourceRequest,
) (*mcp.ReadResourceResult, error) {
	return nil, nil
}

func (m *mockMCPClientWithStuckPing) Subscribe(ctx context.Context, request mcp.SubscribeRequest) error {
	return nil
}

func (m *mockMCPClientWithStuckPing) Unsubscribe(ctx context.Context, request mcp.UnsubscribeRequest) error {
	return nil
}

func (m *mockMCPClientWithStuckPing) ListPromptsByPage(
	ctx context.Context,
	request mcp.ListPromptsRequest,
) (*mcp.ListPromptsResult, error) {
	return nil, nil
}

func (m *mockMCPClientWithStuckPing) ListPrompts(
	ctx context.Context,
	request mcp.ListPromptsRequest,
) (*mcp.ListPromptsResult, error) {
	return nil, nil
}

func (m *mockMCPClientWithStuckPing) GetPrompt(
	ctx context.Context,
	request mcp.GetPromptRequest,
) (*mcp.GetPromptResult, error) {
	return nil, nil
}

func (m *mockMCPClientWithStuckPing) ListToolsByPage(
	ctx context.Context,
	request mcp.ListToolsRequest,
) (*mcp.ListToolsResult, error) {
	return nil, nil
}

func (m *mockMCPClientWithStuckPing) ListTools(
	ctx context.Context,
	request mcp.ListToolsRequest,
) (*mcp.ListToolsResult, error) {
	return nil, nil
}

func (m *mockMCPClientWithStuckPing) CallTool(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	return nil, nil
}

func (m *mockMCPClientWithStuckPing) SetLevel(ctx context.Context, request mcp.SetLevelRequest) error {
	return nil
}

func (m *mockMCPClientWithStuckPing) Complete(
	ctx context.Context,
	request mcp.CompleteRequest,
) (*mcp.CompleteResult, error) {
	return nil, nil
}

func (m *mockMCPClientWithStuckPing) OnNotification(handler func(notification mcp.JSONRPCNotification)) {
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

			servers := []runtime.Server{
				{
					ServerEntry:            config.ServerEntry{},
					ServerExecutionContext: configcontext.ServerExecutionContext{},
				},
			}
			deps, err := NewDependencies(logger, ":8080", servers)
			require.NoError(t, err)
			daemon, err := NewDaemon(deps)
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

	servers := []runtime.Server{
		{
			ServerEntry:            config.ServerEntry{},
			ServerExecutionContext: configcontext.ServerExecutionContext{},
		},
	}
	deps, err := NewDependencies(logger, ":8081", servers)
	require.NoError(t, err)
	daemon, err := NewDaemon(deps)
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

	servers := []runtime.Server{
		{
			ServerEntry:            config.ServerEntry{},
			ServerExecutionContext: configcontext.ServerExecutionContext{},
		},
	}
	deps, err := NewDependencies(logger, ":8082", servers)
	require.NoError(t, err)
	daemon, err := NewDaemon(deps)
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

	servers := []runtime.Server{
		{
			ServerEntry:            config.ServerEntry{},
			ServerExecutionContext: configcontext.ServerExecutionContext{},
		},
	}
	deps, err := NewDependencies(logger, ":8083", servers)
	require.NoError(t, err)
	daemon, err := NewDaemon(deps)
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
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			customSink := &testLoggerSink{}
			logger := hclog.NewInterceptLogger(&hclog.LoggerOptions{
				Name:  "test-daemon",
				Level: hclog.Debug,
			})
			logger.RegisterSink(customSink)

			servers := []runtime.Server{
				{
					ServerEntry:            config.ServerEntry{},
					ServerExecutionContext: configcontext.ServerExecutionContext{},
				},
			}
			deps, err := NewDependencies(logger, ":8084", servers)
			require.NoError(t, err)
			daemon, err := NewDaemon(deps)
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

// TestDaemon_StartMCPServer_NoTools tests that servers without tools are rejected
func TestDaemon_StartMCPServer_NoTools(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		serverName    string
		tools         []string
		expectedError string
	}{
		{
			name:          "server with no tools should fail",
			serverName:    "test-no-tools",
			tools:         []string{},
			expectedError: "server 'test-no-tools' has no tools configured - MCP servers require at least one tool to function",
		},
		{
			name:          "server with empty tools list should fail",
			serverName:    "test-empty-tools",
			tools:         []string{},
			expectedError: "server 'test-empty-tools' has no tools configured - MCP servers require at least one tool to function",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			t.Cleanup(cancel)

			logger := hclog.NewNullLogger()

			// Create a daemon with a dummy server to satisfy validation.
			dummyServer := runtime.Server{
				ServerEntry: config.ServerEntry{
					Name:    "dummy",
					Package: "uvx::dummy",
				},
			}
			deps, err := NewDependencies(logger, ":8085", []runtime.Server{dummyServer})
			require.NoError(t, err)
			daemon, err := NewDaemon(deps)
			require.NoError(t, err)

			// Create server with specified tools.
			server := runtime.Server{
				ServerEntry: config.ServerEntry{
					Name:    tc.serverName,
					Package: "uvx::test-package",
					Tools:   tc.tools,
				},
				ServerExecutionContext: configcontext.ServerExecutionContext{},
			}

			err = daemon.startMCPServer(ctx, server)
			require.EqualError(t, err, tc.expectedError)
		})
	}
}

// TestDaemon_CloseClientWithTimeout_ReturnValue tests the return value behavior of closeClientWithTimeout
func TestDaemon_CloseClientWithTimeout_ReturnValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		clientDelay    time.Duration
		timeout        time.Duration
		expectedResult bool
	}{
		{
			name:           "successful close returns true",
			clientDelay:    50 * time.Millisecond,
			timeout:        200 * time.Millisecond,
			expectedResult: true,
		},
		{
			name:           "timeout returns false",
			clientDelay:    500 * time.Millisecond,
			timeout:        100 * time.Millisecond,
			expectedResult: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			logger := hclog.NewNullLogger()
			servers := []runtime.Server{
				{
					ServerEntry:            config.ServerEntry{},
					ServerExecutionContext: configcontext.ServerExecutionContext{},
				},
			}
			deps, err := NewDependencies(logger, ":8085", servers)
			require.NoError(t, err)
			daemon, err := NewDaemon(deps)
			require.NoError(t, err)

			testClient := newMockMCPClientWithBehavior(tc.clientDelay, nil)

			result := daemon.closeClientWithTimeout("test-client", testClient, tc.timeout)
			assert.Equal(t, tc.expectedResult, result)
		})
	}
}

// TestDaemon_StopMCPServer tests the stopMCPServer method
func TestDaemon_StopMCPServer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		serverExists   bool
		clientDelay    time.Duration
		timeout        time.Duration
		expectError    bool
		expectedErrMsg string
	}{
		{
			name:         "successful stop",
			serverExists: true,
			clientDelay:  50 * time.Millisecond,
			timeout:      200 * time.Millisecond,
			expectError:  false,
		},
		{
			name:           "server not found",
			serverExists:   false,
			expectError:    true,
			expectedErrMsg: "server 'nonexistent' not found",
		},
		{
			name:           "timeout during stop",
			serverExists:   true,
			clientDelay:    500 * time.Millisecond,
			timeout:        100 * time.Millisecond,
			expectError:    true,
			expectedErrMsg: "failed to stop within timeout",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			customSink := &testLoggerSink{}
			logger := hclog.NewInterceptLogger(&hclog.LoggerOptions{
				Name:  "test-daemon",
				Level: hclog.Debug,
			})
			logger.RegisterSink(customSink)

			clientManager := NewClientManager()
			healthTracker := NewHealthTracker([]string{})
			daemon := &Daemon{
				logger:                logger,
				clientManager:         clientManager,
				healthTracker:         healthTracker,
				clientShutdownTimeout: tc.timeout,
			}

			serverName := "test-server"
			if !tc.serverExists {
				serverName = "nonexistent"
			}

			if tc.serverExists {
				testClient := newMockMCPClientWithBehavior(tc.clientDelay, nil)
				clientManager.Add("test-server", testClient, []string{"tool1"})
				healthTracker.Add("test-server")
			}

			err := daemon.stopMCPServer(serverName)

			if tc.expectError {
				require.Error(t, err)
				if tc.expectedErrMsg != "" {
					assert.Contains(t, err.Error(), tc.expectedErrMsg)
				}
			} else {
				require.NoError(t, err)

				// Verify server was removed from managers
				_, exists := clientManager.Client("test-server")
				assert.False(t, exists, "Client should be removed from manager")

				_, err := healthTracker.Status("test-server")
				assert.Error(t, err, "Server should be removed from health tracker")
			}

			// Verify appropriate logs
			if tc.serverExists && !tc.expectError {
				// Should have success log
				found := false
				for _, log := range customSink.messages {
					if strings.Contains(log.message, "stopped successfully") {
						found = true
						break
					}
				}
				assert.True(t, found, "Should have success log")
			} else if tc.serverExists && tc.expectError {
				// Should have error log for timeout
				found := false
				for _, log := range customSink.messages {
					if log.level == hclog.Error && strings.Contains(log.message, "timed out") {
						found = true
						break
					}
				}
				assert.True(t, found, "Should have timeout error log")
			}
		})
	}
}

// TestDaemon_DockerRuntimeSupport tests that Docker runtime is properly supported
func TestDaemon_DockerRuntimeSupport(t *testing.T) {
	t.Parallel()

	// Test that Docker is in default supported runtimes
	defaultRuntimes := runtime.DefaultSupportedRuntimes()
	require.Contains(t, defaultRuntimes, runtime.Docker, "Docker should be in default supported runtimes")

	// Test that a Docker server configuration is valid
	dockerServer := runtime.Server{
		ServerEntry: config.ServerEntry{
			Name:    "docker-test",
			Package: "docker::test/mcp-server@latest",
			Tools:   []string{"test-tool"},
		},
		ServerExecutionContext: configcontext.ServerExecutionContext{
			Env: map[string]string{
				"TEST_ENV": "test-value",
			},
			Args: []string{"--test-arg", "value"},
		},
	}

	// Verify the runtime is extracted correctly
	runtimeType := dockerServer.Runtime()
	require.Equal(t, "docker", runtimeType, "Should extract docker as runtime")

	// Test daemon recognizes Docker as supported
	logger := hclog.NewNullLogger()
	deps, err := NewDependencies(logger, ":8085", []runtime.Server{dockerServer})
	require.NoError(t, err)
	daemon, err := NewDaemon(deps)
	require.NoError(t, err)

	// Verify Docker is in the daemon's supported runtimes
	require.Contains(t, daemon.supportedRuntimes, runtime.Docker, "Daemon should support Docker runtime")
}

func TestDaemon_DockerVolumeArguments(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                string
		volumes             map[string]config.VolumeEntry
		volumeContext       map[string]string
		expectedVolumeFlags []string
	}{
		{
			name:                "no volumes",
			volumes:             map[string]config.VolumeEntry{},
			volumeContext:       map[string]string{},
			expectedVolumeFlags: []string{},
		},
		{
			name: "single required volume",
			volumes: map[string]config.VolumeEntry{
				"workspace": {
					Path:     "/workspace",
					Required: true,
				},
			},
			volumeContext: map[string]string{
				"workspace": "/Users/foo/repos/mcpd",
			},
			expectedVolumeFlags: []string{
				"--volume", "/Users/foo/repos/mcpd:/workspace",
			},
		},
		{
			name: "multiple volumes",
			volumes: map[string]config.VolumeEntry{
				"workspace": {
					Path:     "/workspace",
					Required: true,
				},
				"kubeconfig": {
					Path:     "/home/nonroot/.kube/config",
					Required: true,
				},
			},
			volumeContext: map[string]string{
				"workspace":  "/Users/foo/repos",
				"kubeconfig": "~/.kube/config",
			},
			expectedVolumeFlags: []string{
				"--volume", "/Users/foo/repos:/workspace",
				"--volume", "~/.kube/config:/home/nonroot/.kube/config",
			},
		},
		{
			name: "named docker volume",
			volumes: map[string]config.VolumeEntry{
				"data": {
					Path:     "/data",
					Required: true,
				},
			},
			volumeContext: map[string]string{
				"data": "mcp-data",
			},
			expectedVolumeFlags: []string{
				"--volume", "mcp-data:/data",
			},
		},
		{
			name: "optional volume not configured",
			volumes: map[string]config.VolumeEntry{
				"workspace": {
					Path:     "/workspace",
					Required: true,
				},
				"cache": {
					Path:     "/cache",
					Required: false,
				},
			},
			volumeContext: map[string]string{
				"workspace": "/Users/foo/repos",
			},
			expectedVolumeFlags: []string{
				"--volume", "/Users/foo/repos:/workspace",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Manually construct Volume structs to test the Docker argument formatting.
			// This simulates what would be produced after AggregateConfigs/computeVolumes.
			var volumes []runtime.Volume
			for name, entry := range tc.volumes {
				from, exists := tc.volumeContext[name]
				// Skip optional volumes without runtime configuration.
				if !entry.Required && !exists {
					continue
				}
				volumes = append(volumes, runtime.Volume{
					Name:        name,
					VolumeEntry: entry,
					From:        from,
				})
			}

			// Build actual volume arguments as the daemon would.
			var actualVolumeFlags []string
			for _, vol := range volumes {
				actualVolumeFlags = append(actualVolumeFlags, "--volume", vol.String())
			}

			// Verify the volume flags match expectations.
			// Note: We use ElementsMatch instead of Equal because Go map iteration order is not guaranteed.
			require.ElementsMatch(t, tc.expectedVolumeFlags, actualVolumeFlags,
				"Docker volume flags should match expected format")
		})
	}
}

// TestDaemon_PingAllServers_ContextCancellation_Deadlock verifies that pingAllServers
// returns immediately when context is cancelled, even if some pings are stuck in
// uninterruptible I/O operations.
func TestDaemon_PingAllServers_ContextCancellation_Deadlock(t *testing.T) {
	t.Parallel()

	customSink := &testLoggerSink{}
	logger := hclog.NewInterceptLogger(&hclog.LoggerOptions{
		Name:  "test-daemon",
		Level: hclog.Debug,
	})
	logger.RegisterSink(customSink)

	clientManager := NewClientManager()
	healthTracker := NewHealthTracker([]string{"fast-server", "stuck-server"})

	daemon := &Daemon{
		logger:                   logger,
		clientManager:            clientManager,
		healthTracker:            healthTracker,
		clientHealthCheckTimeout: 3 * time.Second,
	}

	// Add a fast-responding server.
	fastClient := newMockMCPClientWithBehavior(0, nil)
	clientManager.Add("fast-server", fastClient, []string{"tool1"})
	healthTracker.Add("fast-server")

	// Add a stuck server that simulates blocked Docker I/O (will block for 30 seconds).
	stuckClient := newMockMCPClientWithStuckPing(30 * time.Second)
	clientManager.Add("stuck-server", stuckClient, []string{"tool2"})
	healthTracker.Add("stuck-server")

	// Create a context that we'll cancel after the stuck ping starts.
	ctx, cancel := context.WithCancel(context.Background())

	// Call pingAllServers in a goroutine.
	done := make(chan error, 1)
	go func() {
		done <- daemon.pingAllServers(ctx, 3*time.Second)
	}()

	// Wait for the stuck ping to start.
	select {
	case <-stuckClient.pingStarted:
		// Ping has started and is now stuck.
	case <-time.After(1 * time.Second):
		t.Fatal("Stuck ping never started")
	}

	// Cancel the context while the stuck ping is in progress.
	cancel()

	// Verify pingAllServers returns immediately when context is cancelled.
	select {
	case err := <-done:
		require.ErrorIs(t, err, context.Canceled)

		// Verify the interruption warning was logged.
		found := false
		for _, log := range customSink.messages {
			if log.level == hclog.Warn && strings.Contains(log.message, "interrupted") {
				found = true
				break
			}
		}
		assert.True(t, found, "Expected interruption warning in logs")

	case <-time.After(2 * time.Second):
		t.Fatal("pingAllServers did not return within 2 seconds after context cancellation")
	}
}
