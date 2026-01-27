package runtime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mozilla-ai/mcpd/internal/config"
	"github.com/mozilla-ai/mcpd/internal/context"
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
			server := &Server{
				ServerEntry: config.ServerEntry{Name: tc.serverName},
			}
			result := server.filterEnv(tc.input)

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
			server := &Server{
				ServerEntry: config.ServerEntry{Name: tc.serverName},
			}
			result := server.filterEnv(tc.input)

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
			server := &Server{
				ServerEntry: config.ServerEntry{Name: tc.serverName},
			}
			result := server.filterEnv(tc.input)

			switch {
			case len(tc.expected) == 0:
				require.Empty(t, result)
			default:
				require.Equal(t, tc.expected, result)
			}
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
		{
			name:       "all set",
			env:        map[string]string{"FOO": "bar"},
			reqEnvVars: []string{"FOO"},
			wantErr:    false,
		},
		{
			name:       "missing",
			env:        map[string]string{},
			reqEnvVars: []string{"FOO"},
			wantErr:    true,
		},
		{
			name:       "empty",
			env:        map[string]string{"FOO": ""},
			reqEnvVars: []string{"FOO"},
			wantErr:    true,
		},
		{
			name:       "extra env",
			env:        map[string]string{"FOO": "bar", "BAR": "baz"},
			reqEnvVars: []string{"FOO"},
			wantErr:    false,
		},
		{
			name:       "no required env",
			env:        map[string]string{"FOO": "bar"},
			reqEnvVars: []string{},
			wantErr:    false,
		},
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
		{
			name:       "single --key=val",
			args:       []string{"--foo=bar"},
			reqValArgs: []string{"--foo"},
			wantErr:    false,
		},
		{
			name:       "single --key val",
			args:       []string{"--foo", "bar"},
			reqValArgs: []string{"--foo"},
			wantErr:    false,
		},
		{
			name:       "missing value",
			args:       []string{"--foo"},
			reqValArgs: []string{"--foo"},
			wantErr:    true,
		},
		{
			name:       "missing arg",
			args:       []string{},
			reqValArgs: []string{"--foo"},
			wantErr:    true,
		},
		{
			name:       "multiple args present",
			args:       []string{"--foo=bar", "--baz=qux"},
			reqValArgs: []string{"--foo", "--baz"},
			wantErr:    false,
		},
		{
			name:       "value looks like flag",
			args:       []string{"--foo", "--bar"},
			reqValArgs: []string{"--foo"},
			wantErr:    true,
		},
		// TODO: Uncomment or remove after deciding whether to support this kind of validation
		// {
		//     name:       "value is short flag",
		//     args:       []string{"--foo", "-b"},
		//     reqValArgs: []string{"--foo"},
		//     wantErr:    true,
		// },
		{
			name:       "empty args & none required",
			args:       []string{},
			reqValArgs: []string{},
			wantErr:    false,
		},
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

func TestValidateRequiredPositionalArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		args       []string
		reqPosArgs []string
		wantErr    bool
	}{
		{
			name:       "all positional present",
			args:       []string{"file.txt", "output.json", "--flag"},
			reqPosArgs: []string{"input", "output"},
			wantErr:    false,
		},
		{
			name:       "missing one positional",
			args:       []string{"file.txt", "--flag"},
			reqPosArgs: []string{"input", "output"},
			wantErr:    true,
		},
		{
			name:       "missing all positional",
			args:       []string{"--flag", "--another"},
			reqPosArgs: []string{"input", "output"},
			wantErr:    true,
		},
		{
			name:       "extra positional ok",
			args:       []string{"file.txt", "output.json", "extra.txt", "--flag"},
			reqPosArgs: []string{"input", "output"},
			wantErr:    false,
		},
		{
			name:       "positional only no flags",
			args:       []string{"file.txt", "output.json"},
			reqPosArgs: []string{"input", "output"},
			wantErr:    false,
		},
		{
			name:       "no positional required",
			args:       []string{"--flag", "--another"},
			reqPosArgs: []string{},
			wantErr:    false,
		},
		{
			name:       "empty args when positional required",
			args:       []string{},
			reqPosArgs: []string{"input"},
			wantErr:    true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			s := &Server{
				ServerEntry: config.ServerEntry{
					RequiredPositionalArgs: tc.reqPosArgs,
				},
				ServerExecutionContext: context.ServerExecutionContext{
					Args: tc.args,
				},
			}
			err := s.validateRequiredPositionalArgs()
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
		{
			name:        "single flag present",
			args:        []string{"--flag"},
			reqBoolArgs: []string{"--flag"},
			wantErr:     false,
		},
		{
			name:        "flag missing",
			args:        []string{},
			reqBoolArgs: []string{"--flag"},
			wantErr:     true,
		},
		{
			name:        "multiple flags present",
			args:        []string{"--foo", "--bar"},
			reqBoolArgs: []string{"--foo", "--bar"},
			wantErr:     false,
		},
		{
			name:        "flag present among others",
			args:        []string{"--foo", "--bar"},
			reqBoolArgs: []string{"--bar"},
			wantErr:     false,
		},
		{
			name:        "flag as prefix no match",
			args:        []string{"--foobar"},
			reqBoolArgs: []string{"--foo"},
			wantErr:     true,
		},
		{
			name:        "empty args & none required",
			args:        []string{},
			reqBoolArgs: []string{},
			wantErr:     false,
		},
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
		reqPosArgs  []string
		reqValArgs  []string
		reqBoolArgs []string
		wantErr     bool
	}{
		{
			name:        "all valid",
			env:         map[string]string{"FOO": "bar"},
			args:        []string{"--foo=val", "--flag"},
			reqEnv:      []string{"FOO"},
			reqPosArgs:  []string{},
			reqValArgs:  []string{"--foo"},
			reqBoolArgs: []string{"--flag"},
			wantErr:     false,
		},
		{
			name:        "all valid with positional",
			env:         map[string]string{"FOO": "bar"},
			args:        []string{"input.txt", "output.txt", "--foo=val", "--flag"},
			reqEnv:      []string{"FOO"},
			reqPosArgs:  []string{"input", "output"},
			reqValArgs:  []string{"--foo"},
			reqBoolArgs: []string{"--flag"},
			wantErr:     false,
		},
		{
			name:        "missing env",
			env:         map[string]string{},
			args:        []string{"--foo=val", "--flag"},
			reqEnv:      []string{"FOO"},
			reqPosArgs:  []string{},
			reqValArgs:  []string{"--foo"},
			reqBoolArgs: []string{"--flag"},
			wantErr:     true,
		},
		{
			name:        "missing positional",
			env:         map[string]string{"FOO": "bar"},
			args:        []string{"input.txt", "--foo=val", "--flag"},
			reqEnv:      []string{"FOO"},
			reqPosArgs:  []string{"input", "output"},
			reqValArgs:  []string{"--foo"},
			reqBoolArgs: []string{"--flag"},
			wantErr:     true,
		},
		{
			name:        "missing val arg",
			env:         map[string]string{"FOO": "bar"},
			args:        []string{"--flag"},
			reqEnv:      []string{"FOO"},
			reqPosArgs:  []string{},
			reqValArgs:  []string{"--foo"},
			reqBoolArgs: []string{"--flag"},
			wantErr:     true,
		},
		{
			name:        "missing bool arg",
			env:         map[string]string{"FOO": "bar"},
			args:        []string{"--foo=val"},
			reqEnv:      []string{"FOO"},
			reqPosArgs:  []string{},
			reqValArgs:  []string{"--foo"},
			reqBoolArgs: []string{"--flag"},
			wantErr:     true,
		},
		{
			name:        "all missing",
			env:         map[string]string{},
			args:        []string{},
			reqEnv:      []string{"FOO"},
			reqPosArgs:  []string{},
			reqValArgs:  []string{"--foo"},
			reqBoolArgs: []string{"--flag"},
			wantErr:     true,
		},
		{
			name:        "empty requirements",
			env:         map[string]string{},
			args:        []string{},
			reqEnv:      nil,
			reqPosArgs:  nil,
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
					RequiredEnvVars:        tc.reqEnv,
					RequiredPositionalArgs: tc.reqPosArgs,
					RequiredValueArgs:      tc.reqValArgs,
					RequiredBoolArgs:       tc.reqBoolArgs,
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

func TestPartitionArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name               string
		args               []string
		expectedPositional []string
		expectedFlags      []string
	}{
		{
			name:               "empty args",
			args:               []string{},
			expectedPositional: []string{},
		},
		{
			name:               "only positional args",
			args:               []string{"file1", "file2", "file3"},
			expectedPositional: []string{"file1", "file2", "file3"},
		},
		{
			name:               "only flags",
			args:               []string{"--flag1", "--flag2=value", "--flag3", "value3"},
			expectedPositional: []string{},
			expectedFlags:      []string{"--flag1", "--flag2=value", "--flag3", "value3"},
		},
		{
			name:               "positional then flags",
			args:               []string{"pos1", "pos2", "--flag1", "--flag2=value"},
			expectedPositional: []string{"pos1", "pos2"},
			expectedFlags:      []string{"--flag1", "--flag2=value"},
		},
		{
			name:               "positional then flag with value",
			args:               []string{"pos1", "--flag", "value", "--another"},
			expectedPositional: []string{"pos1"},
			expectedFlags:      []string{"--flag", "value", "--another"},
		},
		{
			name:               "no positional after first flag",
			args:               []string{"pos1", "--flag", "value", "should-be-flag-value"},
			expectedPositional: []string{"pos1"},
			expectedFlags:      []string{"--flag", "value", "should-be-flag-value"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			positional, flags := partitionArgs(tc.args)

			if tc.expectedPositional == nil {
				require.Nil(t, positional)
			} else {
				require.Equal(t, tc.expectedPositional, positional)
			}

			if tc.expectedFlags == nil {
				require.Nil(t, flags)
			} else {
				require.Equal(t, tc.expectedFlags, flags)
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
			name: "positional arguments before flags",
			args: []string{"/path/to/file", "second-positional", "--host", "localhost", "--debug"},
			seen: map[string]struct{}{},
			expectedArgs: []string{
				"${MCPD__TEST_SERVER__ARG_1}",
				"${MCPD__TEST_SERVER__ARG_2}",
				"--host=${MCPD__TEST_SERVER__HOST}",
				"--debug",
			},
			expectedCalls: map[string]string{
				"MCPD__TEST_SERVER__ARG_1": "${MCPD__TEST_SERVER__ARG_1}",
				"MCPD__TEST_SERVER__ARG_2": "${MCPD__TEST_SERVER__ARG_2}",
				"MCPD__TEST_SERVER__HOST":  "${MCPD__TEST_SERVER__HOST}",
			},
		},
		{
			name: "only positional arguments",
			args: []string{"/path/to/file", "second-arg", "third-arg"},
			seen: map[string]struct{}{},
			expectedArgs: []string{
				"${MCPD__TEST_SERVER__ARG_1}",
				"${MCPD__TEST_SERVER__ARG_2}",
				"${MCPD__TEST_SERVER__ARG_3}",
			},
			expectedCalls: map[string]string{
				"MCPD__TEST_SERVER__ARG_1": "${MCPD__TEST_SERVER__ARG_1}",
				"MCPD__TEST_SERVER__ARG_2": "${MCPD__TEST_SERVER__ARG_2}",
				"MCPD__TEST_SERVER__ARG_3": "${MCPD__TEST_SERVER__ARG_3}",
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
		name                   string
		serverName             string
		requiredPositionalArgs []string
		requiredValueArgs      []string
		requiredBoolArgs       []string
		runtimeArgs            []string
		expectedArgs           []string
		expectedContractCalls  map[string]string
	}{
		{
			name:                   "with positional args",
			serverName:             "test",
			requiredPositionalArgs: []string{"input", "output"},
			requiredValueArgs:      []string{"--config"},
			requiredBoolArgs:       []string{"--verbose"},
			runtimeArgs:            []string{},
			expectedArgs: []string{
				"${MCPD__TEST__INPUT}",
				"${MCPD__TEST__OUTPUT}",
				"--verbose",
				"--config=${MCPD__TEST__CONFIG}",
			},
			expectedContractCalls: map[string]string{
				"MCPD__TEST__INPUT":  "${MCPD__TEST__INPUT}",
				"MCPD__TEST__OUTPUT": "${MCPD__TEST__OUTPUT}",
				"MCPD__TEST__CONFIG": "${MCPD__TEST__CONFIG}",
			},
		},
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
					Name:                   tc.serverName,
					RequiredPositionalArgs: tc.requiredPositionalArgs,
					RequiredValueArgs:      tc.requiredValueArgs,
					RequiredBoolArgs:       tc.requiredBoolArgs,
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

func TestServer_Equal(t *testing.T) {
	t.Parallel()

	baseServer := func() *Server {
		return &Server{
			ServerEntry: config.ServerEntry{
				Name:                   "test-server",
				Package:                "uvx::test-server@1.0.0",
				Tools:                  []string{"tool1", "tool2"},
				RequiredEnvVars:        []string{"API_KEY", "SECRET"},
				RequiredPositionalArgs: []string{"pos1", "pos2"},
				RequiredValueArgs:      []string{"--arg1", "--arg2"},
				RequiredBoolArgs:       []string{"--flag1", "--flag2"},
			},
			ServerExecutionContext: context.ServerExecutionContext{
				Name: "test-server",
				Args: []string{"--test=value"},
				Env:  map[string]string{"TEST": "value"},
			},
		}
	}

	testCases := []struct {
		name     string
		server1  *Server
		server2  *Server
		expected bool
	}{
		{
			name:     "identical servers",
			server1:  baseServer(),
			server2:  baseServer(),
			expected: true,
		},
		{
			name:     "nil comparison",
			server1:  baseServer(),
			server2:  nil,
			expected: false,
		},
		{
			name:    "different static config - tools",
			server1: baseServer(),
			server2: func() *Server {
				srv := baseServer()
				srv.Tools = []string{"tool1", "tool3"}
				return srv
			}(),
			expected: false,
		},
		{
			name:    "different static config - package",
			server1: baseServer(),
			server2: func() *Server {
				srv := baseServer()
				srv.Package = "uvx::test-server@2.0.0"
				return srv
			}(),
			expected: false,
		},
		{
			name:    "different execution context - args",
			server1: baseServer(),
			server2: func() *Server {
				srv := baseServer()
				srv.Args = []string{"--test=different"}
				return srv
			}(),
			expected: false,
		},
		{
			name:    "different execution context - env",
			server1: baseServer(),
			server2: func() *Server {
				srv := baseServer()
				srv.Env = map[string]string{"TEST": "different"}
				return srv
			}(),
			expected: false,
		},
		{
			name:    "different execution context - name",
			server1: baseServer(),
			server2: func() *Server {
				srv := baseServer()
				srv.ServerExecutionContext.Name = "different-name"
				return srv
			}(),
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := tc.server1.Equals(tc.server2)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestServer_EqualExceptTools(t *testing.T) {
	t.Parallel()

	baseServer := func() *Server {
		return &Server{
			ServerEntry: config.ServerEntry{
				Name:                   "test-server",
				Package:                "uvx::test-server@1.0.0",
				Tools:                  []string{"tool1", "tool2"},
				RequiredEnvVars:        []string{"API_KEY", "SECRET"},
				RequiredPositionalArgs: []string{"pos1", "pos2"},
				RequiredValueArgs:      []string{"--arg1", "--arg2"},
				RequiredBoolArgs:       []string{"--flag1", "--flag2"},
			},
			ServerExecutionContext: context.ServerExecutionContext{
				Name: "test-server",
				Args: []string{"--test=value"},
				Env:  map[string]string{"TEST": "value"},
			},
		}
	}

	testCases := []struct {
		name     string
		server1  *Server
		server2  *Server
		expected bool
	}{
		{
			name:     "identical servers",
			server1:  baseServer(),
			server2:  baseServer(),
			expected: false, // No change at all
		},
		{
			name:     "nil comparison",
			server1:  baseServer(),
			server2:  nil,
			expected: false,
		},
		{
			name:    "only tools changed",
			server1: baseServer(),
			server2: func() *Server {
				srv := baseServer()
				srv.Tools = []string{"tool1", "tool3"} // Different tools (tool3)
				return srv
			}(),
			expected: true,
		},
		{
			name:    "tools changed with different order but same content",
			server1: baseServer(),
			server2: func() *Server {
				srv := baseServer()
				srv.Tools = []string{"tool2", "tool1"} // Different order, same tools.
				return srv
			}(),
			expected: false, // Same tools, different order = no real change
		},
		{
			name:    "package changed",
			server1: baseServer(),
			server2: func() *Server {
				srv := baseServer()
				srv.Package = "uvx::test-server@2.0.0" // Different package version
				return srv
			}(),
			expected: false, // Package changed = not tools-only
		},
		{
			name:    "tools added",
			server1: baseServer(),
			server2: func() *Server {
				srv := baseServer()
				srv.Tools = []string{"tool1", "tool2", "tool3"} // Added tool3
				return srv
			}(),
			expected: true,
		},
		{
			name:    "execution context args changed",
			server1: baseServer(),
			server2: func() *Server {
				srv := baseServer()
				srv.Tools = []string{"tool1", "tool3"}  // Different tools
				srv.Args = []string{"--test=different"} // Different args
				return srv
			}(),
			expected: false, // Not only tools differ (args also differ)
		},
		{
			name:    "execution context env changed",
			server1: baseServer(),
			server2: func() *Server {
				srv := baseServer()
				srv.Tools = []string{"tool1", "tool3"}           // Different tools
				srv.Env = map[string]string{"TEST": "different"} // Different env
				return srv
			}(),
			expected: false, // Not only tools differ (env also differs)
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := tc.server1.EqualsExceptTools(tc.server2)
			require.Equal(t, tc.expected, result)
		})
	}
}

// TestServer_SafeEnv_CrossServerFiltering tests that environment variables
// referencing other servers are properly filtered out to maintain security isolation.
func TestServer_SafeEnv_CrossServerFiltering(t *testing.T) {
	testdataDir := filepath.Join("testdata", "cross_server_env_filtering")

	// Set up environment variables that will be referenced in the config
	t.Setenv("MCPD__TIME_SERVER__API_KEY", "time-secret-123")
	t.Setenv("MCPD__DATABASE_SERVER__DB_HOST", "db.example.com")
	t.Setenv("MCPD__DATABASE_SERVER__DB_PORT", "5432")
	t.Setenv("MCPD__AUTH_SERVER__AUTH_TOKEN", "auth-token-789")

	// Load configs
	configLoader := &config.DefaultLoader{}
	configModifier, err := configLoader.Load(filepath.Join(testdataDir, "config.toml"))
	require.NoError(t, err)

	contextLoader := &context.DefaultLoader{}
	contextModifier, err := contextLoader.Load(filepath.Join(testdataDir, "runtime.toml"))
	require.NoError(t, err)

	tests := []struct {
		name          string
		serverName    string
		shouldHave    []string
		shouldNotHave []string
	}{
		{
			name:          "time-server should only access its own variables",
			serverName:    "time-server",
			shouldHave:    []string{"API_KEY"},
			shouldNotHave: []string{"DB_HOST", "DB_PORT"},
		},
		{
			name:          "auth-server should only access its own variables",
			serverName:    "auth-server",
			shouldHave:    []string{"AUTH_TOKEN"},
			shouldNotHave: []string{"DATABASE_URL", "TIME_API_KEY"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Use AggregateConfigs to properly combine both configs
			servers, err := AggregateConfigs(configModifier, contextModifier)
			require.NoError(t, err)

			// Find the server
			var server *Server
			for i := range servers {
				if servers[i].Name() == tc.serverName {
					server = &servers[i]
					break
				}
			}
			require.NotNil(t, server, "server should exist")

			// Get environment after filtering
			envs := server.SafeEnv()
			envMap := make(map[string]string, len(envs))
			for _, env := range envs {
				if parts := strings.SplitN(env, "=", 2); len(parts) == 2 {
					envMap[parts[0]] = parts[1]
				}
			}

			// Check required vars are present
			for _, key := range tc.shouldHave {
				require.Contains(t, envMap, key, "Server should have "+key)
			}

			// Check cross-server vars are filtered
			for _, key := range tc.shouldNotHave {
				require.NotContains(t, envMap, key, "Server should NOT have access to "+key+" (cross-server reference)")
			}
		})
	}
}

// TestServer_SafeEnvIsolated tests that SafeEnvIsolated returns only server-specific
// environment variables without inheriting from os.Environ().
func TestServer_SafeEnvIsolated(t *testing.T) {
	// Set up some OS environment variables that should NOT appear in isolated env
	t.Setenv("PATH", "/usr/bin:/bin")
	t.Setenv("HOME", "/home/test")
	t.Setenv("SOME_GLOBAL_VAR", "global-value")

	// Set up server-specific environment variables
	t.Setenv("MCPD__TEST_SERVER__API_KEY", "test-api-key")
	t.Setenv("MCPD__TEST_SERVER__API_URL", "https://api.example.com")

	server := &Server{
		ServerEntry: config.ServerEntry{
			Name: "test-server",
		},
		ServerExecutionContext: context.ServerExecutionContext{
			Env: map[string]string{
				"API_KEY": "server-specific-key",
				"API_URL": "https://server.example.com",
				"CUSTOM":  "custom-value",
			},
			RawEnv: map[string]string{
				"API_KEY": "${MCPD__TEST_SERVER__API_KEY}",
				"API_URL": "${MCPD__TEST_SERVER__API_URL}",
				"CUSTOM":  "custom-value",
			},
		},
	}

	t.Run("SafeEnv includes OS environment", func(t *testing.T) {
		envs := server.SafeEnv()
		envMap := make(map[string]string)
		for _, env := range envs {
			if parts := strings.SplitN(env, "=", 2); len(parts) == 2 {
				envMap[parts[0]] = parts[1]
			}
		}

		// Should have OS environment variables
		require.Contains(t, envMap, "PATH", "SafeEnv should include PATH from OS")
		require.Contains(t, envMap, "HOME", "SafeEnv should include HOME from OS")
		require.Contains(t, envMap, "SOME_GLOBAL_VAR", "SafeEnv should include SOME_GLOBAL_VAR from OS")

		// Should also have server-specific variables
		require.Contains(t, envMap, "API_KEY", "SafeEnv should include server's API_KEY")
		require.Equal(t, "server-specific-key", envMap["API_KEY"], "Server env should override OS env")
		require.Contains(t, envMap, "API_URL", "SafeEnv should include server's API_URL")
		require.Contains(t, envMap, "CUSTOM", "SafeEnv should include server's CUSTOM")
	})

	t.Run("SafeEnvIsolated excludes OS environment", func(t *testing.T) {
		envs := server.SafeEnvIsolated()
		envMap := make(map[string]string)
		for _, env := range envs {
			if parts := strings.SplitN(env, "=", 2); len(parts) == 2 {
				envMap[parts[0]] = parts[1]
			}
		}

		// Should NOT have OS environment variables
		require.NotContains(t, envMap, "PATH", "SafeEnvIsolated should NOT include PATH from OS")
		require.NotContains(t, envMap, "HOME", "SafeEnvIsolated should NOT include HOME from OS")
		require.NotContains(t, envMap, "SOME_GLOBAL_VAR", "SafeEnvIsolated should NOT include SOME_GLOBAL_VAR from OS")

		// Should still have server-specific variables
		require.Contains(t, envMap, "API_KEY", "SafeEnvIsolated should include server's API_KEY")
		require.Equal(t, "server-specific-key", envMap["API_KEY"])
		require.Contains(t, envMap, "API_URL", "SafeEnvIsolated should include server's API_URL")
		require.Contains(t, envMap, "CUSTOM", "SafeEnvIsolated should include server's CUSTOM")
		require.Equal(t, "custom-value", envMap["CUSTOM"])

		// Should only have exactly the server's configured variables
		require.Len(t, envMap, 3, "SafeEnvIsolated should only have the 3 configured server variables")
	})

	t.Run("SafeEnvIsolated filters cross-server references", func(t *testing.T) {
		// Add a cross-server reference that should be filtered
		serverWithCrossRef := &Server{
			ServerEntry: config.ServerEntry{
				Name: "test-server",
			},
			ServerExecutionContext: context.ServerExecutionContext{
				Env: map[string]string{
					"MY_VAR":    "my-value",
					"OTHER_VAR": "other-server-ref", // This would be expanded, but we check RawEnv
				},
				RawEnv: map[string]string{
					"MY_VAR":    "my-value",
					"OTHER_VAR": "${MCPD__OTHER_SERVER__SECRET}", // Cross-server reference
				},
			},
		}

		envs := serverWithCrossRef.SafeEnvIsolated()
		envMap := make(map[string]string)
		for _, env := range envs {
			if parts := strings.SplitN(env, "=", 2); len(parts) == 2 {
				envMap[parts[0]] = parts[1]
			}
		}

		// Should have the safe variable
		require.Contains(t, envMap, "MY_VAR", "Should include safe variable")
		require.Equal(t, "my-value", envMap["MY_VAR"])

		// Should NOT have the cross-server reference
		require.NotContains(t, envMap, "OTHER_VAR", "Should filter out cross-server reference")
	})
}

// TestServer_SafeArgs_CrossServerFiltering tests that arguments containing cross-server
// references are properly filtered out.
func TestServer_SafeArgs_CrossServerFiltering(t *testing.T) {
	testdataDir := filepath.Join("testdata", "cross_server_env_filtering")

	// Set up environment variables
	t.Setenv("MCPD__TIME_SERVER__API_KEY", "time-server-api-key")
	t.Setenv("MCPD__DATABASE_SERVER__DB_PASSWORD", "super-secret-db-password")

	// Load configs
	configLoader := &config.DefaultLoader{}
	configModifier, err := configLoader.Load(filepath.Join(testdataDir, "config.toml"))
	require.NoError(t, err)

	contextLoader := &context.DefaultLoader{}
	contextModifier, err := contextLoader.Load(filepath.Join(testdataDir, "runtime.toml"))
	require.NoError(t, err)

	servers, err := AggregateConfigs(configModifier, contextModifier)
	require.NoError(t, err)

	// Find servers
	serverMap := make(map[string]*Server, len(servers))
	for i := range servers {
		serverMap[servers[i].Name()] = &servers[i]
	}

	timeServer := serverMap["time-server"]
	dbServer := serverMap["database-server"]
	require.NotNil(t, timeServer, "time-server should exist")
	require.NotNil(t, dbServer, "database-server should exist")

	// Test time-server filtering
	timeArgs := timeServer.SafeArgs()
	require.Contains(t, timeArgs, "--api-key=time-server-api-key", "time-server should have its own API key")
	require.NotContains(
		t,
		timeArgs,
		"--stolen-db-secret=super-secret-db-password",
		"time-server should NOT have database password",
	)

	// Test database-server filtering
	dbArgs := dbServer.SafeArgs()
	require.Contains(
		t,
		dbArgs,
		"--db-password=super-secret-db-password",
		"database-server should have its own password",
	)
	require.NotContains(
		t,
		dbArgs,
		"--stolen-api-key=time-server-api-key",
		"database-server should NOT have time-server API key",
	)
}

// TestServer_SafeVolumes_CrossServerFiltering tests that volume paths containing cross-server
// references are properly filtered out to maintain security isolation.
func TestServer_SafeVolumes_CrossServerFiltering(t *testing.T) {
	testdataDir := filepath.Join("testdata", "cross_server_volume_filtering")

	// Set up environment variables that will be referenced in the config.
	t.Setenv("MCPD__FILESYSTEM_SERVER__WORKSPACE", "/Users/foo/repos/mcpd")
	t.Setenv("MCPD__DATABASE_SERVER__DATA_DIR", "/var/lib/postgres/data")

	// Load configs.
	configLoader := &config.DefaultLoader{}
	configModifier, err := configLoader.Load(filepath.Join(testdataDir, "config.toml"))
	require.NoError(t, err)

	contextLoader := &context.DefaultLoader{}
	contextModifier, err := contextLoader.Load(filepath.Join(testdataDir, "runtime.toml"))
	require.NoError(t, err)

	// Use AggregateConfigs to properly combine both configs.
	servers, err := AggregateConfigs(configModifier, contextModifier)
	require.NoError(t, err)

	// Find servers.
	serverMap := make(map[string]*Server, len(servers))
	for i := range servers {
		serverMap[servers[i].Name()] = &servers[i]
	}

	filesystemServer := serverMap["filesystem-server"]
	databaseServer := serverMap["database-server"]
	require.NotNil(t, filesystemServer, "filesystem-server should exist")
	require.NotNil(t, databaseServer, "database-server should exist")

	// Test filesystem-server filtering.
	fsVolumes := filesystemServer.SafeVolumes()
	fsVolumeMap := make(map[string]Volume)
	for _, vol := range fsVolumes {
		fsVolumeMap[vol.Name] = vol
	}

	// filesystem-server should have its own workspace volume.
	workspace, ok := fsVolumeMap["workspace"]
	require.True(t, ok, "filesystem-server should have workspace volume")
	require.Equal(t, "/Users/foo/repos/mcpd", workspace.From)

	// filesystem-server should NOT have logs volume (contains cross-server reference).
	_, hasLogs := fsVolumeMap["logs"]
	require.False(t, hasLogs, "filesystem-server should NOT have logs volume (cross-server reference)")

	// Test database-server filtering.
	dbVolumes := databaseServer.SafeVolumes()
	dbVolumeMap := make(map[string]Volume)
	for _, vol := range dbVolumes {
		dbVolumeMap[vol.Name] = vol
	}

	// database-server should have its own data volume.
	data, ok := dbVolumeMap["data"]
	require.True(t, ok, "database-server should have data volume")
	require.Equal(t, "/var/lib/postgres/data", data.From)

	// database-server should NOT have backups volume (contains cross-server reference).
	_, hasBackups := dbVolumeMap["backups"]
	require.False(t, hasBackups, "database-server should NOT have backups volume (cross-server reference)")
}

func TestServer_Volumes_LoadFromTestdata(t *testing.T) {
	t.Parallel()

	testdataDir := filepath.Join("testdata", "docker_volumes")

	// Load configs.
	configLoader := &config.DefaultLoader{}
	configModifier, err := configLoader.Load(filepath.Join(testdataDir, "config.toml"))
	require.NoError(t, err)

	contextLoader := &context.DefaultLoader{}
	contextModifier, err := contextLoader.Load(filepath.Join(testdataDir, "runtime.toml"))
	require.NoError(t, err)

	servers, err := AggregateConfigs(configModifier, contextModifier)
	require.NoError(t, err)

	// Find the filesystem server.
	var filesystemServer *Server
	for i := range servers {
		if servers[i].Name() == "filesystem" {
			filesystemServer = &servers[i]
			break
		}
	}
	require.NotNil(t, filesystemServer, "filesystem server should exist")

	// Get volumes.
	volumes := filesystemServer.SafeVolumes()

	// Convert to map for easier lookup in tests.
	volumeMap := make(map[string]Volume)
	for _, vol := range volumes {
		volumeMap[vol.Name] = vol
	}

	// Check that required volumes are present.
	workspace, ok := volumeMap["workspace"]
	require.True(t, ok, "workspace volume should be present")
	require.Equal(t, "workspace", workspace.Name)
	require.Equal(t, "/workspace", workspace.Path)
	require.True(t, workspace.Required)
	require.Equal(t, "/Users/foo/repos/mcpd", workspace.From)

	kubeconfig, ok := volumeMap["kubeconfig"]
	require.True(t, ok, "kubeconfig volume should be present")
	require.Equal(t, "kubeconfig", kubeconfig.Name)
	require.Equal(t, "/home/nonroot/.kube/config", kubeconfig.Path)
	require.True(t, kubeconfig.Required)
	require.Equal(t, "~/.kube/config", kubeconfig.From)

	// Check that optional configured volumes are present.
	gdrive, ok := volumeMap["gdrive"]
	require.True(t, ok, "gdrive volume should be present")
	require.Equal(t, "mcp-gdrive", gdrive.From)
	require.False(t, gdrive.Required)

	// Check that optional unconfigured volumes are NOT present.
	_, ok = volumeMap["calendar"]
	require.False(t, ok, "calendar volume should NOT be present (optional and not configured)")
}

func TestVolume_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		volume   Volume
		expected string
	}{
		{
			name: "absolute host path",
			volume: Volume{
				Name: "workspace",
				VolumeEntry: config.VolumeEntry{
					Path:     "/workspace",
					Required: true,
				},
				From: "/Users/foo/repos/mcpd",
			},
			expected: "/Users/foo/repos/mcpd:/workspace",
		},
		{
			name: "named docker volume",
			volume: Volume{
				Name: "data",
				VolumeEntry: config.VolumeEntry{
					Path:     "/data",
					Required: true,
				},
				From: "mcp-data",
			},
			expected: "mcp-data:/data",
		},
		{
			name: "kubeconfig file mount",
			volume: Volume{
				Name: "kubeconfig",
				VolumeEntry: config.VolumeEntry{
					Path:     "/home/nonroot/.kube/config",
					Required: false,
				},
				From: "~/.kube/config",
			},
			expected: "~/.kube/config:/home/nonroot/.kube/config",
		},
		{
			name: "relative path",
			volume: Volume{
				Name: "relative",
				VolumeEntry: config.VolumeEntry{
					Path:     "/app/config",
					Required: false,
				},
				From: "./config",
			},
			expected: "./config:/app/config",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := tc.volume.String()
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestServer_Volumes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		serverVolumes   config.VolumesEntry
		contextVolumes  context.VolumeExecutionContext
		expectedVolumes map[string]Volume
		wantErr         bool
		errContains     string
	}{
		{
			name:            "no volumes configured",
			serverVolumes:   config.VolumesEntry{},
			contextVolumes:  context.VolumeExecutionContext{},
			expectedVolumes: map[string]Volume{},
			wantErr:         false,
		},
		{
			name: "required volume present",
			serverVolumes: config.VolumesEntry{
				"workspace": config.VolumeEntry{
					Path:     "/workspace",
					Required: true,
				},
			},
			contextVolumes: context.VolumeExecutionContext{
				"workspace": "/Users/foo/repos",
			},
			expectedVolumes: map[string]Volume{
				"workspace": {
					Name: "workspace",
					VolumeEntry: config.VolumeEntry{
						Path:     "/workspace",
						Required: true,
					},
					From: "/Users/foo/repos",
				},
			},
			wantErr: false,
		},
		{
			name: "required volume missing",
			serverVolumes: config.VolumesEntry{
				"kubeconfig": config.VolumeEntry{
					Path:     "/home/nonroot/.kube/config",
					Required: true,
				},
			},
			contextVolumes:  context.VolumeExecutionContext{},
			expectedVolumes: nil,
			wantErr:         true,
			errContains:     "required volume 'kubeconfig' not configured",
		},
		{
			name: "optional volume present",
			serverVolumes: config.VolumesEntry{
				"gdrive": config.VolumeEntry{
					Path:     "/gdrive-server",
					Required: false,
				},
			},
			contextVolumes: context.VolumeExecutionContext{
				"gdrive": "mcp-gdrive",
			},
			expectedVolumes: map[string]Volume{
				"gdrive": {
					Name: "gdrive",
					VolumeEntry: config.VolumeEntry{
						Path:     "/gdrive-server",
						Required: false,
					},
					From: "mcp-gdrive",
				},
			},
			wantErr: false,
		},
		{
			name: "optional volume not configured - skipped",
			serverVolumes: config.VolumesEntry{
				"calendar": config.VolumeEntry{
					Path:     "/calendar-server",
					Required: false,
				},
			},
			contextVolumes:  context.VolumeExecutionContext{},
			expectedVolumes: map[string]Volume{},
			wantErr:         false,
		},
		{
			name: "mix of required and optional volumes",
			serverVolumes: config.VolumesEntry{
				"workspace": config.VolumeEntry{
					Path:     "/workspace",
					Required: true,
				},
				"kubeconfig": config.VolumeEntry{
					Path:     "/home/nonroot/.kube/config",
					Required: true,
				},
				"gdrive": config.VolumeEntry{
					Path:     "/gdrive-server",
					Required: false,
				},
				"calendar": config.VolumeEntry{
					Path:     "/calendar-server",
					Required: false,
				},
			},
			contextVolumes: context.VolumeExecutionContext{
				"workspace":  "/Users/foo/repos",
				"kubeconfig": "~/.kube/config",
				"gdrive":     "mcp-gdrive",
			},
			expectedVolumes: map[string]Volume{
				"workspace": {
					Name: "workspace",
					VolumeEntry: config.VolumeEntry{
						Path:     "/workspace",
						Required: true,
					},
					From: "/Users/foo/repos",
				},
				"kubeconfig": {
					Name: "kubeconfig",
					VolumeEntry: config.VolumeEntry{
						Path:     "/home/nonroot/.kube/config",
						Required: true,
					},
					From: "~/.kube/config",
				},
				"gdrive": {
					Name: "gdrive",
					VolumeEntry: config.VolumeEntry{
						Path:     "/gdrive-server",
						Required: false,
					},
					From: "mcp-gdrive",
				},
			},
			wantErr: false,
		},
		{
			name: "multiple required volumes - one missing",
			serverVolumes: config.VolumesEntry{
				"workspace": config.VolumeEntry{
					Path:     "/workspace",
					Required: true,
				},
				"kubeconfig": config.VolumeEntry{
					Path:     "/home/nonroot/.kube/config",
					Required: true,
				},
			},
			contextVolumes: context.VolumeExecutionContext{
				"workspace": "/Users/foo/repos",
			},
			expectedVolumes: nil,
			wantErr:         true,
			errContains:     "required volume 'kubeconfig' not configured",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := &Server{
				ServerEntry: config.ServerEntry{
					Name:    "filesystem",
					Volumes: tc.serverVolumes,
				},
				ServerExecutionContext: context.ServerExecutionContext{
					Volumes: tc.contextVolumes,
				},
			}

			// Populate volumes field.
			server.computeVolumes()

			if tc.wantErr {
				// For error cases, validation should fail.
				err := server.Validate()
				require.Error(t, err)
				if tc.errContains != "" {
					require.Contains(t, err.Error(), tc.errContains)
				}
			} else {
				// For success cases, get volumes and compare.
				volumes := server.SafeVolumes()

				// Convert to map for comparison.
				volumeMap := make(map[string]Volume)
				for _, vol := range volumes {
					volumeMap[vol.Name] = vol
				}

				require.Equal(t, tc.expectedVolumes, volumeMap)
			}
		})
	}
}
