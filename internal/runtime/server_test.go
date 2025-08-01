package runtime

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mozilla-ai/mcpd/v2/internal/config"
	"github.com/mozilla-ai/mcpd/v2/internal/context"
)

func TestServer_filterEnv(t *testing.T) {
	tests := []struct {
		name       string
		serverName string
		input      []string
		expected   []string
	}{
		{
			name:       "empty input",
			serverName: "time",
			input:      []string{},
			expected:   []string{},
		},
		{
			name:       "no MCP variables",
			serverName: "time",
			input: []string{
				"PATH=/usr/bin",
				"HOME=/home/user",
				"SHELL=/bin/bash",
			},
			expected: []string{
				"HOME=/home/user",
				"PATH=/usr/bin",
				"SHELL=/bin/bash",
			},
		},
		{
			name:       "current server variables kept",
			serverName: "time",
			input: []string{
				"MCPD__TIME__CONFIG=value1",
				"MCPD__TIME__PORT=8080",
				"OTHER_VAR=value2",
			},
			expected: []string{
				"MCPD__TIME__CONFIG=value1",
				"MCPD__TIME__PORT=8080",
				"OTHER_VAR=value2",
			},
		},
		{
			name:       "other server variables filtered",
			serverName: "time",
			input: []string{
				"MCPD__TIME__CONFIG=value1",
				"MCPD__DATABASE__HOST=localhost",
				"MCPD__AUTH__TOKEN=secret",
				"REGULAR_VAR=value2",
			},
			expected: []string{
				"MCPD__TIME__CONFIG=value1",
				"REGULAR_VAR=value2",
			},
		},
		{
			name:       "server name with hyphens",
			serverName: "auth-service",
			input: []string{
				"MCPD__AUTH_SERVICE__PORT=9000",
				"MCPD__OTHER_SERVICE__HOST=localhost",
				"NORMAL_VAR=value",
			},
			expected: []string{
				"MCPD__AUTH_SERVICE__PORT=9000",
				"NORMAL_VAR=value",
			},
		},
		{
			name:       "value references current server",
			serverName: "time",
			input: []string{
				"APP_CONFIG=${MCPD__TIME__HOST}:${MCPD__TIME__PORT}",
				"OTHER_VAR=value",
			},
			expected: []string{
				"APP_CONFIG=${MCPD__TIME__HOST}:${MCPD__TIME__PORT}",
				"OTHER_VAR=value",
			},
		},
		{
			name:       "value references other server - curly braces",
			serverName: "time",
			input: []string{
				"APP_CONFIG=${MCPD__DATABASE__HOST}:5432",
				"OTHER_VAR=value",
			},
			expected: []string{
				"OTHER_VAR=value",
			},
		},
		{
			name:       "value references other server - parentheses",
			serverName: "time",
			input: []string{
				"APP_CONFIG=$(MCPD__DATABASE__HOST):5432",
				"OTHER_VAR=value",
			},
			expected: []string{
				"OTHER_VAR=value",
			},
		},
		{
			name:       "value references other server - no brackets",
			serverName: "time",
			input: []string{
				"APP_CONFIG=$MCPD__DATABASE__HOST_VALUE",
				"OTHER_VAR=value",
			},
			expected: []string{
				"OTHER_VAR=value",
			},
		},
		{
			name:       "case insensitive matching",
			serverName: "time",
			input: []string{
				"app_config=${mcpd__database__host}:5432",
				"OTHER_VAR=value",
			},
			expected: []string{
				"OTHER_VAR=value",
			},
		},
		{
			name:       "mixed case server names",
			serverName: "Time-Service",
			input: []string{
				"MCPD__TIME_SERVICE__PORT=8080",
				"MCPD__OTHER_SERVICE__HOST=localhost",
				"NORMAL_VAR=value",
			},
			expected: []string{
				"MCPD__TIME_SERVICE__PORT=8080",
				"NORMAL_VAR=value",
			},
		},
		{
			name:       "multiple references in same value",
			serverName: "time",
			input: []string{
				"COMPLEX_CONFIG=${MCPD__TIME__HOST}:${MCPD__DATABASE__PORT}",
				"OTHER_VAR=value",
			},
			expected: []string{
				"OTHER_VAR=value",
			},
		},
		{
			name:       "reference in middle of value",
			serverName: "time",
			input: []string{
				"URL=https://${MCPD__AUTH__HOST}/api/v1",
				"OTHER_VAR=value",
			},
			expected: []string{
				"OTHER_VAR=value",
			},
		},
		{
			name:       "similar but different prefixes",
			serverName: "time",
			input: []string{
				"MCPD__TIME__CONFIG=value1",
				"MCPD_TIME_CONFIG=value2",       // Different separator
				"MCPD__TIMEOUT__CONFIG=value3",  // Different server
				"NOT_MCPD__TIME__CONFIG=value4", // Different app prefix
			},
			expected: []string{
				"MCPD__TIME__CONFIG=value1",
				"NOT_MCPD__TIME__CONFIG=value4",
			},
		},
		{
			name:       "no value part",
			serverName: "time",
			input: []string{
				"MCPD__TIME__CONFIG=",
				"MCPD__DATABASE__HOST=",
				"EMPTY_VAR=",
			},
			expected: []string{
				"EMPTY_VAR=",
				"MCPD__TIME__CONFIG=",
			},
		},
		{
			name:       "complex server names with numbers",
			serverName: "service-v2",
			input: []string{
				"MCPD__SERVICE_V2__PORT=8080",
				"MCPD__SERVICE_V1__PORT=8081",
				"CONFIG=${MCPD__SERVICE_V1__HOST}",
			},
			expected: []string{
				"MCPD__SERVICE_V2__PORT=8080",
			},
		},
		{
			name:       "regex edge cases",
			serverName: "time",
			input: []string{
				"VAR1=${MCPD__TIME__CONFIG}extra",
				"VAR2=prefix${MCPD__DATABASE__HOST}suffix",
				"VAR3=$MCPD__TIME__VAR_WITH_UNDERSCORE",
				"VAR4=$MCPD__OTHER__VAR_WITH_UNDERSCORE",
			},
			expected: []string{
				"VAR1=${MCPD__TIME__CONFIG}extra",
				"VAR3=$MCPD__TIME__VAR_WITH_UNDERSCORE",
			},
		},
		{
			name:       "explicit tests for mcpd app level vars",
			serverName: "time",
			input: []string{
				"MCPD_API_KEY=123",
				"PLEASE_GIVE_ME_YOUR_DATA=$MCPD_API_KEY=123",
				"PLEASE_GIVE_ME_YOUR_DATA2=$(MCPD_API_KEY=123)",
				"PLEASE_GIVE_ME_YOUR_DATA3=${MCPD_API_KEY=123}",
				"PLEASE_GIVE_ME_YOUR_DATA3=very${MCPD_API_KEY=123}bad",
				"MCPD__TIME__I_AM_FINE=123",
				"VAR1=NOT_RELATED",
			},
			expected: []string{
				"MCPD__TIME__I_AM_FINE=123",
				"VAR1=NOT_RELATED",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := filterEnv(tc.input, tc.serverName)

			require.Equal(t, tc.expected, result)
		})
	}
}

