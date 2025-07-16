package cmd

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAllowedExportFormats(t *testing.T) {
	t.Parallel()

	expected := ExportFormats{FormatDotEnv}
	got := AllowedExportFormats()

	require.Len(t, got, len(expected))
	require.Equal(t, expected, got)
}

func TestExportFormats_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		formats ExportFormats
		want    string
	}{
		{
			"single",
			ExportFormats{FormatDotEnv},
			"dotenv",
		},
		{
			"multiple",
			ExportFormats{FormatDotEnv, FormatGitHubActions},
			"dotenv, github",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			s := (&tc.formats).String()
			require.Equal(t, tc.want, s)
		})
	}
}

func TestExportFormat_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		f    ExportFormat
		want string
	}{
		{
			"dotenv",
			FormatDotEnv,
			"dotenv",
		},
		{
			"k8s",
			FormatKubernetesSecret,
			"k8s",
		},
		{
			"gha",
			FormatGitHubActions,
			"github",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			s := (&tc.f).String()
			require.Equal(t, tc.want, s)
		})
	}
}

func TestExportFormat_Set(t *testing.T) {
	t.Parallel()

	t.Run("valid", func(t *testing.T) {
		t.Parallel()

		var f ExportFormat
		err := f.Set("dotenv")
		require.NoError(t, err)
		require.Equal(t, FormatDotEnv, f)
	})

	t.Run("invalid", func(t *testing.T) {
		t.Parallel()

		var f ExportFormat
		err := f.Set("invalid")
		require.Error(t, err)
		require.EqualError(t, err, "invalid format 'invalid', must be one of dotenv")
	})
}

func TestExportFormat_Type(t *testing.T) {
	t.Parallel()

	var f ExportFormat
	require.Equal(t, "format", f.Type())
}
