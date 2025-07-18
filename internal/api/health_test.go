package api

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mozilla-ai/mcpd/v2/internal/domain"
)

func TestParseHealthStatus_ValidCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    domain.HealthStatus
		expected HealthStatus
	}{
		{
			"ok",
			domain.HealthStatusOK,
			HealthStatusOK,
		},
		{
			"timeout",
			domain.HealthStatusTimeout,
			HealthStatusTimeout,
		},
		{
			"unreachable",
			domain.HealthStatusUnreachable,
			HealthStatusUnreachable,
		},
		{
			"unknown",
			domain.HealthStatusUnknown,
			HealthStatusUnknown,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := parseHealthStatus(tc.input)
			require.NoError(t, err)
			require.Equal(t, tc.expected, got)
		})
	}
}

func TestParseHealthStatus_InvalidCase(t *testing.T) {
	t.Parallel()

	input := domain.HealthStatus("invalid-status")
	_, err := parseHealthStatus(input)
	require.Error(t, err)
	require.EqualError(t, err, fmt.Sprintf("unknown health status: %s", input))
}