func TestServer_filterEnv_EdgeCases(t *testing.T) {
	tests := []struct {
		name       string
		serverName string
		input      []string
		expected   []string
	}{
		{
			name:       "malformed environment variables are dropped",
			serverName: "time",
			input: []string{
				"MALFORMED_VAR_NO_EQUALS",
				"MCPD__TIME__CONFIG=value",
			},
			expected: []string{
				"MCPD__TIME__CONFIG=value",
			},
		},
		{
			name:       "multiple equals signs",
			serverName: "time",
			input: []string{
				"MCPD__TIME__CONFIG=value=with=equals",
				"MCPD__DATABASE__HOST=localhost=5432",
			},
			expected: []string{
				"MCPD__TIME__CONFIG=value=with=equals",
			},
		},
		{
			name:       "very long server name",
			serverName: "very-long-server-name-with-many-hyphens-and-words",
			input: []string{
				"MCPD__VERY_LONG_SERVER_NAME_WITH_MANY_HYPHENS_AND_WORDS__CONFIG=value",
				"MCPD__OTHER__CONFIG=value",
			},
			expected: []string{
				"MCPD__VERY_LONG_SERVER_NAME_WITH_MANY_HYPHENS_AND_WORDS__CONFIG=value",
			},
		},
		{
			name:       "empty server name",
			serverName: "",
			input: []string{
				"MCPD____CONFIG=value",
				"MCPD__OTHER__CONFIG=value",
				"NORMAL_VAR=value",
			},
			expected: []string{
				"MCPD____CONFIG=value",
				"NORMAL_VAR=value",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := filterEnv(tc.input, tc.serverName)

			require.Equal(t, tc.expected, result)
		})
	}
}

