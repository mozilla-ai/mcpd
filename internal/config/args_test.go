package config

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "long flag with separate value",
			input:    []string{"--foo", "bar"},
			expected: []string{"--foo=bar"},
		},
		{
			name:     "long flag with equals value",
			input:    []string{"--foo=bar"},
			expected: []string{"--foo=bar"},
		},
		{
			name:     "short flag with separate value",
			input:    []string{"-f", "bar"},
			expected: []string{"-f=bar"},
		},
		{
			name:     "short flag with equals value",
			input:    []string{"-f=bar"},
			expected: []string{"-f=bar"},
		},
		{
			name:     "combined short flags",
			input:    []string{"-xzv"},
			expected: []string{"-x", "-z", "-v"},
		},
		{
			name:     "positional args are skipped",
			input:    []string{"--foo", "bar", "main.go", "--baz", "qux"},
			expected: []string{"--foo=bar", "--baz=qux"},
		},
		{
			name:     "boolean long and short flags",
			input:    []string{"--verbose", "-h"},
			expected: []string{"--verbose", "-h"},
		},
		{
			name:     "trailing whitespace trimmed",
			input:    []string{"--config ", " dev.toml "},
			expected: []string{"--config=dev.toml"},
		},
		{
			name:     "short flag not followed by value",
			input:    []string{"-f", "-x"},
			expected: []string{"-f", "-x"},
		},
		{
			name:     "long flag not followed by value",
			input:    []string{"--flag", "--other"},
			expected: []string{"--flag", "--other"},
		},
		{
			name:     "short flag with equals value",
			input:    []string{"-f=bar"},
			expected: []string{"-f=bar"},
		},
		{
			name:     "malformed flag like '--='",
			input:    []string{"--="},
			expected: []string{"--="}, // not our job to validate semantics
		},
		{
			name:     "flag with only equal sign",
			input:    []string{"--foo=", "-x="},
			expected: []string{"--foo=", "-x="}, // empty values are valid
		},
		{
			name:     "combined short flag with digit",
			input:    []string{"-x9z"},
			expected: []string{"-x", "-9", "-z"}, // odd but syntactically valid
		},
		{
			name:     "flag followed by another flag-looking value",
			input:    []string{"--foo", "--bar"},
			expected: []string{"--foo", "--bar"}, // --foo is likely a bool
		},
		{
			name:     "flag followed by whitespace-only value",
			input:    []string{"--foo", "   "},
			expected: []string{"--foo="}, // gets trimmed to --foo=, probably invalid but handled
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			actual := NormalizeArgs(tc.input)
			require.Equal(t, tc.expected, actual)
		})
	}
}

