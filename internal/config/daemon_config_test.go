package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/mozilla-ai/mcpd/v2/internal/context"
)

func TestDuration_UnmarshalText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected time.Duration
		wantErr  bool
	}{
		{
			name:     "seconds",
			input:    "30s",
			expected: 30 * time.Second,
			wantErr:  false,
		},
		{
			name:     "minutes",
			input:    "5m",
			expected: 5 * time.Minute,
			wantErr:  false,
		},
		{
			name:     "hours",
			input:    "2h",
			expected: 2 * time.Hour,
			wantErr:  false,
		},
		{
			name:     "milliseconds",
			input:    "500ms",
			expected: 500 * time.Millisecond,
			wantErr:  false,
		},
		{
			name:     "microseconds",
			input:    "100µs",
			expected: 100 * time.Microsecond,
			wantErr:  false,
		},
		{
			name:     "nanoseconds",
			input:    "1000ns",
			expected: 1000 * time.Nanosecond,
			wantErr:  false,
		},
		{
			name:     "combined units",
			input:    "1h30m45s",
			expected: 1*time.Hour + 30*time.Minute + 45*time.Second,
			wantErr:  false,
		},
		{
			name:     "zero duration",
			input:    "0s",
			expected: 0,
			wantErr:  false,
		},
		{
			name:    "invalid format",
			input:   "invalid",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "missing unit",
			input:   "30",
			wantErr: true,
		},
		{
			name:     "negative duration",
			input:    "-5s",
			expected: -5 * time.Second,
			wantErr:  false, // time.ParseDuration allows negative durations
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var d Duration
			err := d.UnmarshalText([]byte(tc.input))

			if tc.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, Duration(tc.expected), d)
			require.Equal(t, tc.expected, time.Duration(d))
		})
	}
}

func TestDuration_MarshalText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		duration Duration
		expected string
	}{
		{
			name:     "seconds",
			duration: Duration(30 * time.Second),
			expected: "30s",
		},
		{
			name:     "minutes",
			duration: Duration(5 * time.Minute),
			expected: "5m0s",
		},
		{
			name:     "hours",
			duration: Duration(2 * time.Hour),
			expected: "2h0m0s",
		},
		{
			name:     "milliseconds",
			duration: Duration(500 * time.Millisecond),
			expected: "500ms",
		},
		{
			name:     "microseconds",
			duration: Duration(100 * time.Microsecond),
			expected: "100µs",
		},
		{
			name:     "nanoseconds",
			duration: Duration(1000 * time.Nanosecond),
			expected: "1µs",
		},
		{
			name:     "combined units",
			duration: Duration(1*time.Hour + 30*time.Minute + 45*time.Second),
			expected: "1h30m45s",
		},
		{
			name:     "zero duration",
			duration: Duration(0),
			expected: "0s",
		},
		{
			name:     "negative duration",
			duration: Duration(-5 * time.Second),
			expected: "-5s",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result, err := tc.duration.MarshalText()
			require.NoError(t, err)
			require.Equal(t, tc.expected, string(result))
		})
	}
}

func TestDuration_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		duration *Duration
		expected string
	}{
		{
			name:     "nil duration",
			duration: nil,
			expected: "",
		},
		{
			name:     "exact hours",
			duration: testDurationPtr(t, 2*time.Hour),
			expected: "2h",
		},
		{
			name:     "exact minutes",
			duration: testDurationPtr(t, 30*time.Minute),
			expected: "30m",
		},
		{
			name:     "exact seconds",
			duration: testDurationPtr(t, 45*time.Second),
			expected: "45s",
		},
		{
			name:     "exact milliseconds",
			duration: testDurationPtr(t, 500*time.Millisecond),
			expected: "500ms",
		},
		{
			name:     "exact microseconds",
			duration: testDurationPtr(t, 100*time.Microsecond),
			expected: "100µs",
		},
		{
			name:     "exact nanoseconds - converts to microseconds",
			duration: testDurationPtr(t, 1000*time.Nanosecond),
			expected: "1µs", // 1000ns = 1µs, so it gets formatted as microseconds
		},
		{
			name:     "true nanoseconds - not divisible by larger units",
			duration: testDurationPtr(t, 1337*time.Nanosecond),
			expected: "1337ns", // This will format as nanoseconds
		},
		{
			name:     "non-exact duration - fractional seconds",
			duration: testDurationPtr(t, 1500*time.Millisecond),
			expected: "1500ms",
		},
		{
			name:     "non-exact duration - fractional minutes",
			duration: testDurationPtr(t, 90*time.Second),
			expected: "90s",
		},
		{
			name:     "zero duration",
			duration: testDurationPtr(t, 0),
			expected: "0h", // Zero is evenly divisible by hour, so it formats as hours
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := tc.duration.String()
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestCORSConfigSection_EnableOrDefault(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		cors           *CORSConfigSection
		defaultEnable  bool
		expectedResult bool
	}{
		{
			name:           "nil CORS section uses default true",
			cors:           nil,
			defaultEnable:  true,
			expectedResult: true,
		},
		{
			name:           "nil CORS section uses default false",
			cors:           nil,
			defaultEnable:  false,
			expectedResult: false,
		},
		{
			name:           "CORS section with nil Enable uses default true",
			cors:           &CORSConfigSection{Enable: nil},
			defaultEnable:  true,
			expectedResult: true,
		},
		{
			name:           "CORS section with nil Enable uses default false",
			cors:           &CORSConfigSection{Enable: nil},
			defaultEnable:  false,
			expectedResult: false,
		},
		{
			name:           "CORS section with Enable=true overrides default false",
			cors:           &CORSConfigSection{Enable: testBoolPtr(t, true)},
			defaultEnable:  false,
			expectedResult: true,
		},
		{
			name:           "CORS section with Enable=false overrides default true",
			cors:           &CORSConfigSection{Enable: testBoolPtr(t, false)},
			defaultEnable:  true,
			expectedResult: false,
		},
		{
			name:           "CORS section with Enable=true matches default true",
			cors:           &CORSConfigSection{Enable: testBoolPtr(t, true)},
			defaultEnable:  true,
			expectedResult: true,
		},
		{
			name:           "CORS section with Enable=false matches default false",
			cors:           &CORSConfigSection{Enable: testBoolPtr(t, false)},
			defaultEnable:  false,
			expectedResult: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := tc.cors.EnableOrDefault(tc.defaultEnable)
			require.Equal(t, tc.expectedResult, result)
		})
	}
}

func TestDaemonConfig_StructureValidation(t *testing.T) {
	t.Parallel()

	t.Run("empty config is valid", func(t *testing.T) {
		t.Parallel()

		config := &DaemonConfig{}
		require.NotNil(t, config)
		require.Nil(t, config.API)
		require.Nil(t, config.MCP)
	})

	t.Run("populated config maintains structure", func(t *testing.T) {
		t.Parallel()

		config := &DaemonConfig{
			API: &APIConfigSection{
				Addr: testStringPtr(t, "localhost:8080"),
				Timeout: &APITimeoutConfigSection{
					Shutdown: testDurationPtr(t, 30*time.Second),
				},
				CORS: &CORSConfigSection{
					Enable:        testBoolPtr(t, true),
					Origins:       []string{"localhost:3000"},
					Methods:       []string{"GET", "POST"},
					Headers:       []string{"Content-Type"},
					ExposeHeaders: []string{"X-Custom"},
					Credentials:   testBoolPtr(t, false),
					MaxAge:        testDurationPtr(t, 5*time.Minute),
				},
			},
			MCP: &MCPConfigSection{
				Timeout: &MCPTimeoutConfigSection{
					Shutdown: testDurationPtr(t, 20*time.Second),
					Init:     testDurationPtr(t, 30*time.Second),
					Health:   testDurationPtr(t, 5*time.Second),
				},
				Interval: &MCPIntervalConfigSection{
					Health: testDurationPtr(t, 10*time.Second),
				},
			},
		}

		// Verify structure integrity
		require.NotNil(t, config.API)
		require.NotNil(t, config.API.Addr)
		require.Equal(t, "localhost:8080", *config.API.Addr)

		require.NotNil(t, config.API.Timeout)
		require.NotNil(t, config.API.Timeout.Shutdown)
		require.Equal(t, 30*time.Second, time.Duration(*config.API.Timeout.Shutdown))

		require.NotNil(t, config.API.CORS)
		require.NotNil(t, config.API.CORS.Enable)
		require.True(t, *config.API.CORS.Enable)
		require.ElementsMatch(t, []string{"localhost:3000"}, config.API.CORS.Origins)

		require.NotNil(t, config.MCP)
		require.NotNil(t, config.MCP.Timeout)
		require.NotNil(t, config.MCP.Timeout.Health)
		require.Equal(t, 5*time.Second, time.Duration(*config.MCP.Timeout.Health))
	})
}