func TestServer_filterEnv_RegexPatterns(t *testing.T) {
	tests := []struct {
		name       string
		serverName string
		input      []string
		expected   []string
	}{
		{
			name:       "various bracket combinations",
			serverName: "time",
			input: []string{
				"VAR1=${MCPD__DATABASE__HOST}",
				"VAR2=$(MCPD__DATABASE__HOST)",
				"VAR3=$MCPD__DATABASE__HOST",
				"VAR4=${MCPD__TIME__HOST}",
				"VAR5=normal_value",
			},
			expected: []string{
				"VAR4=${MCPD__TIME__HOST}",
				"VAR5=normal_value",
			},
		},
		{
			name:       "incomplete bracket patterns",
			serverName: "time",
			input: []string{
				"VAR1=${MCPD__DATABASE__HOST",  // Missing closing brace
				"VAR2=$MCPD__DATABASE__HOST}",  // Missing opening brace
				"VAR3=$(MCPD__DATABASE__HOST",  // Missing closing paren
				"VAR4=$MCPD__DATABASE__HOST)",  // Missing opening paren
				"VAR4=$(MCPD__DATABASE__HOST}", // Different opening/closing char
				"VAR4=foo$MCPD__DATABASE__HOST)bar",
				"VAR4=foo$MCPD__DATABASE__HOST)bar",
				"VAR4=$MCPD__DATABASE__HOST)bar",
			},
			expected: []string{},
		},
		{
			name:       "special characters in values",
			serverName: "time",
			input: []string{
				"VAR1=${MCPD__DATABASE__HOST}/path/to/resource",
				"VAR2=http://${MCPD__AUTH__HOST}:8080/api",
				"VAR3=${MCPD__TIME__HOST}",
				"VAR4=value-with-special-chars!@#$%",
			},
			expected: []string{
				"VAR3=${MCPD__TIME__HOST}",
				"VAR4=value-with-special-chars!@#$%",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := filterEnv(tc.input, tc.serverName)

			switch {
			case len(tc.expected) == 0:
				require.Empty(t, result)
			default:
				require.Equal(t, tc.expected, result)
			}
		})
	}
}

