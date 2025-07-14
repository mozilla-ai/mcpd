package runtime

import (
	"testing"

	"github.com/stretchr/testify/require"
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
				"MCPD__DATABASE__HOST":     "localhost",
				"MCPD__DATABASE__ARG__LOG": "true",
				"RANDOM_1":                 "foo",
				"RANDOM_2":                 "bar",
			},
			input: []string{
				"MCPD__DATABASE__HOST=${MCPD__DATABASE__HOST}",
				"LOGGING_ENABLED=${MCPD__DATABASE__ARG__LOG}",
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
				"MCPD__DATABASE__HOST":     "localhost",
				"MCPD__DATABASE__ARG__LOG": "true",
				"RANDOM_1":                 "foo",
				"RANDOM_2":                 "bar",
			},
			input: []string{
				"--my-arg1=${MCPD__DATABASE__HOST}",
				"--my-arg2=${MCPD__DATABASE__ARG__LOG}",
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
