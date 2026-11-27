package api

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/mozilla-ai/mcpd/internal/domain"
	"github.com/mozilla-ai/mcpd/internal/errors"
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
// It indexes by each name's key, matching the real tracker, so a test exercises the same
// case-insensitive lookup the API relies on.
type mockHealthMonitor struct {
	servers map[string]domain.ServerHealth
}

func newMockHealthMonitor() *mockHealthMonitor {
	return &mockHealthMonitor{
		servers: make(map[string]domain.ServerHealth),
	}
}

func (m *mockHealthMonitor) Add(name domain.ServerName) {
	// TODO implement MCPHealthMonitor 'Add' when needed.
	panic("implement me")
}

func (m *mockHealthMonitor) List() []domain.ServerHealth {
	servers := make([]domain.ServerHealth, 0, len(m.servers))
	for _, server := range m.servers {
		servers = append(servers, server)
	}

	return servers
}

func (m *mockHealthMonitor) Remove(name domain.ServerName) {
	// TODO implement MCPHealthMonitor 'Remove' when needed.
	panic("implement me")
}

func (m *mockHealthMonitor) Status(name domain.ServerName) (domain.ServerHealth, error) {
	if health, ok := m.servers[name.Key()]; ok {
		return health, nil
	}

	return domain.ServerHealth{}, fmt.Errorf("%w: %s", errors.ErrHealthNotTracked, name)
}

func (m *mockHealthMonitor) Update(
	name domain.ServerName,
	status domain.HealthStatus,
	latency *time.Duration,
) error {
	m.servers[name.Key()] = domain.ServerHealth{
		Name:    name,
		Status:  status,
		Latency: latency,
	}

	return nil
}

func TestHandleStatus_ServerNotTracked(t *testing.T) {
	t.Parallel()

	monitor := newMockHealthMonitor()

	// Try to get health for a server that doesn't exist.
	result, err := (&healthRoutes{monitor: monitor}).handleStatus("nonexistent-server")
	require.Error(t, err)
	require.Nil(t, result)

	require.ErrorIs(t, err, errors.ErrHealthNotTracked)
}

func TestHandleStatus_ServerExists(t *testing.T) {
	t.Parallel()

	monitor := newMockHealthMonitor()

	// Add a server to the monitor.
	latency := 100 * time.Millisecond
	err := monitor.Update("test-server", domain.HealthStatusOK, &latency)
	require.NoError(t, err)

	// Get health for existing server.
	result, err := (&healthRoutes{monitor: monitor}).handleStatus("test-server")
	require.NoError(t, err)
	require.NotNil(t, result)

	require.Equal(t, "test-server", result.Body.Name)
	require.Equal(t, HealthStatusOK, result.Body.Status)
	require.NotNil(t, result.Body.Latency)
	require.Equal(t, "100ms", *result.Body.Latency)
}

// TestDomainServerHealth_ToAPIType_PreservesNameCase pins the name the health endpoints report.
// A client reads a server's name from the server listing and uses it in a health URL, so the two surfaces
// have to agree on the name.
func TestDomainServerHealth_ToAPIType_PreservesNameCase(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		server string
	}{
		{
			name:   "pascal case",
			server: "GitHub",
		},
		{
			name:   "camel case",
			server: "gitHub",
		},
		{
			name:   "already lowercase",
			server: "github",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := DomainServerHealth(domain.ServerHealth{
				Name:   domain.ServerName(tc.server),
				Status: domain.HealthStatusOK,
			}).ToAPIType()
			require.NoError(t, err)
			require.Equal(t, tc.server, got.Name)
		})
	}
}