func TestServer_ExpandEnvSlice(t *testing.T) {
	tests := []struct {
		name     string
		env      map[string]string
		input    []string
		expected []string
	}{
		{
			name: "basic",
			env: map[string]string{
				"MCPD__DATABASE__HOST": "foo",
			},
			input: []string{
				"MCPD__DATABASE__HOST=${MCPD__DATABASE__HOST}",
			},
			expected: []string{
				"MCPD__DATABASE__HOST=foo",
			},
		},
		{
			name: "no chained expansion",
			env: map[string]string{
				"MCPD__GITHUB__GITHUB_PERSONAL_ACCESS_TOKEN": "foo123",
			},
			input: []string{
				"GITHUB_TOKEN=${MCPD__GITHUB__GITHUB_PERSONAL_ACCESS_TOKEN}",
				"BAD_ACTOR_SEND_TOKEN_VALUE=${GITHUB_TOKEN}",
			},
			expected: []string{
				"GITHUB_TOKEN=foo123",
				"BAD_ACTOR_SEND_TOKEN_VALUE=", // Empty env var.
			},
		},
		{
			name: "malformed entry",
			env: map[string]string{
				"MCPD__GITHUB__GITHUB_PERSONAL_ACCESS_TOKEN": "foo123",
			},
			input: []string{
				"GITHUB_TOKEN=${MCPD__GITHUB__GITHUB_PERSONAL_ACCESS_TOKEN}",
				"BAD_ACTOR_SEND_TOKEN_VALUE",
			},
			expected: []string{
				"GITHUB_TOKEN=foo123",
				"BAD_ACTOR_SEND_TOKEN_VALUE", // Malformed in terms of an env var, but OK as an arg.
			},
		},
		{
			name: "full env",
			env: map[string]string{
				"MCPD__DATABASE__HOST": "localhost",
				"MCPD__DATABASE__LOG":  "true",
				"RANDOM_1":             "foo",
				"RANDOM_2":             "bar",
			},
			input: []string{
				"MCPD__DATABASE__HOST=${MCPD__DATABASE__HOST}",
				"LOGGING_ENABLED=${MCPD__DATABASE__LOG}",
				"RANDOM_2=${RANDOM_2}",
			},
			expected: []string{
				"MCPD__DATABASE__HOST=localhost",
				"LOGGING_ENABLED=true",
				"RANDOM_2=bar",
			},
		},
		{
			name: "full args",
			env: map[string]string{
				"MCPD__DATABASE__HOST": "localhost",
				"MCPD__DATABASE__LOG":  "true",
				"RANDOM_1":             "foo",
				"RANDOM_2":             "bar",
			},
			input: []string{
				"--my-arg1=${MCPD__DATABASE__HOST}",
				"--my-arg2=${MCPD__DATABASE__LOG}",
				"--my-arg3=${RANDOM_2}",
				"--my-flag",
			},
			expected: []string{
				"--my-arg1=localhost",
				"--my-arg2=true",
				"--my-arg3=bar",
				"--my-flag",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for k, v := range tc.env {
				t.Setenv(k, v)
			}

			result := expandEnvSlice(tc.input)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestValidateRequiredEnvVars(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		env        map[string]string
		reqEnvVars []string
		wantErr    bool
	}{
		{"all set", map[string]string{"FOO": "bar"}, []string{"FOO"}, false},
		{"missing", map[string]string{}, []string{"FOO"}, true},
		{"empty", map[string]string{"FOO": ""}, []string{"FOO"}, true},
		{"extra env", map[string]string{"FOO": "bar", "BAR": "baz"}, []string{"FOO"}, false},
		{"no required env", map[string]string{"FOO": "bar"}, []string{}, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			s := &Server{
				ServerEntry: config.ServerEntry{
					RequiredEnvVars:   tc.reqEnvVars,
					RequiredValueArgs: nil,
					RequiredBoolArgs:  nil,
				},
				ServerExecutionContext: context.ServerExecutionContext{
					Env:  tc.env,
					Args: nil,
				},
			}

			err := s.validateRequiredEnvVars()
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateRequiredValueArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		args       []string
		reqValArgs []string
		wantErr    bool
	}{
		{"single --key=val", []string{"--foo=bar"}, []string{"--foo"}, false},
		{"single --key val", []string{"--foo", "bar"}, []string{"--foo"}, false},
		{"missing value", []string{"--foo"}, []string{"--foo"}, true},
		{"missing arg", []string{}, []string{"--foo"}, true},
		{"multiple args present", []string{"--foo=bar", "--baz=qux"}, []string{"--foo", "--baz"}, false},
		{"value looks like flag", []string{"--foo", "--bar"}, []string{"--foo"}, true},
		// TODO: Uncomment or remove after deciding whether to support this kind of validation
		// {"value is short flag", []string{"--foo", "-b"}, []string{"--foo"}, true},
		{"empty args & none required", []string{}, []string{}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			s := &Server{
				ServerEntry: config.ServerEntry{
					RequiredEnvVars:   nil,
					RequiredValueArgs: tc.reqValArgs,
					RequiredBoolArgs:  nil,
				},
				ServerExecutionContext: context.ServerExecutionContext{
					Env:  nil,
					Args: tc.args,
				},
			}

			err := s.validateRequiredValueArgs()
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateRequiredBoolArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		args        []string
		reqBoolArgs []string
		wantErr     bool
	}{
		{"single flag present", []string{"--flag"}, []string{"--flag"}, false},
		{"flag missing", []string{}, []string{"--flag"}, true},
		{"multiple flags present", []string{"--foo", "--bar"}, []string{"--foo", "--bar"}, false},
		{"flag present among others", []string{"--foo", "--bar"}, []string{"--bar"}, false},
		{"flag as prefix no match", []string{"--foobar"}, []string{"--foo"}, true},
		{"empty args & none required", []string{}, []string{}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			s := &Server{
				ServerEntry: config.ServerEntry{
					RequiredEnvVars:   nil,
					RequiredValueArgs: nil,
					RequiredBoolArgs:  tc.reqBoolArgs,
				},
				ServerExecutionContext: context.ServerExecutionContext{
					Env:  nil,
					Args: tc.args,
				},
			}

			err := s.validateRequiredBoolArgs()
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		env         map[string]string
		args        []string
		reqEnv      []string
		reqValArgs  []string
		reqBoolArgs []string
		wantErr     bool
	}{
		{
			name:        "all valid",
			env:         map[string]string{"FOO": "bar"},
			args:        []string{"--foo=val", "--flag"},
			reqEnv:      []string{"FOO"},
			reqValArgs:  []string{"--foo"},
			reqBoolArgs: []string{"--flag"},
			wantErr:     false,
		},
		{
			name:        "missing env",
			env:         map[string]string{},
			args:        []string{"--foo=val", "--flag"},
			reqEnv:      []string{"FOO"},
			reqValArgs:  []string{"--foo"},
			reqBoolArgs: []string{"--flag"},
			wantErr:     true,
		},
		{
			name:        "missing val arg",
			env:         map[string]string{"FOO": "bar"},
			args:        []string{"--flag"},
			reqEnv:      []string{"FOO"},
			reqValArgs:  []string{"--foo"},
			reqBoolArgs: []string{"--flag"},
			wantErr:     true,
		},
		{
			name:        "missing bool arg",
			env:         map[string]string{"FOO": "bar"},
			args:        []string{"--foo=val"},
			reqEnv:      []string{"FOO"},
			reqValArgs:  []string{"--foo"},
			reqBoolArgs: []string{"--flag"},
			wantErr:     true,
		},
		{
			name:        "all missing",
			env:         map[string]string{},
			args:        []string{},
			reqEnv:      []string{"FOO"},
			reqValArgs:  []string{"--foo"},
			reqBoolArgs: []string{"--flag"},
			wantErr:     true,
		},
		{
			name:        "empty requirements",
			env:         map[string]string{},
			args:        []string{},
			reqEnv:      nil,
			reqValArgs:  nil,
			reqBoolArgs: nil,
			wantErr:     false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			s := &Server{
				ServerEntry: config.ServerEntry{
					RequiredEnvVars:   tc.reqEnv,
					RequiredValueArgs: tc.reqValArgs,
					RequiredBoolArgs:  tc.reqBoolArgs,
				},
				ServerExecutionContext: context.ServerExecutionContext{
					Env:  tc.env,
					Args: tc.args,
				},
			}

			err := s.Validate()
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestServer_exportRuntimeArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		args          []string
		seen          map[string]struct{}
		expectedArgs  []string
		expectedCalls map[string]string // env var name -> env var reference
	}{
		{
			name:          "empty args",
			args:          []string{},
			seen:          map[string]struct{}{},
			expectedArgs:  []string{},
			expectedCalls: map[string]string{},
		},
		{
			name:          "single bool flag",
			args:          []string{"--verbose"},
			seen:          map[string]struct{}{},
			expectedArgs:  []string{"--verbose"},
			expectedCalls: map[string]string{},
		},
		{
			name:         "single value arg with equals",
			args:         []string{"--host=localhost"},
			seen:         map[string]struct{}{},
			expectedArgs: []string{"--host=${MCPD__TEST_SERVER__HOST}"},
			expectedCalls: map[string]string{
				"MCPD__TEST_SERVER__HOST": "${MCPD__TEST_SERVER__HOST}",
			},
		},
		{
			name:         "single value arg separate",
			args:         []string{"--host", "localhost"},
			seen:         map[string]struct{}{},
			expectedArgs: []string{"--host=${MCPD__TEST_SERVER__HOST}"},
			expectedCalls: map[string]string{
				"MCPD__TEST_SERVER__HOST": "${MCPD__TEST_SERVER__HOST}",
			},
		},
		{
			name: "mixed args",
			args: []string{"--host=localhost", "--port", "8080", "--debug"},
			seen: map[string]struct{}{},
			expectedArgs: []string{
				"--host=${MCPD__TEST_SERVER__HOST}",
				"--port=${MCPD__TEST_SERVER__PORT}",
				"--debug",
			},
			expectedCalls: map[string]string{
				"MCPD__TEST_SERVER__HOST": "${MCPD__TEST_SERVER__HOST}",
				"MCPD__TEST_SERVER__PORT": "${MCPD__TEST_SERVER__PORT}",
			},
		},
		{
			name: "skip already seen args",
			args: []string{"--host=localhost", "--port", "8080", "--debug"},
			seen: map[string]struct{}{
				"--host":  {},
				"--debug": {},
			},
			expectedArgs: []string{"--port=${MCPD__TEST_SERVER__PORT}"},
			expectedCalls: map[string]string{
				"MCPD__TEST_SERVER__PORT": "${MCPD__TEST_SERVER__PORT}",
			},
		},
		{
			name: "skip non-flag arguments",
			args: []string{"--host", "localhost", "not-a-flag", "--debug"},
			seen: map[string]struct{}{},
			expectedArgs: []string{
				"--host=${MCPD__TEST_SERVER__HOST}",
				"--debug",
			},
			expectedCalls: map[string]string{
				"MCPD__TEST_SERVER__HOST": "${MCPD__TEST_SERVER__HOST}",
			},
		},
		{
			name:          "flag followed by another flag",
			args:          []string{"--verbose", "--debug"},
			seen:          map[string]struct{}{},
			expectedArgs:  []string{"--verbose", "--debug"},
			expectedCalls: map[string]string{},
		},
		{
			name:         "complex server name normalization",
			args:         []string{"--api-key=secret"},
			seen:         map[string]struct{}{},
			expectedArgs: []string{"--api-key=${MCPD__TEST_SERVER__API_KEY}"},
			expectedCalls: map[string]string{
				"MCPD__TEST_SERVER__API_KEY": "${MCPD__TEST_SERVER__API_KEY}",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := &Server{
				ServerEntry: config.ServerEntry{
					Name: "test-server",
				},
				ServerExecutionContext: context.ServerExecutionContext{
					Args: tc.args,
				},
			}

			actualCalls := make(map[string]string)
			recordFunc := func(k, v string) {
				actualCalls[k] = v
			}

			result := server.exportRuntimeArgs("mcpd", tc.seen, recordFunc)

			if len(tc.expectedArgs) == 0 {
				require.Empty(t, result)
			} else {
				require.Equal(t, tc.expectedArgs, result)
			}
			require.Equal(t, tc.expectedCalls, actualCalls)
		})
	}
}

func TestServer_exportArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                  string
		serverName            string
		requiredValueArgs     []string
		requiredBoolArgs      []string
		runtimeArgs           []string
		expectedArgs          []string
		expectedContractCalls map[string]string
	}{
		{
			name:              "only required args",
			serverName:        "test",
			requiredValueArgs: []string{"--config", "--port"},
			requiredBoolArgs:  []string{"--verbose"},
			runtimeArgs:       []string{},
			expectedArgs: []string{
				"--verbose",
				"--config=${MCPD__TEST__CONFIG}",
				"--port=${MCPD__TEST__PORT}",
			},
			expectedContractCalls: map[string]string{
				"MCPD__TEST__CONFIG": "${MCPD__TEST__CONFIG}",
				"MCPD__TEST__PORT":   "${MCPD__TEST__PORT}",
			},
		},
		{
			name:              "only runtime args",
			serverName:        "test",
			requiredValueArgs: []string{},
			requiredBoolArgs:  []string{},
			runtimeArgs:       []string{"--host=localhost", "--debug"},
			expectedArgs: []string{
				"--host=${MCPD__TEST__HOST}",
				"--debug",
			},
			expectedContractCalls: map[string]string{
				"MCPD__TEST__HOST": "${MCPD__TEST__HOST}",
			},
		},
		{
			name:              "mixed required and runtime args",
			serverName:        "test",
			requiredValueArgs: []string{"--config"},
			requiredBoolArgs:  []string{"--verbose"},
			runtimeArgs:       []string{"--host=localhost", "--debug"},
			expectedArgs: []string{
				"--verbose",
				"--config=${MCPD__TEST__CONFIG}",
				"--host=${MCPD__TEST__HOST}",
				"--debug",
			},
			expectedContractCalls: map[string]string{
				"MCPD__TEST__CONFIG": "${MCPD__TEST__CONFIG}",
				"MCPD__TEST__HOST":   "${MCPD__TEST__HOST}",
			},
		},
		{
			name:              "runtime args duplicate required args",
			serverName:        "test",
			requiredValueArgs: []string{"--config"},
			requiredBoolArgs:  []string{"--verbose"},
			runtimeArgs:       []string{"--config=override", "--verbose", "--extra"},
			expectedArgs: []string{
				"--verbose",
				"--config=${MCPD__TEST__CONFIG}",
				"--extra",
			},
			expectedContractCalls: map[string]string{
				"MCPD__TEST__CONFIG": "${MCPD__TEST__CONFIG}",
			},
		},
		{
			name:              "server name with hyphens",
			serverName:        "github-server",
			requiredValueArgs: []string{"--token"},
			requiredBoolArgs:  []string{},
			runtimeArgs:       []string{"--repo-name=test"},
			expectedArgs: []string{
				"--token=${MCPD__GITHUB_SERVER__TOKEN}",
				"--repo-name=${MCPD__GITHUB_SERVER__REPO_NAME}",
			},
			expectedContractCalls: map[string]string{
				"MCPD__GITHUB_SERVER__TOKEN":     "${MCPD__GITHUB_SERVER__TOKEN}",
				"MCPD__GITHUB_SERVER__REPO_NAME": "${MCPD__GITHUB_SERVER__REPO_NAME}",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := &Server{
				ServerEntry: config.ServerEntry{
					Name:              tc.serverName,
					RequiredValueArgs: tc.requiredValueArgs,
					RequiredBoolArgs:  tc.requiredBoolArgs,
				},
				ServerExecutionContext: context.ServerExecutionContext{
					Args: tc.runtimeArgs,
				},
			}

			actualContractCalls := make(map[string]string)
			recordFunc := func(k, v string) {
				actualContractCalls[k] = v
			}

			result := server.exportArgs("mcpd", recordFunc)

			require.Equal(t, tc.expectedArgs, result)
			require.Equal(t, tc.expectedContractCalls, actualContractCalls)
		})
	}
}

func TestServer_exportEnvVars(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		serverName      string
		requiredEnvVars []string
		runtimeEnv      map[string]string
		expectedEnv     map[string]string
	}{
		{
			name:            "only required env vars",
			serverName:      "test",
			requiredEnvVars: []string{"API_KEY", "HOST"},
			runtimeEnv:      map[string]string{},
			expectedEnv: map[string]string{
				"API_KEY": "${MCPD__TEST__API_KEY}",
				"HOST":    "${MCPD__TEST__HOST}",
			},
		},
		{
			name:            "only runtime env vars",
			serverName:      "test",
			requiredEnvVars: []string{},
			runtimeEnv: map[string]string{
				"DEBUG": "true",
				"PORT":  "8080",
			},
			expectedEnv: map[string]string{
				"DEBUG": "${MCPD__TEST__DEBUG}",
				"PORT":  "${MCPD__TEST__PORT}",
			},
		},
		{
			name:            "mixed required and runtime env vars",
			serverName:      "test",
			requiredEnvVars: []string{"API_KEY"},
			runtimeEnv: map[string]string{
				"DEBUG": "true",
				"PORT":  "8080",
			},
			expectedEnv: map[string]string{
				"API_KEY": "${MCPD__TEST__API_KEY}",
				"DEBUG":   "${MCPD__TEST__DEBUG}",
				"PORT":    "${MCPD__TEST__PORT}",
			},
		},
		{
			name:            "runtime env overrides required env (should not duplicate)",
			serverName:      "test",
			requiredEnvVars: []string{"API_KEY"},
			runtimeEnv: map[string]string{
				"API_KEY": "override",
				"DEBUG":   "true",
			},
			expectedEnv: map[string]string{
				"API_KEY": "${MCPD__TEST__API_KEY}",
				"DEBUG":   "${MCPD__TEST__DEBUG}",
			},
		},
		{
			name:            "server name with hyphens",
			serverName:      "github-server",
			requiredEnvVars: []string{"GITHUB_TOKEN"},
			runtimeEnv: map[string]string{
				"DEBUG_MODE": "true",
			},
			expectedEnv: map[string]string{
				"GITHUB_TOKEN": "${MCPD__GITHUB_SERVER__GITHUB_TOKEN}",
				"DEBUG_MODE":   "${MCPD__GITHUB_SERVER__DEBUG_MODE}",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := &Server{
				ServerEntry: config.ServerEntry{
					Name:            tc.serverName,
					RequiredEnvVars: tc.requiredEnvVars,
				},
				ServerExecutionContext: context.ServerExecutionContext{
					Env: tc.runtimeEnv,
				},
			}

			result := server.exportEnvVars("mcpd")

			require.Equal(t, tc.expectedEnv, result)
		})
	}
}

func TestServers_Export(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		servers          Servers
		expectedErr      string
		expectedContract map[string]string
	}{
		{
			name:        "no servers",
			servers:     Servers{},
			expectedErr: "export error, no servers defined in runtime config",
		},
		{
			name: "single server with all types of configuration",
			servers: Servers{
				{
					ServerEntry: config.ServerEntry{
						Name:              "test-server",
						RequiredEnvVars:   []string{"API_KEY"},
						RequiredValueArgs: []string{"--config"},
						RequiredBoolArgs:  []string{"--verbose"},
					},
					ServerExecutionContext: context.ServerExecutionContext{
						Env: map[string]string{
							"DEBUG": "true",
						},
						Args: []string{"--host=localhost", "--flag"},
					},
				},
			},
			expectedContract: map[string]string{
				"MCPD__TEST_SERVER__API_KEY": "${MCPD__TEST_SERVER__API_KEY}", // From RequiredEnvVars
				"MCPD__TEST_SERVER__DEBUG":   "${MCPD__TEST_SERVER__DEBUG}",   // From runtime Env
				"MCPD__TEST_SERVER__CONFIG":  "${MCPD__TEST_SERVER__CONFIG}",  // From RequiredValueArgs
				"MCPD__TEST_SERVER__HOST":    "${MCPD__TEST_SERVER__HOST}",    // From runtime Args
			},
		},
		{
			name: "multiple servers",
			servers: Servers{
				{
					ServerEntry: config.ServerEntry{
						Name:              "server-a",
						RequiredValueArgs: []string{"--token"},
					},
					ServerExecutionContext: context.ServerExecutionContext{
						Args: []string{"--port=8080"},
					},
				},
				{
					ServerEntry: config.ServerEntry{
						Name:            "server-b",
						RequiredEnvVars: []string{"SECRET"},
					},
					ServerExecutionContext: context.ServerExecutionContext{
						Env: map[string]string{
							"DEBUG": "false",
						},
					},
				},
			},
			expectedContract: map[string]string{
				"MCPD__SERVER_A__TOKEN":  "${MCPD__SERVER_A__TOKEN}",  // From RequiredValueArgs
				"MCPD__SERVER_A__PORT":   "${MCPD__SERVER_A__PORT}",   // From runtime Args
				"MCPD__SERVER_B__SECRET": "${MCPD__SERVER_B__SECRET}", // From RequiredEnvVars
				"MCPD__SERVER_B__DEBUG":  "${MCPD__SERVER_B__DEBUG}",  // From runtime Env
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			path := filepath.Join(dir, "mcpd-export-test.toml")

			contract, err := tc.servers.Export(path)

			if tc.expectedErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expectedErr)
				require.Nil(t, contract)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectedContract, contract)

				// Verify config file was created
				fi, err := os.Stat(path)
				require.NoError(t, err)
				require.Greater(t, fi.Size(), int64(0))

				// Verify we can load the created config
				loader := context.DefaultLoader{}
				loadedCfg, err := loader.Load(path)
				require.NoError(t, err)
				require.NotNil(t, loadedCfg)
				require.Len(t, loadedCfg.List(), len(tc.servers))
			}
		})
	}
}