func TestCORSConfigSection_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("empty slices are valid", func(t *testing.T) {
		t.Parallel()

		cors := &CORSConfigSection{
			Enable:        testBoolPtr(t, true),
			Origins:       []string{},
			Methods:       []string{},
			Headers:       []string{},
			ExposeHeaders: []string{},
		}

		require.True(t, cors.EnableOrDefault(false))
		require.Empty(t, cors.Origins)
		require.Empty(t, cors.Methods)
		require.Empty(t, cors.Headers)
		require.Empty(t, cors.ExposeHeaders)
	})

	t.Run("nil slices are valid", func(t *testing.T) {
		t.Parallel()

		cors := &CORSConfigSection{
			Enable:        testBoolPtr(t, true),
			Origins:       nil,
			Methods:       nil,
			Headers:       nil,
			ExposeHeaders: nil,
		}

		require.True(t, cors.EnableOrDefault(false))
		require.Nil(t, cors.Origins)
		require.Nil(t, cors.Methods)
		require.Nil(t, cors.Headers)
		require.Nil(t, cors.ExposeHeaders)
	})

	t.Run("wildcard origin handling", func(t *testing.T) {
		t.Parallel()

		cors := &CORSConfigSection{
			Enable:      testBoolPtr(t, true),
			Origins:     []string{"*"},
			Credentials: testBoolPtr(t, true), // This would be invalid in practice
		}

		require.True(t, cors.EnableOrDefault(false))
		require.ElementsMatch(t, []string{"*"}, cors.Origins)
		require.NotNil(t, cors.Credentials)
		require.True(t, *cors.Credentials)
	})
}

// Test helper functions
func testDurationPtr(t *testing.T, d time.Duration) *Duration {
	t.Helper()
	configDur := Duration(d)
	return &configDur
}

func testStringPtr(t *testing.T, s string) *string {
	t.Helper()
	return &s
}

func testBoolPtr(t *testing.T, b bool) *bool {
	t.Helper()
	return &b
}

func TestDaemonConfig_Set(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		config         *DaemonConfig
		path           string
		value          string
		expectedResult context.UpsertResult
		expectedError  string
		validateFn     func(t *testing.T, config *DaemonConfig)
	}{
		{
			name:           "empty path returns error",
			config:         &DaemonConfig{},
			path:           "",
			value:          "test",
			expectedResult: context.Noop,
			expectedError:  "config path cannot be empty",
		},
		{
			name:           "whitespace path returns error",
			config:         &DaemonConfig{},
			path:           "   ",
			value:          "test",
			expectedResult: context.Noop,
			expectedError:  "config path cannot be empty",
		},
		{
			name:           "unknown section returns error",
			config:         &DaemonConfig{},
			path:           "unknown.key",
			value:          "test",
			expectedResult: context.Noop,
			expectedError:  "unknown daemon config section: unknown",
		},
		{
			name:           "api section routes correctly",
			config:         &DaemonConfig{},
			path:           "api.addr",
			value:          "0.0.0.0:8080",
			expectedResult: context.Created,
			validateFn: func(t *testing.T, config *DaemonConfig) {
				t.Helper()
				require.NotNil(t, config.API)
				require.NotNil(t, config.API.Addr)
				require.Equal(t, "0.0.0.0:8080", *config.API.Addr)
			},
		},
		{
			name:           "mcp section routes correctly",
			config:         &DaemonConfig{},
			path:           "mcp.timeout.shutdown",
			value:          "30s",
			expectedResult: context.Created,
			validateFn: func(t *testing.T, config *DaemonConfig) {
				t.Helper()
				require.NotNil(t, config.MCP)
				require.NotNil(t, config.MCP.Timeout)
				require.NotNil(t, config.MCP.Timeout.Shutdown)
				require.Equal(t, Duration(30*time.Second), *config.MCP.Timeout.Shutdown)
			},
		},
		{
			name:           "path case normalization",
			config:         &DaemonConfig{},
			path:           "API.ADDR",
			value:          "0.0.0.0:8080",
			expectedResult: context.Created,
			validateFn: func(t *testing.T, config *DaemonConfig) {
				t.Helper()
				require.NotNil(t, config.API)
				require.NotNil(t, config.API.Addr)
				require.Equal(t, "0.0.0.0:8080", *config.API.Addr)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result, err := tc.config.Set(tc.path, tc.value)

			if tc.expectedError != "" {
				require.Error(t, err)
				require.EqualError(t, err, tc.expectedError)
				require.Equal(t, tc.expectedResult, result)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectedResult, result)
				if tc.validateFn != nil {
					tc.validateFn(t, tc.config)
				}
			}
		})
	}
}

func TestAPIConfigSection_Set(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		config         *APIConfigSection
		path           string
		value          string
		expectedResult context.UpsertResult
		expectedError  string
		validateFn     func(t *testing.T, config *APIConfigSection)
	}{
		{
			name:           "empty path returns error",
			config:         &APIConfigSection{},
			path:           "",
			value:          "test",
			expectedResult: context.Noop,
			expectedError:  "API config path cannot be empty",
		},
		{
			name:           "set addr creates new value",
			config:         &APIConfigSection{},
			path:           "addr",
			value:          "0.0.0.0:9090",
			expectedResult: context.Created,
			validateFn: func(t *testing.T, config *APIConfigSection) {
				t.Helper()
				require.NotNil(t, config.Addr)
				require.Equal(t, "0.0.0.0:9090", *config.Addr)
			},
		},
		{
			name: "set addr updates existing value",
			config: &APIConfigSection{
				Addr: testStringPtr(t, "localhost:8080"),
			},
			path:           "addr",
			value:          "0.0.0.0:9090",
			expectedResult: context.Updated,
			validateFn: func(t *testing.T, config *APIConfigSection) {
				t.Helper()
				require.NotNil(t, config.Addr)
				require.Equal(t, "0.0.0.0:9090", *config.Addr)
			},
		},
		{
			name: "set addr to empty deletes value",
			config: &APIConfigSection{
				Addr: testStringPtr(t, "localhost:8080"),
			},
			path:           "addr",
			value:          "",
			expectedResult: context.Deleted,
			validateFn: func(t *testing.T, config *APIConfigSection) {
				t.Helper()
				require.Nil(t, config.Addr)
			},
		},
		{
			name:           "timeout subsection routes correctly",
			config:         &APIConfigSection{},
			path:           "timeout.shutdown",
			value:          "20s",
			expectedResult: context.Created,
			validateFn: func(t *testing.T, config *APIConfigSection) {
				t.Helper()
				require.NotNil(t, config.Timeout)
				require.NotNil(t, config.Timeout.Shutdown)
				require.Equal(t, Duration(20*time.Second), *config.Timeout.Shutdown)
			},
		},
		{
			name:           "cors subsection routes correctly",
			config:         &APIConfigSection{},
			path:           "cors.enable",
			value:          "true",
			expectedResult: context.Created,
			validateFn: func(t *testing.T, config *APIConfigSection) {
				t.Helper()
				require.NotNil(t, config.CORS)
				require.NotNil(t, config.CORS.Enable)
				require.True(t, *config.CORS.Enable)
			},
		},
		{
			name:           "unknown key returns error",
			config:         &APIConfigSection{},
			path:           "unknown",
			value:          "test",
			expectedResult: context.Noop,
			expectedError:  "unknown API config key: unknown",
		},
		{
			name:           "unknown subsection returns error",
			config:         &APIConfigSection{},
			path:           "unknown.key",
			value:          "test",
			expectedResult: context.Noop,
			expectedError:  "unknown API subsection: unknown",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result, err := tc.config.Set(tc.path, tc.value)

			if tc.expectedError != "" {
				require.Error(t, err)
				require.EqualError(t, err, tc.expectedError)
				require.Equal(t, tc.expectedResult, result)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectedResult, result)
				if tc.validateFn != nil {
					tc.validateFn(t, tc.config)
				}
			}
		})
	}
}