func TestRemoveMatchingFlags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    []string
		remove   []string
		expected []string
	}{
		{
			name:     "remove bare and assigned long flags",
			input:    []string{"--foo", "--foo=bar", "--bar", "--bar=baz", "--baz"},
			remove:   []string{"--foo", "--bar"},
			expected: []string{"--baz"},
		},
		{
			name:     "preserve unrelated flags",
			input:    []string{"--alpha", "--beta=123"},
			remove:   []string{"--gamma"},
			expected: []string{"--alpha", "--beta=123"},
		},
		{
			name:     "no flags to remove",
			input:    []string{"--x", "--y=val"},
			remove:   []string{},
			expected: []string{"--x", "--y=val"},
		},
		{
			name:     "remove everything matching a long flag prefix",
			input:    []string{"--flag", "--flag=123", "--flag=value", "--flag="},
			remove:   []string{"--flag"},
			expected: []string{},
		},
		{
			name:     "short flags are not removed",
			input:    []string{"-f", "-f=123", "--foo", "--foo=bar"},
			remove:   []string{"--foo"},
			expected: []string{"-f", "-f=123"},
		},
		{
			name:     "remove multiple different flags",
			input:    []string{"--one", "--two", "--two=dos", "--three=3"},
			remove:   []string{"--one", "--two"},
			expected: []string{"--three=3"},
		},
		{
			name:     "do not remove short flags unless explicitly listed",
			input:    []string{"-f", "-f=bar", "-x", "--flag"},
			remove:   []string{"--flag"},
			expected: []string{"-f", "-f=bar", "-x"},
		},
		{
			name:     "remove short flag only if exact match",
			input:    []string{"-f", "-f=bar", "-x"},
			remove:   []string{"-f"},
			expected: []string{"-x"},
		},
		{
			name:     "remove both short and long if both in remove list",
			input:    []string{"-f", "--flag", "--flag=val"},
			remove:   []string{"-f", "--flag"},
			expected: []string{},
		},
		{
			name:     "remove short flag with value",
			input:    []string{"-f=123", "-x=1", "-y"},
			remove:   []string{"-f"},
			expected: []string{"-x=1", "-y"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := RemoveMatchingFlags(tc.input, tc.remove)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestMergeArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		a        []string
		b        []string
		expected []string
	}{
		{
			name:     "merge disjoint sets",
			a:        []string{"--foo", "--bar=baz"},
			b:        []string{"--baz=qux", "--flag"},
			expected: []string{"--foo", "--bar=baz", "--baz=qux", "--flag"},
		},
		{
			name:     "overwrite values from b",
			a:        []string{"--config=dev.toml", "--verbose"},
			b:        []string{"--config=prod.toml"},
			expected: []string{"--config=prod.toml", "--verbose"},
		},
		{
			name:     "bool flag in a overwritten by value flag in b",
			a:        []string{"--debug"},
			b:        []string{"--debug=true"},
			expected: []string{"--debug=true"},
		},
		{
			name:     "value flag in a overwritten by bool flag in b",
			a:        []string{"--debug=true"},
			b:        []string{"--debug"},
			expected: []string{"--debug"},
		},
		{
			name:     "no overlap between a and b",
			a:        []string{"--x", "--y=1"},
			b:        []string{"--a", "--b=2"},
			expected: []string{"--x", "--y=1", "--a", "--b=2"},
		},
		{
			name:     "empty a",
			a:        []string{},
			b:        []string{"--flag", "--opt=value"},
			expected: []string{"--flag", "--opt=value"},
		},
		{
			name:     "empty b",
			a:        []string{"--a=1", "--b"},
			b:        []string{},
			expected: []string{"--a=1", "--b"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			actual := MergeArgs(tc.a, tc.b)

			// sort for order-insensitive comparison since map iteration is unordered.
			sort.Strings(actual)
			sort.Strings(tc.expected)

			require.Equal(t, tc.expected, actual)
		})
	}
}

func TestMergeArgs_OrderSensitive(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		a        []string
		b        []string
		expected []string
	}{
		{
			name:     "disjoint merge preserves order a then b",
			a:        []string{"--foo", "--bar=baz"},
			b:        []string{"--baz=qux", "--flag"},
			expected: []string{"--foo", "--bar=baz", "--baz=qux", "--flag"},
		},
		{
			name:     "b overwrites value in a",
			a:        []string{"--config=dev.toml", "--verbose"},
			b:        []string{"--config=prod.toml"},
			expected: []string{"--config=prod.toml", "--verbose"},
		},
		{
			name:     "b overwrites boolean with value",
			a:        []string{"--debug"},
			b:        []string{"--debug=true"},
			expected: []string{"--debug=true"},
		},
		{
			name:     "b overwrites value with boolean",
			a:        []string{"--debug=true"},
			b:        []string{"--debug"},
			expected: []string{"--debug"},
		},
		{
			name:     "non-overlapping preserves a then b order",
			a:        []string{"--x", "--y=1"},
			b:        []string{"--a", "--b=2"},
			expected: []string{"--x", "--y=1", "--a", "--b=2"},
		},
		{
			name:     "empty a includes b in order",
			a:        []string{},
			b:        []string{"--flag", "--opt=value"},
			expected: []string{"--flag", "--opt=value"},
		},
		{
			name:     "empty b preserves a order",
			a:        []string{"--a=1", "--b"},
			b:        []string{},
			expected: []string{"--a=1", "--b"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			actual := MergeArgs(tc.a, tc.b)
			require.Equal(t, tc.expected, actual)
		})
	}
}