func TestEnvVarsToContract(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		envs     map[string]string
		expected map[string]string
	}{
		{
			name:     "empty input",
			envs:     map[string]string{},
			expected: map[string]string{},
		},
		{
			name: "single env var",
			envs: map[string]string{
				"API_KEY": "${MCPD__SERVER__API_KEY}",
			},
			expected: map[string]string{
				"MCPD__SERVER__API_KEY": "${MCPD__SERVER__API_KEY}",
			},
		},
		{
			name: "multiple env vars",
			envs: map[string]string{
				"API_KEY":    "${MCPD__SERVER__API_KEY}",
				"DEBUG_MODE": "${MCPD__SERVER__DEBUG_MODE}",
				"HOST":       "${MCPD__SERVER__HOST}",
			},
			expected: map[string]string{
				"MCPD__SERVER__API_KEY":    "${MCPD__SERVER__API_KEY}",
				"MCPD__SERVER__DEBUG_MODE": "${MCPD__SERVER__DEBUG_MODE}",
				"MCPD__SERVER__HOST":       "${MCPD__SERVER__HOST}",
			},
		},
		{
			name: "server name with hyphens (underscores)",
			envs: map[string]string{
				"GITHUB_TOKEN": "${MCPD__GITHUB_SERVER__GITHUB_TOKEN}",
				"API_BASE_URL": "${MCPD__GITHUB_SERVER__API_BASE_URL}",
			},
			expected: map[string]string{
				"MCPD__GITHUB_SERVER__GITHUB_TOKEN": "${MCPD__GITHUB_SERVER__GITHUB_TOKEN}",
				"MCPD__GITHUB_SERVER__API_BASE_URL": "${MCPD__GITHUB_SERVER__API_BASE_URL}",
			},
		},
		{
			name: "malformed placeholder references (no ${} wrapper)",
			envs: map[string]string{
				"API_KEY": "MCPD__SERVER__API_KEY", // Missing ${}
				"HOST":    "localhost",             // Not a placeholder at all
			},
			expected: map[string]string{}, // Should be filtered out
		},
		{
			name: "mixed valid and invalid placeholder references",
			envs: map[string]string{
				"API_KEY":    "${MCPD__SERVER__API_KEY}", // Valid
				"DEBUG_MODE": "MCPD__SERVER__DEBUG_MODE", // Invalid - missing ${}
				"HOST":       "${MCPD__SERVER__HOST}",    // Valid
				"PORT":       "8080",                     // Invalid - not a placeholder
			},
			expected: map[string]string{
				"MCPD__SERVER__API_KEY": "${MCPD__SERVER__API_KEY}",
				"MCPD__SERVER__HOST":    "${MCPD__SERVER__HOST}",
			},
		},
		{
			name: "edge case - empty placeholder",
			envs: map[string]string{
				"EMPTY": "${}",
			},
			expected: map[string]string{}, // Should be filtered out
		},
		{
			name: "edge case - whitespace-only placeholder",
			envs: map[string]string{
				"WHITESPACE": "${   }",
			},
			expected: map[string]string{}, // Should be filtered out
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := envVarsToContract(tc.envs)

			require.Equal(t, tc.expected, result)
		})
	}
}
