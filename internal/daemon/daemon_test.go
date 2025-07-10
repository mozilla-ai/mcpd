package daemon

import (
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
