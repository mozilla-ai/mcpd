package runtime

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeForEnvVarName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		serverName string
		expected   string
	}{
		{
			name:       "simple server name",
			serverName: "test",
			expected:   "TEST",
		},
		{
			name:       "server name with hyphens",
			serverName: "my-server",
			expected:   "MY_SERVER",
		},
		{
			name:       "server name with multiple hyphens",
			serverName: "my-test-server",
			expected:   "MY_TEST_SERVER",
		},
		{
			name:       "mixed case with hyphens",
			serverName: "My-Test-Server",
			expected:   "MY_TEST_SERVER",
		},
		{
			name:       "already uppercase with underscores",
			serverName: "MY_SERVER",
			expected:   "MY_SERVER",
		},
		{
			name:       "empty string",
			serverName: "",
			expected:   "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := normalizeForEnvVarName(tc.serverName)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestExtractArgName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		rawArg   string
		expected string
	}{
		{
			name:     "simple flag",
			rawArg:   "--foo",
			expected: "foo",
		},
		{
			name:     "value argument",
			rawArg:   "--foo=bar",
			expected: "foo",
		},
		{
			name:     "hyphenated argument",
			rawArg:   "--my-arg",
			expected: "my-arg",
		},
		{
			name:     "hyphenated argument with value",
			rawArg:   "--my-arg=123",
			expected: "my-arg",
		},
		{
			name:     "single dash",
			rawArg:   "-f",
			expected: "f",
		},
		{
			name:     "single dash with value",
			rawArg:   "-f=value",
			expected: "f",
		},
		{
			name:     "no dashes",
			rawArg:   "foo=bar",
			expected: "foo",
		},
		{
			name:     "no dashes no value",
			rawArg:   "foo",
			expected: "foo",
		},
		{
			name:     "empty value",
			rawArg:   "--foo=",
			expected: "foo",
		},
		{
			name:     "multiple equals",
			rawArg:   "--foo=bar=baz",
			expected: "foo",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := extractArgName(tc.rawArg)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestExtractArgNameWithPrefix(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		rawArg   string
		expected string
	}{
		{
			name:     "simple flag",
			rawArg:   "--foo",
			expected: "--foo",
		},
		{
			name:     "value argument",
			rawArg:   "--foo=bar",
			expected: "--foo",
		},
		{
			name:     "hyphenated argument",
			rawArg:   "--my-arg",
			expected: "--my-arg",
		},
		{
			name:     "hyphenated argument with value",
			rawArg:   "--my-arg=123",
			expected: "--my-arg",
		},
		{
			name:     "single dash",
			rawArg:   "-f",
			expected: "-f",
		},
		{
			name:     "single dash with value",
			rawArg:   "-f=value",
			expected: "-f",
		},
		{
			name:     "empty value",
			rawArg:   "--foo=",
			expected: "--foo",
		},
		{
			name:     "multiple equals",
			rawArg:   "--foo=bar=baz",
			expected: "--foo",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := extractArgNameWithPrefix(tc.rawArg)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestBuildEnvVarName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		appName    string
		serverName string
		argName    string
		expected   string
	}{
		{
			name:       "simple case",
			appName:    "mcpd",
			serverName: "test",
			argName:    "foo",
			expected:   "MCPD__TEST__FOO",
		},
		{
			name:       "hyphenated server name",
			appName:    "mcpd",
			serverName: "my-server",
			argName:    "foo",
			expected:   "MCPD__MY_SERVER__FOO",
		},
		{
			name:       "hyphenated arg name",
			appName:    "mcpd",
			serverName: "test",
			argName:    "my-arg",
			expected:   "MCPD__TEST__MY_ARG",
		},
		{
			name:       "all hyphenated",
			appName:    "my-app",
			serverName: "my-server",
			argName:    "my-arg",
			expected:   "MY_APP__MY_SERVER__MY_ARG",
		},
		{
			name:       "mixed case inputs",
			appName:    "McPd",
			serverName: "Test-Server",
			argName:    "My-Arg",
			expected:   "MCPD__TEST_SERVER__MY_ARG",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := buildEnvVarName(tc.appName, tc.serverName, tc.argName)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestTransformArg(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		appName    string
		serverName string
		rawArg     string
		expected   *valueArgTransformation
	}{
		{
			name:       "value argument",
			appName:    "mcpd",
			serverName: "test",
			rawArg:     "--foo=bar",
			expected: &valueArgTransformation{
				Raw:             "--foo=bar",
				Name:            "foo",
				EnvVarName:      "MCPD__TEST__FOO",
				EnvVarReference: "${MCPD__TEST__FOO}",
				FormattedArg:    "--foo=${MCPD__TEST__FOO}",
			},
		},
		{
			name:       "hyphenated flag",
			appName:    "mcpd",
			serverName: "test",
			rawArg:     "--my-arg",
			expected: &valueArgTransformation{
				Raw:             "--my-arg",
				Name:            "my-arg",
				EnvVarName:      "MCPD__TEST__MY_ARG",
				EnvVarReference: "${MCPD__TEST__MY_ARG}",
				FormattedArg:    "--my-arg=${MCPD__TEST__MY_ARG}",
			},
		},
		{
			name:       "hyphenated value argument",
			appName:    "mcpd",
			serverName: "test",
			rawArg:     "--my-arg=123",
			expected: &valueArgTransformation{
				Raw:             "--my-arg=123",
				Name:            "my-arg",
				EnvVarName:      "MCPD__TEST__MY_ARG",
				EnvVarReference: "${MCPD__TEST__MY_ARG}",
				FormattedArg:    "--my-arg=${MCPD__TEST__MY_ARG}",
			},
		},
		{
			name:       "hyphenated server name",
			appName:    "mcpd",
			serverName: "my-server",
			rawArg:     "--config=file.json",
			expected: &valueArgTransformation{
				Raw:             "--config=file.json",
				Name:            "config",
				EnvVarName:      "MCPD__MY_SERVER__CONFIG",
				EnvVarReference: "${MCPD__MY_SERVER__CONFIG}",
				FormattedArg:    "--config=${MCPD__MY_SERVER__CONFIG}",
			},
		},
		{
			name:       "single dash argument",
			appName:    "mcpd",
			serverName: "test",
			rawArg:     "-f=file.txt",
			expected: &valueArgTransformation{
				Raw:             "-f=file.txt",
				Name:            "f",
				EnvVarName:      "MCPD__TEST__F",
				EnvVarReference: "${MCPD__TEST__F}",
				FormattedArg:    "-f=${MCPD__TEST__F}",
			},
		},
		{
			name:       "empty value",
			appName:    "mcpd",
			serverName: "test",
			rawArg:     "--empty=",
			expected: &valueArgTransformation{
				Raw:             "--empty=",
				Name:            "empty",
				EnvVarName:      "MCPD__TEST__EMPTY",
				EnvVarReference: "${MCPD__TEST__EMPTY}",
				FormattedArg:    "--empty=${MCPD__TEST__EMPTY}",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := transformValueArg(tc.appName, tc.serverName, tc.rawArg)
			require.Equal(t, tc.expected, result)
		})
	}
}
