package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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
			assert.Equal(t, Duration(tc.expected), d)
			assert.Equal(t, tc.expected, time.Duration(d))
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
			assert.Equal(t, tc.expected, string(result))
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
			assert.Equal(t, tc.expected, result)
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
			assert.Equal(t, tc.expectedResult, result)
		})
	}
}

func TestDaemonConfig_StructureValidation(t *testing.T) {
	t.Parallel()

	t.Run("empty config is valid", func(t *testing.T) {
		t.Parallel()

		config := &DaemonConfig{}
		assert.NotNil(t, config)
		assert.Nil(t, config.API)
		assert.Nil(t, config.MCP)
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
					Origins:       []string{"http://localhost:3000"},
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
		assert.Equal(t, "localhost:8080", *config.API.Addr)

		require.NotNil(t, config.API.Timeout)
		require.NotNil(t, config.API.Timeout.Shutdown)
		assert.Equal(t, 30*time.Second, time.Duration(*config.API.Timeout.Shutdown))

		require.NotNil(t, config.API.CORS)
		require.NotNil(t, config.API.CORS.Enable)
		assert.True(t, *config.API.CORS.Enable)
		assert.ElementsMatch(t, []string{"http://localhost:3000"}, config.API.CORS.Origins)

		require.NotNil(t, config.MCP)
		require.NotNil(t, config.MCP.Timeout)
		require.NotNil(t, config.MCP.Timeout.Health)
		assert.Equal(t, 5*time.Second, time.Duration(*config.MCP.Timeout.Health))
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

		assert.True(t, cors.EnableOrDefault(false))
		assert.Empty(t, cors.Origins)
		assert.Empty(t, cors.Methods)
		assert.Empty(t, cors.Headers)
		assert.Empty(t, cors.ExposeHeaders)
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

		assert.True(t, cors.EnableOrDefault(false))
		assert.Nil(t, cors.Origins)
		assert.Nil(t, cors.Methods)
		assert.Nil(t, cors.Headers)
		assert.Nil(t, cors.ExposeHeaders)
	})

	t.Run("wildcard origin handling", func(t *testing.T) {
		t.Parallel()

		cors := &CORSConfigSection{
			Enable:      testBoolPtr(t, true),
			Origins:     []string{"*"},
			Credentials: testBoolPtr(t, true), // This would be invalid in practice
		}

		assert.True(t, cors.EnableOrDefault(false))
		assert.ElementsMatch(t, []string{"*"}, cors.Origins)
		require.NotNil(t, cors.Credentials)
		assert.True(t, *cors.Credentials)
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
				require.Contains(t, err.Error(), tc.expectedError)
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
				require.Contains(t, err.Error(), tc.expectedError)
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
			expectedError:  "invalid boolean value for enable: invalid",
		},
		{
			name:           "set allow_origins single value",
			config:         &CORSConfigSection{},
			path:           "allow_origins",
			value:          "http://localhost:3000",
			expectedResult: context.Created,
			validateFn: func(t *testing.T, config *CORSConfigSection) {
				t.Helper()
				require.Equal(t, []string{"http://localhost:3000"}, config.Origins)
			},
		},
		{
			name:           "set allow_origins multiple values",
			config:         &CORSConfigSection{},
			path:           "allow_origins",
			value:          "http://localhost:3000,https://app.example.com",
			expectedResult: context.Created,
			validateFn: func(t *testing.T, config *CORSConfigSection) {
				t.Helper()
				require.Equal(t, []string{"http://localhost:3000", "https://app.example.com"}, config.Origins)
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
			expectedError:  "invalid duration value for max_age: invalid",
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
				require.Contains(t, err.Error(), tc.expectedError)
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
			expectedError:  "unknown MCP subsection: unknown",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result, err := tc.config.Set(tc.path, tc.value)

			if tc.expectedError != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expectedError)
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
			{"yes", true, false},
			{"y", true, false},
			{"false", false, false},
			{"False", false, false},
			{"FALSE", false, false},
			{"f", false, false},
			{"F", false, false},
			{"0", false, false},
			{"no", false, false},
			{"n", false, false},
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
			{"", []string{}},
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
