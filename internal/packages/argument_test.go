package packages

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
)

// Test data used across multiple tests
func testArguments(t *testing.T) Arguments {
	t.Helper()

	return Arguments{
		"DATABASE_URL": {
			Name:         "DATABASE_URL",
			Description:  "Database connection URL",
			Required:     true,
			Example:      "postgresql://localhost:5432/mydb",
			VariableType: VariableTypeEnv,
		},
		"API_KEY": {
			Name:         "API_KEY",
			Description:  "API key for external service",
			Required:     true,
			Example:      "sk-1234567890abcdef",
			VariableType: VariableTypeEnv,
		},
		"DEBUG_MODE": {
			Name:         "DEBUG_MODE",
			Description:  "Enable debug mode",
			Required:     false,
			Example:      "true",
			VariableType: VariableTypeEnv,
		},
		"OPTIONAL_CONFIG": {
			Name:         "OPTIONAL_CONFIG",
			Description:  "",
			Required:     false,
			Example:      "",
			VariableType: VariableTypeEnv,
		},
		"--port": {
			Name:         "--port",
			Description:  "Port to listen on",
			Required:     true,
			Example:      "8080",
			VariableType: VariableTypeArg,
		},
		"--verbose": {
			Name:         "--verbose",
			Description:  "Enable verbose output",
			Required:     false,
			Example:      "true",
			VariableType: VariableTypeArg,
		},
		"--config": {
			Name:         "--config",
			Description:  "",
			Required:     true,
			Example:      "/path/to/config.json",
			VariableType: VariableTypeArg,
		},
	}
}

