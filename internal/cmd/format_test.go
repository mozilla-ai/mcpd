package cmd

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAllowedOutputFormats(t *testing.T) {
	t.Parallel()

	want := OutputFormats{FormatJSON, FormatText, FormatYAML}
	got := AllowedOutputFormats()

	require.Equal(t, want, got)
}

func TestOutputFormats_String(t *testing.T) {
	t.Parallel()

	f := AllowedOutputFormats()
	// Should join lower-case names in lexicographical order
	want := "json, text, yaml"
	got := f.String()

	require.Equal(t, want, got)
}

func TestOutputFormat_StringAndType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		fmt  OutputFormat
		want string
	}{
		{
			"JSON",
			FormatJSON,
			"json",
		},
		{
			"Text",
			FormatText,
			"text",
		},
		{
			"YAML",
			FormatYAML,
			"yaml",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tc.want, tc.fmt.String())
			require.Equal(t, "format", tc.fmt.Type())
		})
	}
}

func TestOutputFormat_Set_Valid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  OutputFormat
	}{
		{
			"json",
			"json",
			FormatJSON,
		},
		{
			"text",
			"text",
			FormatText,
		},
		{
			"yaml",
			"yaml",
			FormatYAML,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var f OutputFormat
			err := f.Set(tc.input)
			require.NoError(t, err)
			require.Equal(t, tc.want, f)
		})
	}
}

func TestOutputFormat_Set_Invalid(t *testing.T) {
	t.Parallel()

	invalid := "xml"
	var f OutputFormat
	err := f.Set(invalid)
	require.Error(t, err)
	// error message should mention invalid value and allowed list
	require.ErrorContains(t, err, fmt.Sprintf("invalid format '%s'", invalid))
	allowed := AllowedOutputFormats()
	require.Contains(t, err.Error(), allowed.String())
}
