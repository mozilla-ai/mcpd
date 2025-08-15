package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