func TestParseArg(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected argEntry
	}{
		{
			name:  "boolean flag",
			input: "--verbose",
			expected: argEntry{
				key: "--verbose",
			},
		},
		{
			name:  "key-value pair",
			input: "--output=file.txt",
			expected: argEntry{
				key:   "--output",
				value: "file.txt",
			},
		},
		{
			name:  "key-value with spaces",
			input: " --config = config.yaml ",
			expected: argEntry{
				key:   "--config",
				value: "config.yaml",
			},
		},
		{
			name:  "empty value",
			input: "--flag=",
			expected: argEntry{
				key:   "--flag",
				value: "",
			},
		},
		{
			name:  "value with equals sign",
			input: "--env=NODE_ENV=production",
			expected: argEntry{
				key:   "--env",
				value: "NODE_ENV=production",
			},
		},
		{
			name:  "single character flag",
			input: "-v",
			expected: argEntry{
				key: "-v",
			},
		},
		{
			name:  "empty string",
			input: "",
			expected: argEntry{
				key:   "",
				value: "",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := parseArg(tc.input)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestArgEntry_HasValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		entry    argEntry
		expected bool
	}{
		{
			name:     "has non-empty value",
			entry:    argEntry{key: "--output", value: "file.txt"},
			expected: true,
		},
		{
			name:     "empty value",
			entry:    argEntry{key: "--flag", value: ""},
			expected: false,
		},
		{
			name:     "whitespace only value",
			entry:    argEntry{key: "--flag", value: "   "},
			expected: false,
		},
		{
			name:     "tab and newline value",
			entry:    argEntry{key: "--flag", value: "\t\n"},
			expected: false,
		},
		{
			name:     "single space value",
			entry:    argEntry{key: "--flag", value: " "},
			expected: false,
		},
		{
			name:     "value with leading/trailing spaces",
			entry:    argEntry{key: "--config", value: " config.yaml "},
			expected: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tc.expected, tc.entry.hasValue())
		})
	}
}

func TestArgEntry_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		entry    argEntry
		expected string
	}{
		{
			name:     "boolean flag",
			entry:    argEntry{key: "--verbose", value: ""},
			expected: "--verbose",
		},
		{
			name:     "key-value pair",
			entry:    argEntry{key: "--output", value: "file.txt"},
			expected: "--output=file.txt",
		},
		{
			name:     "value with spaces gets preserved",
			entry:    argEntry{key: "--message", value: "hello world"},
			expected: "--message=hello world",
		},
		{
			name:     "empty key and value",
			entry:    argEntry{key: "", value: ""},
			expected: "",
		},
		{
			name:     "whitespace-only value treated as boolean",
			entry:    argEntry{key: "--flag", value: "   "},
			expected: "--flag",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tc.expected, tc.entry.String())
		})
	}
}

func TestProcessAllArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "empty input",
			input:    []string{},
			expected: []string{},
		},
		{
			name:     "only positional args",
			input:    []string{"pos1", "pos2", "pos3"},
			expected: []string{"pos1", "pos2", "pos3"},
		},
		{
			name:     "only flags with embedded values",
			input:    []string{"--flag=value", "--other=val"},
			expected: []string{"--flag=value", "--other=val"},
		},
		{
			name:     "only boolean flags",
			input:    []string{"--verbose", "--debug"},
			expected: []string{"--verbose", "--debug"},
		},
		{
			name:     "flags with separate values (regression test)",
			input:    []string{"--config", "dev.toml", "--output", "result.txt"},
			expected: []string{"--config=dev.toml", "--output=result.txt"},
		},
		{
			name:     "mixed flags and positional args maintaining order",
			input:    []string{"pos1", "--flag=value", "pos2", "--verbose", "pos3"},
			expected: []string{"pos1", "--flag=value", "pos2", "--verbose=pos3"},
		},
		{
			name:     "flag with separate value followed by positional",
			input:    []string{"--config", "dev.toml", "positional"},
			expected: []string{"--config=dev.toml", "positional"},
		},
		{
			name:     "positional followed by flag with separate value",
			input:    []string{"positional", "--config", "dev.toml"},
			expected: []string{"positional", "--config=dev.toml"},
		},
		{
			name:     "complex mix with various flag formats",
			input:    []string{"pos1", "--flag=embedded", "--other", "separate", "--bool", "pos2"},
			expected: []string{"pos1", "--flag=embedded", "--other=separate", "--bool=pos2"},
		},
		{
			name:     "short flags with separate values",
			input:    []string{"-f", "value", "-x", "other"},
			expected: []string{"-f=value", "-x=other"},
		},
		{
			name:     "combined short flags",
			input:    []string{"-xyz", "pos1"},
			expected: []string{"-x", "-y", "-z", "pos1"},
		},
		{
			name:     "flag followed by another flag (no value consumption)",
			input:    []string{"--verbose", "--debug", "pos1"},
			expected: []string{"--verbose", "--debug=pos1"},
		},
		{
			name:     "multiple consecutive flags with separate values",
			input:    []string{"--first", "val1", "--second", "val2", "--third", "val3"},
			expected: []string{"--first=val1", "--second=val2", "--third=val3"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			actual := ProcessAllArgs(tc.input)
			require.Equal(t, tc.expected, actual, "ProcessAllArgs should preserve order and normalize flags correctly")
		})
	}
}