func TestRequired(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		metadata ArgumentMetadata
		expected bool
	}{
		{
			name: "required argument",
			metadata: ArgumentMetadata{
				Required: true,
			},
			expected: true,
		},
		{
			name: "optional argument",
			metadata: ArgumentMetadata{
				Required: false,
			},
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := Required("test_name", tc.metadata)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestEnvVar(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		metadata ArgumentMetadata
		expected bool
	}{
		{
			name: "environment variable",
			metadata: ArgumentMetadata{
				VariableType: VariableTypeEnv,
			},
			expected: true,
		},
		{
			name: "command line argument",
			metadata: ArgumentMetadata{
				VariableType: VariableTypeArg,
			},
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := EnvVar("test_name", tc.metadata)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestArgument(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		metadata ArgumentMetadata
		expected bool
	}{
		{
			name: "command line argument",
			metadata: ArgumentMetadata{
				VariableType: VariableTypeArg,
			},
			expected: true,
		},
		{
			name: "environment variable",
			metadata: ArgumentMetadata{
				VariableType: VariableTypeEnv,
			},
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := Argument("test_name", tc.metadata)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestArguments_Names(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		arguments Arguments
		expected  []string
	}{
		{
			name:      "empty arguments",
			arguments: Arguments{},
			expected:  []string{},
		},
		{
			name: "single argument",
			arguments: Arguments{
				"TEST_VAR": {
					Description:  "Test variable",
					Required:     true,
					VariableType: VariableTypeEnv,
				},
			},
			expected: []string{"TEST_VAR"},
		},
		{
			name: "multiple arguments",
			arguments: Arguments{
				"VAR_A": {
					Description:  "Variable A",
					Required:     true,
					VariableType: VariableTypeEnv,
				},
				"VAR_B": {
					Description:  "Variable B",
					Required:     false,
					VariableType: VariableTypeArg,
				},
			},
			expected: []string{"VAR_A", "VAR_B"},
		},
		{
			name:      "test data arguments",
			arguments: testArguments(t),
			expected: []string{
				"DATABASE_URL", "API_KEY", "DEBUG_MODE", "OPTIONAL_CONFIG",
				"--port", "--verbose", "--config",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := tc.arguments.Names()

			// Sort both slices for reliable comparison since map iteration order is not guaranteed
			slices.Sort(result)
			slices.Sort(tc.expected)

			require.ElementsMatch(t, tc.expected, result)
		})
	}
}

func TestFilterArguments(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		arguments  Arguments
		predicates []func(name string, data ArgumentMetadata) bool
		expected   []string // Expected argument names
	}{
		{
			name:       "no predicates returns all",
			arguments:  testArguments(t),
			predicates: []func(name string, data ArgumentMetadata) bool{},
			expected: []string{
				"DATABASE_URL", "API_KEY", "DEBUG_MODE", "OPTIONAL_CONFIG",
				"--port", "--verbose", "--config",
			},
		},
		{
			name:       "filter by required only",
			arguments:  testArguments(t),
			predicates: []func(name string, data ArgumentMetadata) bool{Required},
			expected:   []string{"DATABASE_URL", "API_KEY", "--port", "--config"},
		},
		{
			name:       "filter by env vars only",
			arguments:  testArguments(t),
			predicates: []func(name string, data ArgumentMetadata) bool{EnvVar},
			expected:   []string{"DATABASE_URL", "API_KEY", "DEBUG_MODE", "OPTIONAL_CONFIG"},
		},
		{
			name:       "filter by arguments only",
			arguments:  testArguments(t),
			predicates: []func(name string, data ArgumentMetadata) bool{Argument},
			expected:   []string{"--port", "--verbose", "--config"},
		},
		{
			name:       "filter by required env vars",
			arguments:  testArguments(t),
			predicates: []func(name string, data ArgumentMetadata) bool{Required, EnvVar},
			expected:   []string{"DATABASE_URL", "API_KEY"},
		},
		{
			name:       "filter by required arguments",
			arguments:  testArguments(t),
			predicates: []func(name string, data ArgumentMetadata) bool{Required, Argument},
			expected:   []string{"--port", "--config"},
		},
		{
			name:      "filter by custom predicate - has description",
			arguments: testArguments(t),
			predicates: []func(name string, data ArgumentMetadata) bool{
				func(_ string, data ArgumentMetadata) bool {
					return data.Description != ""
				},
			},
			expected: []string{"DATABASE_URL", "API_KEY", "DEBUG_MODE", "--port", "--verbose"},
		},
		{
			name:      "filter by name prefix",
			arguments: testArguments(t),
			predicates: []func(name string, data ArgumentMetadata) bool{
				func(name string, _ ArgumentMetadata) bool {
					return len(name) > 0 && name[0] == '-'
				},
			},
			expected: []string{"--port", "--verbose", "--config"},
		},
		{
			name:      "multiple predicates with no matches",
			arguments: testArguments(t),
			predicates: []func(name string, data ArgumentMetadata) bool{
				Required,
				func(_ string, data ArgumentMetadata) bool {
					return data.Description == "non-existent description"
				},
			},
			expected: []string{},
		},
		{
			name:       "empty arguments",
			arguments:  Arguments{},
			predicates: []func(name string, data ArgumentMetadata) bool{Required},
			expected:   []string{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := FilterArguments(tc.arguments, tc.predicates...)
			resultNames := result.Names()

			// Sort both slices for reliable comparison
			slices.Sort(resultNames)
			slices.Sort(tc.expected)

			require.ElementsMatch(t, tc.expected, resultNames)
		})
	}
}

func TestArguments_FilterBy(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		arguments  Arguments
		predicates []func(name string, data ArgumentMetadata) bool
		expected   []string
	}{
		{
			name:       "filter by required",
			arguments:  testArguments(t),
			predicates: []func(name string, data ArgumentMetadata) bool{Required},
			expected:   []string{"DATABASE_URL", "API_KEY", "--port", "--config"},
		},
		{
			name:       "filter by env vars",
			arguments:  testArguments(t),
			predicates: []func(name string, data ArgumentMetadata) bool{EnvVar},
			expected:   []string{"DATABASE_URL", "API_KEY", "DEBUG_MODE", "OPTIONAL_CONFIG"},
		},
		{
			name:       "chained predicates - required env vars",
			arguments:  testArguments(t),
			predicates: []func(name string, data ArgumentMetadata) bool{Required, EnvVar},
			expected:   []string{"DATABASE_URL", "API_KEY"},
		},
		{
			name:       "empty arguments",
			arguments:  Arguments{},
			predicates: []func(name string, data ArgumentMetadata) bool{Required},
			expected:   []string{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := tc.arguments.FilterBy(tc.predicates...)
			resultNames := result.Names()

			// Sort both slices for reliable comparison
			slices.Sort(resultNames)
			slices.Sort(tc.expected)

			require.ElementsMatch(t, tc.expected, resultNames)
		})
	}
}

func TestFilterBy_Chaining(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		arguments Arguments
		expected  []string
	}{
		{
			name:      "chain FilterBy calls - required env vars",
			arguments: testArguments(t),
			expected:  []string{"DATABASE_URL", "API_KEY"},
		},
		{
			name:      "chain FilterBy calls - required args",
			arguments: testArguments(t),
			expected:  []string{"--port", "--config"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var result Arguments

			if tc.name == "chain FilterBy calls - required env vars" {
				result = tc.arguments.FilterBy(EnvVar).FilterBy(Required)
			} else {
				result = tc.arguments.FilterBy(Argument).FilterBy(Required)
			}

			resultNames := result.Names()

			// Sort both slices for reliable comparison
			slices.Sort(resultNames)
			slices.Sort(tc.expected)

			require.Equal(t, tc.expected, resultNames)
		})
	}
}

func TestFilterArguments_EdgeEases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		arguments Arguments
		predicate func(name string, data ArgumentMetadata) bool
		expected  int
	}{
		{
			name:      "nil arguments map",
			arguments: nil,
			predicate: Required,
			expected:  0,
		},
		{
			name: "predicate always returns false",
			arguments: Arguments{
				"TEST": {Required: true, VariableType: VariableTypeEnv},
			},
			predicate: func(_ string, _ ArgumentMetadata) bool { return false },
			expected:  0,
		},
		{
			name: "predicate always returns true",
			arguments: Arguments{
				"TEST1": {Required: true, VariableType: VariableTypeEnv},
				"TEST2": {Required: false, VariableType: VariableTypeArg},
			},
			predicate: func(_ string, _ ArgumentMetadata) bool { return true },
			expected:  2,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := FilterArguments(tc.arguments, tc.predicate)
			require.Len(t, result, tc.expected)
		})
	}
}

