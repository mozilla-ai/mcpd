package api

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/mozilla-ai/mcpd/v2/internal/domain"
	"github.com/mozilla-ai/mcpd/v2/internal/errors"
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

// mockHealthMonitor implements the MCPHealthMonitor interface for testing.
type mockHealthMonitor struct {
	servers map[string]domain.ServerHealth
}

func newMockHealthMonitor() *mockHealthMonitor {
	return &mockHealthMonitor{
		servers: make(map[string]domain.ServerHealth),
	}
}

func (m *mockHealthMonitor) Status(name string) (domain.ServerHealth, error) {
	if health, ok := m.servers[name]; ok {
		return health, nil
	}
	return domain.ServerHealth{}, fmt.Errorf("%w: %s", errors.ErrHealthNotTracked, name)
}

func (m *mockHealthMonitor) List() []domain.ServerHealth {
	servers := make([]domain.ServerHealth, 0, len(m.servers))
	for _, server := range m.servers {
		servers = append(servers, server)
	}
	return servers
}

func (m *mockHealthMonitor) Update(name string, status domain.HealthStatus, latency *time.Duration) error {
	m.servers[name] = domain.ServerHealth{
		Name:    name,
		Status:  status,
		Latency: latency,
	}
	return nil
}

func TestHandleHealthServer_ServerNotTracked(t *testing.T) {
	t.Parallel()

	monitor := newMockHealthMonitor()

	// Try to get health for a server that doesn't exist.
	result, err := handleHealthServer(monitor, "nonexistent-server")
	require.Error(t, err)
	require.Nil(t, result)

	require.ErrorIs(t, err, errors.ErrHealthNotTracked)
}

func TestHandleHealthServer_ServerExists(t *testing.T) {
	t.Parallel()

	monitor := newMockHealthMonitor()

	// Add a server to the monitor.
	latency := 100 * time.Millisecond
	err := monitor.Update("test-server", domain.HealthStatusOK, &latency)
	require.NoError(t, err)

	// Get health for existing server.
	result, err := handleHealthServer(monitor, "test-server")
	require.NoError(t, err)
	require.NotNil(t, result)

	require.Equal(t, "test-server", result.Body.Name)
	require.Equal(t, HealthStatusOK, result.Body.Status)
	require.NotNil(t, result.Body.Latency)
	require.Equal(t, "100ms", *result.Body.Latency)
}