func TestCORSConfigSection_Set(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		config         *CORSConfigSection
		path           string
		value          string
		expectedResult context.UpsertResult
		expectedError  string
		validateFn     func(t *testing.T, config *CORSConfigSection)
	}{
		{
			name:           "empty path returns error",
			config:         &CORSConfigSection{},
			path:           "",
			value:          "test",
			expectedResult: context.Noop,
			expectedError:  "CORS config path cannot be empty",
		},
		{
			name:           "set enable to true",
			config:         &CORSConfigSection{},
			path:           "enable",
			value:          "true",
			expectedResult: context.Created,
			validateFn: func(t *testing.T, config *CORSConfigSection) {
				t.Helper()
				require.NotNil(t, config.Enable)
				require.True(t, *config.Enable)
			},
		},
		{
			name:           "set enable to false",
			config:         &CORSConfigSection{},
			path:           "enable",
			value:          "false",
			expectedResult: context.Created,
			validateFn: func(t *testing.T, config *CORSConfigSection) {
				t.Helper()
				require.NotNil(t, config.Enable)
				require.False(t, *config.Enable)
			},
		},
		{
			name: "update enable value",
			config: &CORSConfigSection{
				Enable: testBoolPtr(t, false),
			},
			path:           "enable",
			value:          "true",
			expectedResult: context.Updated,
			validateFn: func(t *testing.T, config *CORSConfigSection) {
				t.Helper()
				require.NotNil(t, config.Enable)
				require.True(t, *config.Enable)
			},
		},
		{
			name:           "set enable with various boolean values",
			config:         &CORSConfigSection{},
			path:           "enable",
			value:          "1",
			expectedResult: context.Created,
			validateFn: func(t *testing.T, config *CORSConfigSection) {
				t.Helper()
				require.NotNil(t, config.Enable)
				require.True(t, *config.Enable)
			},
		},
		{
			name:           "set enable invalid boolean returns error",
			config:         &CORSConfigSection{},
			path:           "enable",
			value:          "invalid",
			expectedResult: context.Noop,
			expectedError:  "config value invalid: 'enable' (value: 'invalid')",
		},
		{
			name:           "set allow_origins single value",
			config:         &CORSConfigSection{},
			path:           "allow_origins",
			value:          "localhost:3000",
			expectedResult: context.Created,
			validateFn: func(t *testing.T, config *CORSConfigSection) {
				t.Helper()
				require.Equal(t, []string{"localhost:3000"}, config.Origins)
			},
		},
		{
			name:           "set allow_origins multiple values",
			config:         &CORSConfigSection{},
			path:           "allow_origins",
			value:          "localhost:3000,app.example.com:443",
			expectedResult: context.Created,
			validateFn: func(t *testing.T, config *CORSConfigSection) {
				t.Helper()
				require.Equal(t, []string{"localhost:3000", "app.example.com:443"}, config.Origins)
			},
		},
		{
			name: "update allow_origins",
			config: &CORSConfigSection{
				Origins: []string{"old.example.com"},
			},
			path:           "allow_origins",
			value:          "new.example.com",
			expectedResult: context.Updated,
			validateFn: func(t *testing.T, config *CORSConfigSection) {
				t.Helper()
				require.Equal(t, []string{"new.example.com"}, config.Origins)
			},
		},
		{
			name: "clear allow_origins with empty value",
			config: &CORSConfigSection{
				Origins: []string{"example.com"},
			},
			path:           "allow_origins",
			value:          "",
			expectedResult: context.Deleted,
			validateFn: func(t *testing.T, config *CORSConfigSection) {
				t.Helper()
				require.Empty(t, config.Origins)
			},
		},
		{
			name:           "set max_age",
			config:         &CORSConfigSection{},
			path:           "max_age",
			value:          "5m",
			expectedResult: context.Created,
			validateFn: func(t *testing.T, config *CORSConfigSection) {
				t.Helper()
				require.NotNil(t, config.MaxAge)
				require.Equal(t, Duration(5*time.Minute), *config.MaxAge)
			},
		},
		{
			name:           "set max_age invalid duration returns error",
			config:         &CORSConfigSection{},
			path:           "max_age",
			value:          "invalid",
			expectedResult: context.Noop,
			expectedError:  "config value invalid: 'max_age' (value: 'invalid'): invalid duration format: time: invalid duration \"invalid\"",
		},
		{
			name:           "unknown key returns error",
			config:         &CORSConfigSection{},
			path:           "unknown",
			value:          "test",
			expectedResult: context.Noop,
			expectedError:  "unknown CORS config key: unknown",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result, err := tc.config.Set(tc.path, tc.value)

			if tc.expectedError != "" {
				require.Error(t, err)
				require.EqualError(t, err, tc.expectedError)
				require.Equal(t, tc.expectedResult, result)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectedResult, result)
				if tc.validateFn != nil {
					tc.validateFn(t, tc.config)
				}
			}
		})
	}
}

func TestMCPConfigSection_Set(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		config         *MCPConfigSection
		path           string
		value          string
		expectedResult context.UpsertResult
		expectedError  string
		validateFn     func(t *testing.T, config *MCPConfigSection)
	}{
		{
			name:           "empty path returns error",
			config:         &MCPConfigSection{},
			path:           "",
			value:          "test",
			expectedResult: context.Noop,
			expectedError:  "MCP config path cannot be empty",
		},
		{
			name:           "invalid path without subsection returns error",
			config:         &MCPConfigSection{},
			path:           "invalid",
			value:          "test",
			expectedResult: context.Noop,
			expectedError:  "invalid MCP path, expected subsection.key: invalid",
		},
		{
			name:           "timeout subsection routes correctly",
			config:         &MCPConfigSection{},
			path:           "timeout.shutdown",
			value:          "30s",
			expectedResult: context.Created,
			validateFn: func(t *testing.T, config *MCPConfigSection) {
				t.Helper()
				require.NotNil(t, config.Timeout)
				require.NotNil(t, config.Timeout.Shutdown)
				require.Equal(t, Duration(30*time.Second), *config.Timeout.Shutdown)
			},
		},
		{
			name:           "interval subsection routes correctly",
			config:         &MCPConfigSection{},
			path:           "interval.health",
			value:          "10s",
			expectedResult: context.Created,
			validateFn: func(t *testing.T, config *MCPConfigSection) {
				t.Helper()
				require.NotNil(t, config.Interval)
				require.NotNil(t, config.Interval.Health)
				require.Equal(t, Duration(10*time.Second), *config.Interval.Health)
			},
		},
		{
			name:           "unknown subsection returns error",
			config:         &MCPConfigSection{},
			path:           "unknown.key",
			value:          "test",
			expectedResult: context.Noop,
			expectedError:  "invalid MCP path, expected subsection.key: unknown.key",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result, err := tc.config.Set(tc.path, tc.value)

			if tc.expectedError != "" {
				require.Error(t, err)
				require.EqualError(t, err, tc.expectedError)
				require.Equal(t, tc.expectedResult, result)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectedResult, result)
				if tc.validateFn != nil {
					tc.validateFn(t, tc.config)
				}
			}
		})
	}
}

func TestParsingHelpers(t *testing.T) {
	t.Parallel()

	t.Run("parseBool", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			input    string
			expected bool
			hasError bool
		}{
			{"true", true, false},
			{"True", true, false},
			{"TRUE", true, false},
			{"t", true, false},
			{"T", true, false},
			{"1", true, false},
			{"false", false, false},
			{"False", false, false},
			{"FALSE", false, false},
			{"f", false, false},
			{"F", false, false},
			{"0", false, false},
			{"invalid", false, true},
			{"", false, true},
		}

		for _, tc := range testCases {
			t.Run(tc.input, func(t *testing.T) {
				result, err := parseBool(tc.input)
				if tc.hasError {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
					require.Equal(t, tc.expected, result)
				}
			})
		}
	})

	t.Run("parseDuration", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			input    string
			expected Duration
			hasError bool
		}{
			{"1s", Duration(time.Second), false},
			{"5m", Duration(5 * time.Minute), false},
			{"2h", Duration(2 * time.Hour), false},
			{"500ms", Duration(500 * time.Millisecond), false},
			{"invalid", Duration(0), true},
			{"", Duration(0), true},
		}

		for _, tc := range testCases {
			t.Run(tc.input, func(t *testing.T) {
				result, err := parseDuration(tc.input)
				if tc.hasError {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
					require.Equal(t, tc.expected, result)
				}
			})
		}
	})

	t.Run("parseStringArray", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			input    string
			expected []string
		}{
			{"", nil},
			{"single", []string{"single"}},
			{"one,two,three", []string{"one", "two", "three"}},
			{"one, two, three", []string{"one", "two", "three"}},
			{" spaced , values ", []string{"spaced", "values"}},
		}

		for _, tc := range testCases {
			t.Run(tc.input, func(t *testing.T) {
				result := parseStringArray(tc.input)
				require.Equal(t, tc.expected, result)
			})
		}
	})
}