func TestArgumentMetadata_NameAndExample(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		metadata ArgumentMetadata
		expected ArgumentMetadata
	}{
		{
			name: "all fields set",
			metadata: ArgumentMetadata{
				Name:         "TEST_VAR",
				Description:  "Test variable",
				Required:     true,
				Example:      "example_value",
				VariableType: VariableTypeEnv,
			},
			expected: ArgumentMetadata{
				Name:         "TEST_VAR",
				Description:  "Test variable",
				Required:     true,
				Example:      "example_value",
				VariableType: VariableTypeEnv,
			},
		},
		{
			name: "empty name and example",
			metadata: ArgumentMetadata{
				Name:         "",
				Description:  "Test variable",
				Required:     false,
				Example:      "",
				VariableType: VariableTypeArg,
			},
			expected: ArgumentMetadata{
				Name:         "",
				Description:  "Test variable",
				Required:     false,
				Example:      "",
				VariableType: VariableTypeArg,
			},
		},
		{
			name: "positional argument with position",
			metadata: ArgumentMetadata{
				Name:         "POS_ARG",
				Description:  "Positional argument",
				Required:     true,
				Example:      "/path/to/file",
				VariableType: VariableTypeArgPositional,
				Position:     &[]int{1}[0],
			},
			expected: ArgumentMetadata{
				Name:         "POS_ARG",
				Description:  "Positional argument",
				Required:     true,
				Example:      "/path/to/file",
				VariableType: VariableTypeArgPositional,
				Position:     &[]int{1}[0],
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tc.expected.Name, tc.metadata.Name)
			require.Equal(t, tc.expected.Example, tc.metadata.Example)
			require.Equal(t, tc.expected.Description, tc.metadata.Description)
			require.Equal(t, tc.expected.Required, tc.metadata.Required)
			require.Equal(t, tc.expected.VariableType, tc.metadata.VariableType)
			if tc.expected.Position != nil {
				require.NotNil(t, tc.metadata.Position)
				require.Equal(t, *tc.expected.Position, *tc.metadata.Position)
			} else {
				require.Nil(t, tc.metadata.Position)
			}
		})
	}
}

func TestArguments_Ordered_NameSetting(t *testing.T) {
	t.Parallel()

	args := Arguments{
		"ENV_VAR": {
			Description:  "Environment variable",
			Required:     true,
			Example:      "env_example",
			VariableType: VariableTypeEnv,
		},
		"POS_ARG": {
			Description:  "Positional argument",
			Required:     true,
			Example:      "pos_example",
			VariableType: VariableTypeArgPositional,
			Position:     &[]int{1}[0],
		},
		"--flag": {
			Description:  "Value flag",
			Required:     false,
			Example:      "flag_example",
			VariableType: VariableTypeArg,
		},
	}

	ordered := args.Ordered()
	require.Len(t, ordered, 3)

	for _, arg := range ordered {
		require.NotEmpty(t, arg.Name, "Name should be set on ordered arguments")

		switch arg.Name {
		case "POS_ARG":
			require.Equal(t, "pos_example", arg.Example)
			require.Equal(t, VariableTypeArgPositional, arg.VariableType)
			require.NotNil(t, arg.Position)
			require.Equal(t, 1, *arg.Position)
		case "ENV_VAR":
			require.Equal(t, "env_example", arg.Example)
			require.Equal(t, VariableTypeEnv, arg.VariableType)
		case "--flag":
			require.Equal(t, "flag_example", arg.Example)
			require.Equal(t, VariableTypeArg, arg.VariableType)
		default:
			t.Errorf("Unexpected argument name: %s", arg.Name)
		}
	}

	require.Equal(t, "POS_ARG", ordered[0].Name, "Positional argument should be first")

	require.Contains(
		t,
		[]string{"--flag", "ENV_VAR"},
		ordered[1].Name,
		"Non-positional should be in alphabetical order",
	)
	require.Contains(
		t,
		[]string{"--flag", "ENV_VAR"},
		ordered[2].Name,
		"Non-positional should be in alphabetical order",
	)
}