func TestAPITimeoutConfigSection_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		config      *APITimeoutConfigSection
		expectError bool
		errorMsg    string
	}{
		{
			name:        "nil config is valid",
			config:      nil,
			expectError: false,
		},
		{
			name:        "empty config is valid",
			config:      &APITimeoutConfigSection{},
			expectError: false,
		},
		{
			name: "positive shutdown timeout is valid",
			config: &APITimeoutConfigSection{
				Shutdown: testDurationPtr(t, 30*time.Second),
			},
			expectError: false,
		},
		{
			name: "zero shutdown timeout is invalid",
			config: &APITimeoutConfigSection{
				Shutdown: testDurationPtr(t, 0),
			},
			expectError: true,
			errorMsg:    "API shutdown timeout must be positive",
		},
		{
			name: "negative shutdown timeout is invalid",
			config: &APITimeoutConfigSection{
				Shutdown: testDurationPtr(t, -5*time.Second),
			},
			expectError: true,
			errorMsg:    "API shutdown timeout must be positive",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var err error
			if tc.config != nil {
				err = tc.config.Validate()
			}

			if tc.expectError {
				require.Error(t, err)
				require.EqualError(t, err, tc.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestCORSConfigSection_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		config      *CORSConfigSection
		expectError bool
		errorMsg    string
	}{
		{
			name:        "nil config is valid",
			config:      nil,
			expectError: false,
		},
		{
			name:        "empty config is valid",
			config:      &CORSConfigSection{},
			expectError: false,
		},
		{
			name: "origin is URL not network address (http:// -> but contains port)",
			config: &CORSConfigSection{
				Enable:  testBoolPtr(t, false),
				Origins: []string{"http://localhost:3000", "https://example.com"},
				Methods: []string{"GET"},
			},
			expectError: true,
			errorMsg:    "invalid origin address: http://localhost:3000",
		},
		{
			name: "valid enabled CORS config",
			config: &CORSConfigSection{
				Enable:      testBoolPtr(t, true),
				Origins:     []string{"localhost:3000", "example.com:443"},
				Methods:     []string{"GET", "POST", "PUT"},
				Headers:     []string{"Content-Type", "Authorization"},
				Credentials: testBoolPtr(t, true),
				MaxAge:      testDurationPtr(t, 5*time.Minute),
			},
			expectError: false,
		},
		{
			name: "wildcard methods are valid",
			config: &CORSConfigSection{
				Enable:  testBoolPtr(t, true),
				Methods: []string{"*"},
			},
			expectError: false,
		},
		{
			name: "mixed valid and wildcard methods",
			config: &CORSConfigSection{
				Enable:  testBoolPtr(t, true),
				Methods: []string{"GET", "*", "POST"},
			},
			expectError: false,
		},
		{
			name: "empty origin is invalid",
			config: &CORSConfigSection{
				Enable:  testBoolPtr(t, true),
				Origins: []string{"localhost:3000", "", "example.com:443"},
			},
			expectError: true,
			errorMsg:    "CORS origin cannot be empty",
		},
		{
			name: "empty method is invalid",
			config: &CORSConfigSection{
				Enable:  testBoolPtr(t, true),
				Methods: []string{"GET", "", "POST"},
			},
			expectError: true,
			errorMsg:    "CORS method cannot be empty",
		},
		{
			name: "invalid method is rejected",
			config: &CORSConfigSection{
				Enable:  testBoolPtr(t, true),
				Methods: []string{"GET", "INVALID", "POST"},
			},
			expectError: true,
			errorMsg:    "CORS method INVALID is not a valid HTTP request method",
		},
		{
			name: "zero max age is invalid",
			config: &CORSConfigSection{
				Enable: testBoolPtr(t, true),
				MaxAge: testDurationPtr(t, 0),
			},
			expectError: true,
			errorMsg:    "CORS max age must be positive",
		},
		{
			name: "negative max age is invalid",
			config: &CORSConfigSection{
				Enable: testBoolPtr(t, true),
				MaxAge: testDurationPtr(t, -5*time.Minute),
			},
			expectError: true,
			errorMsg:    "CORS max age must be positive",
		},
		{
			name: "multiple validation errors are combined",
			config: &CORSConfigSection{
				Enable:  testBoolPtr(t, true),
				Origins: []string{""},
				Methods: []string{"INVALID"},
				MaxAge:  testDurationPtr(t, 0),
			},
			expectError: true,
			errorMsg:    "CORS origin cannot be empty\nCORS method INVALID is not a valid HTTP request method\nCORS max age must be positive",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var err error
			if tc.config != nil {
				err = tc.config.Validate()
			}

			if tc.expectError {
				require.Error(t, err)
				require.EqualError(t, err, tc.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestMCPIntervalConfigSection_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		config      *MCPIntervalConfigSection
		expectError bool
		errorMsg    string
	}{
		{
			name:        "nil config is valid",
			config:      nil,
			expectError: false,
		},
		{
			name:        "empty config is valid",
			config:      &MCPIntervalConfigSection{},
			expectError: false,
		},
		{
			name: "positive health interval is valid",
			config: &MCPIntervalConfigSection{
				Health: testDurationPtr(t, 10*time.Second),
			},
			expectError: false,
		},
		{
			name: "zero health interval is invalid",
			config: &MCPIntervalConfigSection{
				Health: testDurationPtr(t, 0),
			},
			expectError: true,
			errorMsg:    "MCP health interval must be positive",
		},
		{
			name: "negative health interval is invalid",
			config: &MCPIntervalConfigSection{
				Health: testDurationPtr(t, -5*time.Second),
			},
			expectError: true,
			errorMsg:    "MCP health interval must be positive",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var err error
			if tc.config != nil {
				err = tc.config.Validate()
			}

			if tc.expectError {
				require.Error(t, err)
				require.EqualError(t, err, tc.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestMCPTimeoutConfigSection_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		config      *MCPTimeoutConfigSection
		expectError bool
		errorMsg    string
	}{
		{
			name:        "nil config is valid",
			config:      nil,
			expectError: false,
		},
		{
			name:        "empty config is valid",
			config:      &MCPTimeoutConfigSection{},
			expectError: false,
		},
		{
			name: "all positive timeouts are valid",
			config: &MCPTimeoutConfigSection{
				Shutdown: testDurationPtr(t, 30*time.Second),
				Init:     testDurationPtr(t, 60*time.Second),
				Health:   testDurationPtr(t, 5*time.Second),
			},
			expectError: false,
		},
		{
			name: "zero shutdown timeout is invalid",
			config: &MCPTimeoutConfigSection{
				Shutdown: testDurationPtr(t, 0),
			},
			expectError: true,
			errorMsg:    "MCP shutdown timeout must be positive",
		},
		{
			name: "zero init timeout is invalid",
			config: &MCPTimeoutConfigSection{
				Init: testDurationPtr(t, 0),
			},
			expectError: true,
			errorMsg:    "MCP init timeout must be positive",
		},
		{
			name: "zero health timeout is invalid",
			config: &MCPTimeoutConfigSection{
				Health: testDurationPtr(t, 0),
			},
			expectError: true,
			errorMsg:    "MCP health timeout must be positive",
		},
		{
			name: "negative shutdown timeout is invalid",
			config: &MCPTimeoutConfigSection{
				Shutdown: testDurationPtr(t, -5*time.Second),
			},
			expectError: true,
			errorMsg:    "MCP shutdown timeout must be positive",
		},
		{
			name: "multiple invalid timeouts combine errors",
			config: &MCPTimeoutConfigSection{
				Shutdown: testDurationPtr(t, 0),
				Init:     testDurationPtr(t, -10*time.Second),
				Health:   testDurationPtr(t, 0),
			},
			expectError: true,
			errorMsg:    "MCP shutdown timeout must be positive\nMCP init timeout must be positive\nMCP health timeout must be positive",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var err error
			if tc.config != nil {
				err = tc.config.Validate()
			}

			if tc.expectError {
				require.Error(t, err)
				require.EqualError(t, err, tc.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestAPIConfigSection_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		config      *APIConfigSection
		expectError bool
		errorMsg    string
	}{
		{
			name:        "nil config is valid",
			config:      nil,
			expectError: false,
		},
		{
			name:        "empty config is valid",
			config:      &APIConfigSection{},
			expectError: false,
		},
		{
			name: "valid API config",
			config: &APIConfigSection{
				Addr: testStringPtr(t, "localhost:8080"),
				Timeout: &APITimeoutConfigSection{
					Shutdown: testDurationPtr(t, 30*time.Second),
				},
				CORS: &CORSConfigSection{
					Enable:  testBoolPtr(t, true),
					Origins: []string{"localhost:3000"},
					Methods: []string{"GET", "POST"},
				},
			},
			expectError: false,
		},
		{
			name: "valid address formats",
			config: &APIConfigSection{
				Addr: testStringPtr(t, "0.0.0.0:8080"),
			},
			expectError: false,
		},
		{
			name: "colon-only address is valid",
			config: &APIConfigSection{
				Addr: testStringPtr(t, ":"),
			},
			expectError: false,
		},
		{
			name: "IPv6 address format is valid",
			config: &APIConfigSection{
				Addr: testStringPtr(t, "[::1]:8080"),
			},
			expectError: false,
		},
		{
			name: "empty address is invalid",
			config: &APIConfigSection{
				Addr: testStringPtr(t, ""),
			},
			expectError: true,
			errorMsg:    "API address cannot be empty",
		},
		{
			name: "invalid address format is rejected",
			config: &APIConfigSection{
				Addr: testStringPtr(t, "invalid-address"),
			},
			expectError: true,
			errorMsg:    `API address "invalid-address" appears to be invalid (expected format: host:port)`,
		},
		{
			name: "timeout validation error propagates",
			config: &APIConfigSection{
				Addr: testStringPtr(t, "localhost:8080"),
				Timeout: &APITimeoutConfigSection{
					Shutdown: testDurationPtr(t, 0), // Invalid
				},
			},
			expectError: true,
			errorMsg:    "timeout configuration error: API shutdown timeout must be positive",
		},
		{
			name: "CORS validation error propagates",
			config: &APIConfigSection{
				Addr: testStringPtr(t, "localhost:8080"),
				CORS: &CORSConfigSection{
					Enable:  testBoolPtr(t, true),
					Methods: []string{"INVALID"}, // Invalid method
				},
			},
			expectError: true,
			errorMsg:    "CORS configuration error: CORS method INVALID is not a valid HTTP request method",
		},
		{
			name: "multiple subsection errors are combined",
			config: &APIConfigSection{
				Addr: testStringPtr(t, "invalid"),
				Timeout: &APITimeoutConfigSection{
					Shutdown: testDurationPtr(t, 0),
				},
				CORS: &CORSConfigSection{
					Enable:  testBoolPtr(t, true),
					Methods: []string{"INVALID"},
				},
			},
			expectError: true,
			errorMsg:    "API address \"invalid\" appears to be invalid (expected format: host:port)\ntimeout configuration error: API shutdown timeout must be positive\nCORS configuration error: CORS method INVALID is not a valid HTTP request method",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var err error
			if tc.config != nil {
				err = tc.config.Validate()
			}

			if tc.expectError {
				require.Error(t, err)
				require.EqualError(t, err, tc.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestMCPConfigSection_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		config      *MCPConfigSection
		expectError bool
		errorMsg    string
	}{
		{
			name:        "nil config is valid",
			config:      nil,
			expectError: false,
		},
		{
			name:        "empty config is valid",
			config:      &MCPConfigSection{},
			expectError: false,
		},
		{
			name: "valid MCP config",
			config: &MCPConfigSection{
				Timeout: &MCPTimeoutConfigSection{
					Shutdown: testDurationPtr(t, 30*time.Second),
					Init:     testDurationPtr(t, 60*time.Second),
					Health:   testDurationPtr(t, 5*time.Second),
				},
				Interval: &MCPIntervalConfigSection{
					Health: testDurationPtr(t, 10*time.Second),
				},
			},
			expectError: false,
		},
		{
			name: "timeout validation error propagates",
			config: &MCPConfigSection{
				Timeout: &MCPTimeoutConfigSection{
					Shutdown: testDurationPtr(t, 0), // Invalid
				},
			},
			expectError: true,
			errorMsg:    "timeout configuration error: MCP shutdown timeout must be positive",
		},
		{
			name: "interval validation error propagates",
			config: &MCPConfigSection{
				Interval: &MCPIntervalConfigSection{
					Health: testDurationPtr(t, 0), // Invalid
				},
			},
			expectError: true,
			errorMsg:    "interval configuration error: MCP health interval must be positive",
		},
		{
			name: "multiple subsection errors are combined",
			config: &MCPConfigSection{
				Timeout: &MCPTimeoutConfigSection{
					Shutdown: testDurationPtr(t, 0),
				},
				Interval: &MCPIntervalConfigSection{
					Health: testDurationPtr(t, -5*time.Second),
				},
			},
			expectError: true,
			errorMsg:    "timeout configuration error: MCP shutdown timeout must be positive\ninterval configuration error: MCP health interval must be positive",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var err error
			if tc.config != nil {
				err = tc.config.Validate()
			}

			if tc.expectError {
				require.Error(t, err)
				require.EqualError(t, err, tc.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestDaemonConfig_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		config      *DaemonConfig
		expectError bool
		errorMsg    string
	}{
		{
			name:        "nil config is invalid",
			config:      nil,
			expectError: true,
			errorMsg:    "no daemon configuration found",
		},
		{
			name:        "empty config is valid",
			config:      &DaemonConfig{},
			expectError: false,
		},
		{
			name: "valid daemon config",
			config: &DaemonConfig{
				API: &APIConfigSection{
					Addr: testStringPtr(t, "localhost:8080"),
					Timeout: &APITimeoutConfigSection{
						Shutdown: testDurationPtr(t, 30*time.Second),
					},
					CORS: &CORSConfigSection{
						Enable:  testBoolPtr(t, true),
						Origins: []string{"localhost:3000"},
					},
				},
				MCP: &MCPConfigSection{
					Timeout: &MCPTimeoutConfigSection{
						Shutdown: testDurationPtr(t, 30*time.Second),
					},
					Interval: &MCPIntervalConfigSection{
						Health: testDurationPtr(t, 10*time.Second),
					},
				},
			},
			expectError: false,
		},
		{
			name: "API validation error propagates",
			config: &DaemonConfig{
				API: &APIConfigSection{
					Addr: testStringPtr(t, "invalid"), // Invalid address
				},
			},
			expectError: true,
			errorMsg:    "API configuration error: API address \"invalid\" appears to be invalid (expected format: host:port)",
		},
		{
			name: "MCP validation error propagates",
			config: &DaemonConfig{
				MCP: &MCPConfigSection{
					Timeout: &MCPTimeoutConfigSection{
						Shutdown: testDurationPtr(t, 0), // Invalid timeout
					},
				},
			},
			expectError: true,
			errorMsg:    "MCP configuration error: timeout configuration error: MCP shutdown timeout must be positive",
		},
		{
			name: "multiple section errors are combined",
			config: &DaemonConfig{
				API: &APIConfigSection{
					Addr: testStringPtr(t, "invalid"),
				},
				MCP: &MCPConfigSection{
					Timeout: &MCPTimeoutConfigSection{
						Shutdown: testDurationPtr(t, 0),
					},
				},
			},
			expectError: true,
			errorMsg:    "API configuration error: API address \"invalid\" appears to be invalid (expected format: host:port)\nMCP configuration error: timeout configuration error: MCP shutdown timeout must be positive",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.config.Validate()

			if tc.expectError {
				require.Error(t, err)
				require.EqualError(t, err, tc.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestContainsValidAddress(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		addr     string
		expected bool
	}{
		// Valid IPv4 host:port formats.
		{name: "localhost with port", addr: "localhost:8080", expected: true},
		{name: "IP with port", addr: "192.168.1.1:8080", expected: true},
		{name: "hostname with port", addr: "example.com:443", expected: true},
		{name: "wildcard IP with port", addr: "0.0.0.0:8080", expected: true},
		{name: "port only", addr: ":8080", expected: true},
		{name: "hostname with hyphens", addr: "my-server:8080", expected: true},
		{name: "just colon", addr: ":", expected: true}, // Represents bind to all interfaces on default port

		// Valid IPv6 formats.
		{name: "IPv6 loopback", addr: "[::1]:8080", expected: true},
		{name: "IPv6 full", addr: "[2001:db8::1]:8080", expected: true},
		{name: "IPv6 wildcard", addr: "[::]:8080", expected: true},

		// SHOULD be invalid formats.
		{name: "IPv6 non-numeric port", addr: "[::1]:abc", expected: true},            // net.SplitHostPort allows this
		{name: "hostname with @", addr: "user@host:8080", expected: true},             // net.SplitHostPort allows this
		{name: "hostname with special chars", addr: "host#name:8080", expected: true}, // net.SplitHostPort allows this
		{name: "non-numeric port", addr: "localhost:abc", expected: true},             // net.SplitHostPort allows this

		// Invalid formats.
		{name: "no port", addr: "localhost", expected: false},
		{name: "no host", addr: "8080", expected: false},
		{name: "empty string", addr: "", expected: false},
		{name: "multiple colons no brackets", addr: "2001:db8::1:8080", expected: false},
		{name: "space in hostname", addr: "local host:8080", expected: false}, // Spaces not allowed
		{name: "empty port", addr: "localhost:", expected: false},
		{name: "IPv6 missing brackets", addr: "::1:8080", expected: false},
		{name: "IPv6 empty host", addr: "[:8080", expected: false},
		{name: "IPv6 missing port", addr: "[::1]:", expected: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := isValidAddr(tc.addr)
			require.Equal(t, tc.expected, result, "Address: %s", tc.addr)
		})
	}
}

func TestDaemonConfig_Get(t *testing.T) {
	t.Parallel()

	// Create a fully populated config for testing
	fullConfig := &DaemonConfig{
		API: &APIConfigSection{
			Addr: testStringPtr(t, "localhost:8080"),
			Timeout: &APITimeoutConfigSection{
				Shutdown: testDurationPtr(t, 30*time.Second),
			},
			CORS: &CORSConfigSection{
				Enable:        testBoolPtr(t, true),
				Origins:       []string{"localhost:3000", "example.com:443"},
				Methods:       []string{"GET", "POST", "PUT"},
				Headers:       []string{"Content-Type", "Authorization"},
				ExposeHeaders: []string{"X-Request-ID"},
				Credentials:   testBoolPtr(t, false),
				MaxAge:        testDurationPtr(t, 5*time.Minute),
			},
		},
		MCP: &MCPConfigSection{
			Timeout: &MCPTimeoutConfigSection{
				Shutdown: testDurationPtr(t, 20*time.Second),
				Init:     testDurationPtr(t, 60*time.Second),
				Health:   testDurationPtr(t, 5*time.Second),
			},
			Interval: &MCPIntervalConfigSection{
				Health: testDurationPtr(t, 10*time.Second),
			},
		},
	}

	tests := []struct {
		name           string
		config         *DaemonConfig
		keys           []string
		expectedResult any
		expectedError  string
	}{
		{
			name:   "get all config with no keys",
			config: fullConfig,
			keys:   []string{},
			expectedResult: map[string]any{
				"api": map[string]any{
					"addr": "localhost:8080",
					"timeout": map[string]any{
						"shutdown": Duration(30 * time.Second),
					},
					"cors": map[string]any{
						"enable":            true,
						"allow_origins":     []string{"localhost:3000", "example.com:443"},
						"allow_methods":     []string{"GET", "POST", "PUT"},
						"allow_headers":     []string{"Content-Type", "Authorization"},
						"expose_headers":    []string{"X-Request-ID"},
						"allow_credentials": false,
						"max_age":           Duration(5 * time.Minute),
					},
				},
				"mcp": map[string]any{
					"timeout": map[string]any{
						"shutdown": Duration(20 * time.Second),
						"init":     Duration(1 * time.Minute),
						"health":   Duration(5 * time.Second),
					},
					"interval": map[string]any{
						"health": Duration(10 * time.Second),
					},
				},
			},
		},
		{
			name:   "get api section",
			config: fullConfig,
			keys:   []string{"api"},
			expectedResult: map[string]any{
				"addr": "localhost:8080",
				"timeout": map[string]any{
					"shutdown": Duration(30 * time.Second),
				},
				"cors": map[string]any{
					"enable":            true,
					"allow_origins":     []string{"localhost:3000", "example.com:443"},
					"allow_methods":     []string{"GET", "POST", "PUT"},
					"allow_headers":     []string{"Content-Type", "Authorization"},
					"expose_headers":    []string{"X-Request-ID"},
					"allow_credentials": false,
					"max_age":           Duration(5 * time.Minute),
				},
			},
		},
		{
			name:           "get api addr",
			config:         fullConfig,
			keys:           []string{"api", "addr"},
			expectedResult: "localhost:8080",
		},
		{
			name:   "get api timeout section",
			config: fullConfig,
			keys:   []string{"api", "timeout"},
			expectedResult: map[string]any{
				"shutdown": Duration(30 * time.Second),
			},
		},
		{
			name:           "get api timeout shutdown",
			config:         fullConfig,
			keys:           []string{"api", "timeout", "shutdown"},
			expectedResult: Duration(30 * time.Second),
		},
		{
			name:   "get api cors section",
			config: fullConfig,
			keys:   []string{"api", "cors"},
			expectedResult: map[string]any{
				"enable":            true,
				"allow_origins":     []string{"localhost:3000", "example.com:443"},
				"allow_methods":     []string{"GET", "POST", "PUT"},
				"allow_headers":     []string{"Content-Type", "Authorization"},
				"expose_headers":    []string{"X-Request-ID"},
				"allow_credentials": false,
				"max_age":           Duration(5 * time.Minute),
			},
		},
		{
			name:           "get api cors enable",
			config:         fullConfig,
			keys:           []string{"api", "cors", "enable"},
			expectedResult: true,
		},
		{
			name:           "get api cors allow_origins",
			config:         fullConfig,
			keys:           []string{"api", "cors", "allow_origins"},
			expectedResult: []string{"localhost:3000", "example.com:443"},
		},
		{
			name:   "get mcp section",
			config: fullConfig,
			keys:   []string{"mcp"},
			expectedResult: map[string]any{
				"timeout": map[string]any{
					"shutdown": Duration(20 * time.Second),
					"init":     Duration(1 * time.Minute),
					"health":   Duration(5 * time.Second),
				},
				"interval": map[string]any{
					"health": Duration(10 * time.Second),
				},
			},
		},
		{
			name:           "get mcp timeout health",
			config:         fullConfig,
			keys:           []string{"mcp", "timeout", "health"},
			expectedResult: Duration(5 * time.Second),
		},
		{
			name:           "get mcp interval health",
			config:         fullConfig,
			keys:           []string{"mcp", "interval", "health"},
			expectedResult: Duration(10 * time.Second),
		},
		{
			name:           "empty config returns empty map",
			config:         &DaemonConfig{},
			keys:           []string{},
			expectedResult: map[string]any{},
		},
		{
			name:          "get from empty config with keys returns error",
			config:        &DaemonConfig{},
			keys:          []string{"api", "addr"},
			expectedError: "no API configuration found",
		},
		{
			name:          "unknown section returns error",
			config:        fullConfig,
			keys:          []string{"unknown"},
			expectedError: "unknown daemon config section: unknown",
		},
		{
			name:          "api with invalid key returns error",
			config:        fullConfig,
			keys:          []string{"api", "invalid"},
			expectedError: "unknown API config key: invalid",
		},
		{
			name:          "mcp with invalid key returns error",
			config:        fullConfig,
			keys:          []string{"mcp", "invalid"},
			expectedError: "unknown MCP config key: invalid",
		},
		{
			name:          "deeply nested invalid path returns error",
			config:        fullConfig,
			keys:          []string{"api", "cors", "invalid"},
			expectedError: "unknown CORS config key: invalid",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result, err := tc.config.Get(tc.keys...)

			if tc.expectedError != "" {
				require.Error(t, err)
				require.EqualError(t, err, tc.expectedError)
				require.EqualError(t, err, tc.expectedError)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectedResult, result)
			}
		})
	}
}

func TestAPIConfigSection_Get(t *testing.T) {
	t.Parallel()

	// Create a fully populated API config for testing
	fullConfig := &APIConfigSection{
		Addr: testStringPtr(t, "0.0.0.0:8080"),
		Timeout: &APITimeoutConfigSection{
			Shutdown: testDurationPtr(t, 45*time.Second),
		},
		CORS: &CORSConfigSection{
			Enable:      testBoolPtr(t, false),
			Origins:     []string{"app.example.com:443"},
			Methods:     []string{"GET", "POST"},
			Headers:     []string{"Authorization"},
			Credentials: testBoolPtr(t, true),
			MaxAge:      testDurationPtr(t, 3*time.Minute),
		},
	}

	tests := []struct {
		name           string
		config         *APIConfigSection
		keys           []string
		expectedResult any
		expectedError  string
	}{
		{
			name:   "get all config with no keys",
			config: fullConfig,
			keys:   []string{},
			expectedResult: map[string]any{
				"addr": "0.0.0.0:8080",
				"timeout": map[string]any{
					"shutdown": Duration(45 * time.Second),
				},
				"cors": map[string]any{
					"enable":            false,
					"allow_origins":     []string{"app.example.com:443"},
					"allow_methods":     []string{"GET", "POST"},
					"allow_headers":     []string{"Authorization"},
					"allow_credentials": true,
					"max_age":           Duration(3 * time.Minute),
				},
			},
		},
		{
			name:           "get addr",
			config:         fullConfig,
			keys:           []string{"addr"},
			expectedResult: "0.0.0.0:8080",
		},
		{
			name:   "get timeout section",
			config: fullConfig,
			keys:   []string{"timeout"},
			expectedResult: map[string]any{
				"shutdown": Duration(45 * time.Second),
			},
		},
		{
			name:           "get timeout shutdown",
			config:         fullConfig,
			keys:           []string{"timeout", "shutdown"},
			expectedResult: Duration(45 * time.Second),
		},
		{
			name:   "get cors section",
			config: fullConfig,
			keys:   []string{"cors"},
			expectedResult: map[string]any{
				"enable":            false,
				"allow_origins":     []string{"app.example.com:443"},
				"allow_methods":     []string{"GET", "POST"},
				"allow_headers":     []string{"Authorization"},
				"allow_credentials": true,
				"max_age":           Duration(3 * time.Minute),
			},
		},
		{
			name:           "get cors enable",
			config:         fullConfig,
			keys:           []string{"cors", "enable"},
			expectedResult: false,
		},
		{
			name:           "get cors allow_origins",
			config:         fullConfig,
			keys:           []string{"cors", "allow_origins"},
			expectedResult: []string{"app.example.com:443"},
		},
		{
			name:           "get from empty config returns empty map",
			config:         &APIConfigSection{},
			keys:           []string{},
			expectedResult: map[string]any{},
		},
		{
			name:          "get missing addr returns nil",
			config:        &APIConfigSection{},
			keys:          []string{"addr"},
			expectedError: "api.addr not set",
		},
		{
			name:          "get from missing timeout section returns nil",
			config:        &APIConfigSection{},
			keys:          []string{"timeout", "shutdown"},
			expectedError: "api.timeout not set",
		},
		{
			name:          "unknown key returns error",
			config:        fullConfig,
			keys:          []string{"invalid"},
			expectedError: "unknown API config key: invalid",
		},
		{
			name:          "unknown subsection returns error",
			config:        fullConfig,
			keys:          []string{"unknown", "key"},
			expectedError: "unknown API subsection: unknown",
		},
		{
			name:          "invalid cors key returns error",
			config:        fullConfig,
			keys:          []string{"cors", "invalid"},
			expectedError: "unknown CORS config key: invalid",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result, err := tc.config.Get(tc.keys...)

			if tc.expectedError != "" {
				require.Error(t, err)
				require.EqualError(t, err, tc.expectedError)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectedResult, result)
			}
		})
	}
}

func TestCORSConfigSection_Get(t *testing.T) {
	t.Parallel()

	// Create a fully populated CORS config for testing
	fullConfig := &CORSConfigSection{
		Enable:        testBoolPtr(t, true),
		Origins:       []string{"localhost:3000", "example.com:443"},
		Methods:       []string{"GET", "POST", "PUT", "DELETE"},
		Headers:       []string{"Content-Type", "Authorization", "X-Requested-With"},
		ExposeHeaders: []string{"X-Request-ID", "X-Response-Time"},
		Credentials:   testBoolPtr(t, false),
		MaxAge:        testDurationPtr(t, 10*time.Minute),
	}

	tests := []struct {
		name           string
		config         *CORSConfigSection
		keys           []string
		expectedResult any
		expectedError  string
	}{
		{
			name:   "get all config with no keys",
			config: fullConfig,
			keys:   []string{},
			expectedResult: map[string]any{
				"enable":            true,
				"allow_origins":     []string{"localhost:3000", "example.com:443"},
				"allow_methods":     []string{"GET", "POST", "PUT", "DELETE"},
				"allow_headers":     []string{"Content-Type", "Authorization", "X-Requested-With"},
				"expose_headers":    []string{"X-Request-ID", "X-Response-Time"},
				"allow_credentials": false,
				"max_age":           Duration(10 * time.Minute),
			},
		},
		{
			name:           "get enable",
			config:         fullConfig,
			keys:           []string{"enable"},
			expectedResult: true,
		},
		{
			name:           "get allow_origins",
			config:         fullConfig,
			keys:           []string{"allow_origins"},
			expectedResult: []string{"localhost:3000", "example.com:443"},
		},
		{
			name:           "get allow_methods",
			config:         fullConfig,
			keys:           []string{"allow_methods"},
			expectedResult: []string{"GET", "POST", "PUT", "DELETE"},
		},
		{
			name:           "get allow_headers",
			config:         fullConfig,
			keys:           []string{"allow_headers"},
			expectedResult: []string{"Content-Type", "Authorization", "X-Requested-With"},
		},
		{
			name:           "get expose_headers",
			config:         fullConfig,
			keys:           []string{"expose_headers"},
			expectedResult: []string{"X-Request-ID", "X-Response-Time"},
		},
		{
			name:           "get credentials",
			config:         fullConfig,
			keys:           []string{"allow_credentials"},
			expectedResult: false,
		},
		{
			name:           "get max_age",
			config:         fullConfig,
			keys:           []string{"max_age"},
			expectedResult: Duration(10 * time.Minute),
		},
		{
			name:           "get from empty config returns empty map",
			config:         &CORSConfigSection{},
			keys:           []string{},
			expectedResult: map[string]any{},
		},
		{
			name:          "get missing enable returns error",
			config:        &CORSConfigSection{},
			keys:          []string{"enable"},
			expectedError: "cors.enable not set",
		},
		{
			name:          "get missing origins returns error",
			config:        &CORSConfigSection{},
			keys:          []string{"allow_origins"},
			expectedError: "cors.allow_origins not set",
		},
		{
			name:          "unknown key returns error",
			config:        fullConfig,
			keys:          []string{"invalid"},
			expectedError: "unknown CORS config key: invalid",
		},
		{
			name:          "path with subsection returns error",
			config:        fullConfig,
			keys:          []string{"enable", "subsection"},
			expectedError: "CORS config key invalid: enable.subsection",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result, err := tc.config.Get(tc.keys...)

			if tc.expectedError != "" {
				require.Error(t, err)
				require.EqualError(t, err, tc.expectedError)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectedResult, result)
			}
		})
	}
}

func TestMCPConfigSection_Get(t *testing.T) {
	t.Parallel()

	// Create a fully populated MCP config for testing
	fullConfig := &MCPConfigSection{
		Timeout: &MCPTimeoutConfigSection{
			Shutdown: testDurationPtr(t, 25*time.Second),
			Init:     testDurationPtr(t, 90*time.Second),
			Health:   testDurationPtr(t, 8*time.Second),
		},
		Interval: &MCPIntervalConfigSection{
			Health: testDurationPtr(t, 15*time.Second),
		},
	}

	tests := []struct {
		name           string
		config         *MCPConfigSection
		keys           []string
		expectedResult any
		expectedError  string
	}{
		{
			name:   "get all config with no keys",
			config: fullConfig,
			keys:   []string{},
			expectedResult: map[string]any{
				"timeout": map[string]any{
					"shutdown": Duration(25 * time.Second),
					"init":     Duration(90 * time.Second),
					"health":   Duration(8 * time.Second),
				},
				"interval": map[string]any{
					"health": Duration(15 * time.Second),
				},
			},
		},
		{
			name:   "get timeout section",
			config: fullConfig,
			keys:   []string{"timeout"},
			expectedResult: map[string]any{
				"shutdown": Duration(25 * time.Second),
				"init":     Duration(90 * time.Second),
				"health":   Duration(8 * time.Second),
			},
		},
		{
			name:           "get timeout shutdown",
			config:         fullConfig,
			keys:           []string{"timeout", "shutdown"},
			expectedResult: Duration(25 * time.Second),
		},
		{
			name:           "get timeout init",
			config:         fullConfig,
			keys:           []string{"timeout", "init"},
			expectedResult: Duration(90 * time.Second),
		},
		{
			name:           "get timeout health",
			config:         fullConfig,
			keys:           []string{"timeout", "health"},
			expectedResult: Duration(8 * time.Second),
		},
		{
			name:   "get interval section",
			config: fullConfig,
			keys:   []string{"interval"},
			expectedResult: map[string]any{
				"health": Duration(15 * time.Second),
			},
		},
		{
			name:           "get interval health",
			config:         fullConfig,
			keys:           []string{"interval", "health"},
			expectedResult: Duration(15 * time.Second),
		},
		{
			name:           "get from empty config returns empty map",
			config:         &MCPConfigSection{},
			keys:           []string{},
			expectedResult: map[string]any{},
		},
		{
			name:          "get from missing timeout section returns error",
			config:        &MCPConfigSection{},
			keys:          []string{"timeout", "shutdown"},
			expectedError: "mcp.timeout not set",
		},
		{
			name:          "get from missing interval section returns error",
			config:        &MCPConfigSection{},
			keys:          []string{"interval", "health"},
			expectedError: "mcp.interval not set",
		},
		{
			name:          "unknown subsection returns error",
			config:        fullConfig,
			keys:          []string{"invalid"},
			expectedError: "unknown MCP config key: invalid",
		},
		{
			name:          "invalid timeout key returns error",
			config:        fullConfig,
			keys:          []string{"timeout", "invalid"},
			expectedError: "unknown MCP timeout config key: invalid",
		},
		{
			name:          "invalid interval key returns error",
			config:        fullConfig,
			keys:          []string{"interval", "invalid"},
			expectedError: "unknown MCP interval config key: invalid",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result, err := tc.config.Get(tc.keys...)

			if tc.expectedError != "" {
				require.Error(t, err)
				require.EqualError(t, err, tc.expectedError)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectedResult, result)
			}
		})
	}
}

func TestAPITimeoutConfigSection_Get(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		config         *APITimeoutConfigSection
		keys           []string
		expectedResult any
		expectedError  string
	}{
		{
			name: "get all config with shutdown timeout",
			config: &APITimeoutConfigSection{
				Shutdown: testDurationPtr(t, 20*time.Second),
			},
			keys: []string{},
			expectedResult: map[string]any{
				"shutdown": Duration(20 * time.Second),
			},
		},
		{
			name: "get shutdown timeout",
			config: &APITimeoutConfigSection{
				Shutdown: testDurationPtr(t, 35*time.Second),
			},
			keys:           []string{"shutdown"},
			expectedResult: Duration(35 * time.Second),
		},
		{
			name:           "get from empty config returns empty map",
			config:         &APITimeoutConfigSection{},
			keys:           []string{},
			expectedResult: map[string]any{},
		},
		{
			name:          "get missing shutdown returns error",
			config:        &APITimeoutConfigSection{},
			keys:          []string{"shutdown"},
			expectedError: "api.timeout.shutdown not set",
		},
		{
			name:          "unknown key returns error",
			config:        &APITimeoutConfigSection{},
			keys:          []string{"invalid"},
			expectedError: "API timeout config key invalid: invalid",
		},
		{
			name: "shutdown is not a subsection",
			config: &APITimeoutConfigSection{
				Shutdown: testDurationPtr(t, 20*time.Second),
			},
			keys:          []string{"shutdown", "sub"},
			expectedError: "shutdown is not a subsection",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result, err := tc.config.Get(tc.keys...)

			if tc.expectedError != "" {
				require.Error(t, err)
				require.EqualError(t, err, tc.expectedError)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectedResult, result)
			}
		})
	}
}

func TestMCPTimeoutConfigSection_Get(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		config         *MCPTimeoutConfigSection
		keys           []string
		expectedResult any
		expectedError  string
	}{
		{
			name: "get all config with all timeouts",
			config: &MCPTimeoutConfigSection{
				Shutdown: testDurationPtr(t, 30*time.Second),
				Init:     testDurationPtr(t, 120*time.Second),
				Health:   testDurationPtr(t, 10*time.Second),
			},
			keys: []string{},
			expectedResult: map[string]any{
				"shutdown": Duration(30 * time.Second),
				"init":     Duration(120 * time.Second),
				"health":   Duration(10 * time.Second),
			},
		},
		{
			name: "get shutdown timeout",
			config: &MCPTimeoutConfigSection{
				Shutdown: testDurationPtr(t, 40*time.Second),
			},
			keys:           []string{"shutdown"},
			expectedResult: Duration(40 * time.Second),
		},
		{
			name: "get init timeout",
			config: &MCPTimeoutConfigSection{
				Init: testDurationPtr(t, 150*time.Second),
			},
			keys:           []string{"init"},
			expectedResult: Duration(150 * time.Second),
		},
		{
			name: "get health timeout",
			config: &MCPTimeoutConfigSection{
				Health: testDurationPtr(t, 12*time.Second),
			},
			keys:           []string{"health"},
			expectedResult: Duration(12 * time.Second),
		},
		{
			name:           "get from empty config returns empty map",
			config:         &MCPTimeoutConfigSection{},
			keys:           []string{},
			expectedResult: map[string]any{},
		},
		{
			name:          "get missing shutdown returns error",
			config:        &MCPTimeoutConfigSection{},
			keys:          []string{"shutdown"},
			expectedError: "mcp.timeout.shutdown not set",
		},
		{
			name:          "unknown key returns error",
			config:        &MCPTimeoutConfigSection{},
			keys:          []string{"invalid"},
			expectedError: "unknown MCP timeout config key: invalid",
		},
		{
			name: "timeout key is not a subsection",
			config: &MCPTimeoutConfigSection{
				Health: testDurationPtr(t, 5*time.Second),
			},
			keys:          []string{"health", "sub"},
			expectedError: "MCP timeout config key invalid: health.sub",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result, err := tc.config.Get(tc.keys...)

			if tc.expectedError != "" {
				require.Error(t, err)
				require.EqualError(t, err, tc.expectedError)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectedResult, result)
			}
		})
	}
}

func TestMCPIntervalConfigSection_Get(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		config         *MCPIntervalConfigSection
		keys           []string
		expectedResult any
		expectedError  string
	}{
		{
			name: "get all config with health interval",
			config: &MCPIntervalConfigSection{
				Health: testDurationPtr(t, 25*time.Second),
			},
			keys: []string{},
			expectedResult: map[string]any{
				"health": Duration(25 * time.Second),
			},
		},
		{
			name: "get health interval",
			config: &MCPIntervalConfigSection{
				Health: testDurationPtr(t, 30*time.Second),
			},
			keys:           []string{"health"},
			expectedResult: Duration(30 * time.Second),
		},
		{
			name:           "get from empty config returns empty map",
			config:         &MCPIntervalConfigSection{},
			keys:           []string{},
			expectedResult: map[string]any{},
		},
		{
			name:          "get missing health returns nil",
			config:        &MCPIntervalConfigSection{},
			keys:          []string{"health"},
			expectedError: "mcp.interval.health not set",
		},
		{
			name:          "unknown key returns error",
			config:        &MCPIntervalConfigSection{},
			keys:          []string{"invalid"},
			expectedError: "unknown MCP interval config key: invalid",
		},
		{
			name: "health is not a subsection",
			config: &MCPIntervalConfigSection{
				Health: testDurationPtr(t, 15*time.Second),
			},
			keys:          []string{"health", "sub"},
			expectedError: "MCP interval config key invalid: health.sub",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result, err := tc.config.Get(tc.keys...)

			if tc.expectedError != "" {
				require.Error(t, err)
				require.EqualError(t, err, tc.expectedError)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectedResult, result)
			}
		})
	}
}

func TestDaemonConfig_AvailableKeys(t *testing.T) {
	t.Parallel()

	config := &DaemonConfig{}
	keys := config.AvailableKeys()

	// Should have all API and MCP keys
	expectedKeys := []string{
		"api.addr",
		"api.timeout.shutdown",
		"api.cors.enable",
		"api.cors.allow_origins",
		"api.cors.allow_methods",
		"api.cors.allow_headers",
		"api.cors.expose_headers",
		"api.cors.allow_credentials",
		"api.cors.max_age",
		"mcp.timeout.shutdown",
		"mcp.timeout.init",
		"mcp.timeout.health",
		"mcp.interval.health",
	}

	// Extract key paths for comparison
	var actualKeys []string
	for _, key := range keys {
		actualKeys = append(actualKeys, key.Path)
	}

	// Verify all expected keys are present
	for _, expected := range expectedKeys {
		require.Contains(t, actualKeys, expected, "Missing expected key: %s", expected)
	}

	// Verify schema structure
	for _, key := range keys {
		require.NotEmpty(t, key.Path, "Key path should not be empty")
		require.NotEmpty(t, key.Type, "Key type should not be empty")
		require.NotEmpty(t, key.Description, "Key description should not be empty")

		// Validate key types are correct
		switch key.Path {
		case "api.addr":
			require.Equal(t, "string", key.Type)
		case "api.timeout.shutdown":
			require.Equal(t, "duration", key.Type)
		case "api.cors.enable", "api.cors.allow_credentials":
			require.Equal(t, "bool", key.Type)
		case "api.cors.allow_origins", "api.cors.allow_methods", "api.cors.allow_headers", "api.cors.expose_headers":
			require.Equal(t, "[]string", key.Type)
		case "api.cors.max_age":
			require.Equal(t, "duration", key.Type)
		case "mcp.timeout.shutdown", "mcp.timeout.init", "mcp.timeout.health", "mcp.interval.health":
			require.Equal(t, "duration", key.Type)
		}
	}
}

func TestAPIConfigSection_AvailableKeys(t *testing.T) {
	t.Parallel()

	config := &APIConfigSection{}
	keys := config.AvailableKeys()

	expectedKeys := []string{
		"addr",
		"timeout.shutdown",
		"cors.enable",
		"cors.allow_origins",
		"cors.allow_methods",
		"cors.allow_headers",
		"cors.expose_headers",
		"cors.allow_credentials",
		"cors.max_age",
	}

	// Extract key paths for comparison
	var actualKeys []string
	for _, key := range keys {
		actualKeys = append(actualKeys, key.Path)
	}

	// Verify all expected keys are present
	for _, expected := range expectedKeys {
		require.Contains(t, actualKeys, expected, "Missing expected key: %s", expected)
	}

	// Verify each key has proper schema information
	for _, key := range keys {
		require.NotEmpty(t, key.Path, "Key path should not be empty")
		require.NotEmpty(t, key.Type, "Key type should not be empty")
		require.NotEmpty(t, key.Description, "Key description should not be empty")
	}
}

func TestCORSConfigSection_AvailableKeys(t *testing.T) {
	t.Parallel()

	config := &CORSConfigSection{}
	keys := config.AvailableKeys()

	expectedKeys := []string{
		"enable",
		"allow_origins",
		"allow_methods",
		"allow_headers",
		"expose_headers",
		"allow_credentials",
		"max_age",
	}

	// Extract key paths for comparison
	var actualKeys []string
	for _, key := range keys {
		actualKeys = append(actualKeys, key.Path)
	}

	// Verify all expected keys are present
	for _, expected := range expectedKeys {
		require.Contains(t, actualKeys, expected, "Missing expected key: %s", expected)
	}

	// Verify schema correctness
	for _, key := range keys {
		require.NotEmpty(t, key.Path, "Key path should not be empty")
		require.NotEmpty(t, key.Type, "Key type should not be empty")
		require.NotEmpty(t, key.Description, "Key description should not be empty")

		// Validate specific types
		switch key.Path {
		case "enable", "credentials":
			require.Equal(t, "bool", key.Type)
		case "allow_origins", "allow_methods", "allow_headers", "expose_headers":
			require.Equal(t, "[]string", key.Type)
		case "max_age":
			require.Equal(t, "duration", key.Type)
		}
	}
}

func TestMCPConfigSection_AvailableKeys(t *testing.T) {
	t.Parallel()

	config := &MCPConfigSection{}
	keys := config.AvailableKeys()

	expectedKeys := []string{
		"timeout.shutdown",
		"timeout.init",
		"timeout.health",
		"interval.health",
	}

	// Extract key paths for comparison
	var actualKeys []string
	for _, key := range keys {
		actualKeys = append(actualKeys, key.Path)
	}

	// Verify all expected keys are present
	for _, expected := range expectedKeys {
		require.Contains(t, actualKeys, expected, "Missing expected key: %s", expected)
	}

	// Verify schema correctness
	for _, key := range keys {
		require.NotEmpty(t, key.Path, "Key path should not be empty")
		require.NotEmpty(t, key.Type, "Key type should not be empty")
		require.NotEmpty(t, key.Description, "Key description should not be empty")
		require.Equal(t, "duration", key.Type, "All MCP keys should be duration type")
	}
}

func TestAPITimeoutConfigSection_AvailableKeys(t *testing.T) {
	t.Parallel()

	config := &APITimeoutConfigSection{}
	keys := config.AvailableKeys()

	expectedKeys := []string{
		"shutdown",
	}

	// Extract key paths for comparison
	var actualKeys []string
	for _, key := range keys {
		actualKeys = append(actualKeys, key.Path)
	}

	// Verify all expected keys are present
	for _, expected := range expectedKeys {
		require.Contains(t, actualKeys, expected, "Missing expected key: %s", expected)
	}

	// Verify schema correctness
	for _, key := range keys {
		require.NotEmpty(t, key.Path, "Key path should not be empty")
		require.NotEmpty(t, key.Type, "Key type should not be empty")
		require.NotEmpty(t, key.Description, "Key description should not be empty")
		require.Equal(t, "duration", key.Type, "All API timeout keys should be duration type")
	}
}

func TestMCPTimeoutConfigSection_AvailableKeys(t *testing.T) {
	t.Parallel()

	config := &MCPTimeoutConfigSection{}
	keys := config.AvailableKeys()

	expectedKeys := []string{
		"shutdown",
		"init",
		"health",
	}

	// Extract key paths for comparison
	var actualKeys []string
	for _, key := range keys {
		actualKeys = append(actualKeys, key.Path)
	}

	// Verify all expected keys are present
	for _, expected := range expectedKeys {
		require.Contains(t, actualKeys, expected, "Missing expected key: %s", expected)
	}

	// Verify schema correctness
	for _, key := range keys {
		require.NotEmpty(t, key.Path, "Key path should not be empty")
		require.NotEmpty(t, key.Type, "Key type should not be empty")
		require.NotEmpty(t, key.Description, "Key description should not be empty")
		require.Equal(t, "duration", key.Type, "All MCP timeout keys should be duration type")
	}
}

func TestMCPIntervalConfigSection_AvailableKeys(t *testing.T) {
	t.Parallel()

	config := &MCPIntervalConfigSection{}
	keys := config.AvailableKeys()

	expectedKeys := []string{
		"health",
	}

	// Extract key paths for comparison
	var actualKeys []string
	for _, key := range keys {
		actualKeys = append(actualKeys, key.Path)
	}

	// Verify all expected keys are present
	for _, expected := range expectedKeys {
		require.Contains(t, actualKeys, expected, "Missing expected key: %s", expected)
	}

	// Verify schema correctness
	for _, key := range keys {
		require.NotEmpty(t, key.Path, "Key path should not be empty")
		require.NotEmpty(t, key.Type, "Key type should not be empty")
		require.NotEmpty(t, key.Description, "Key description should not be empty")
		require.Equal(t, "duration", key.Type, "All MCP interval keys should be duration type")
	}
}
