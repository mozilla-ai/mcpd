package daemon

import (
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/mark3labs/mcp-go/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mozilla-ai/mcpd/v2/internal/domain"
)

// Mock implementations for testing
type mockClientManager struct{}

func (m *mockClientManager) Add(name string, c client.MCPClient, tools []string) {
}

func (m *mockClientManager) Client(name string) (client.MCPClient, bool) {
	return nil, false
}

func (m *mockClientManager) Tools(name string) ([]string, bool) {
	return nil, false
}

func (m *mockClientManager) List() []string {
	return nil
}

func (m *mockClientManager) UpdateTools(name string, tools []string) error {
	return nil
}

func (m *mockClientManager) Remove(name string) {
}

type mockHealthTracker struct{}

func (m *mockHealthTracker) Status(name string) (domain.ServerHealth, error) {
	return domain.ServerHealth{}, nil
}

func (m *mockHealthTracker) List() []domain.ServerHealth {
	return nil
}

func (m *mockHealthTracker) Update(name string, status domain.HealthStatus, latency *time.Duration) error {
	return nil
}

func (m *mockHealthTracker) Add(name string) {
	// Mock implementation.
}

func (m *mockHealthTracker) Remove(name string) {
	// Mock implementation.
}

func TestDaemon_APIDependencies_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		deps    APIDependencies
		wantErr string
	}{
		{
			name: "valid dependencies",
			deps: APIDependencies{
				Logger:        hclog.NewNullLogger(),
				ClientManager: &mockClientManager{},
				HealthTracker: &mockHealthTracker{},
				Addr:          "localhost:8090",
			},
		},
		{
			name: "nil logger",
			deps: APIDependencies{
				Logger:        nil,
				ClientManager: &mockClientManager{},
				HealthTracker: &mockHealthTracker{},
				Addr:          "localhost:8090",
			},
			wantErr: "logger cannot be nil",
		},
		{
			name: "nil client manager",
			deps: APIDependencies{
				Logger:        hclog.NewNullLogger(),
				ClientManager: nil,
				HealthTracker: &mockHealthTracker{},
				Addr:          "localhost:8090",
			},
			wantErr: "client manager cannot be nil",
		},
		{
			name: "nil health tracker",
			deps: APIDependencies{
				Logger:        hclog.NewNullLogger(),
				ClientManager: &mockClientManager{},
				HealthTracker: nil,
				Addr:          "localhost:8090",
			},
			wantErr: "health tracker cannot be nil",
		},
		{
			name: "invalid address",
			deps: APIDependencies{
				Logger:        hclog.NewNullLogger(),
				ClientManager: &mockClientManager{},
				HealthTracker: &mockHealthTracker{},
				Addr:          "invalid-address",
			},
			wantErr: "invalid API address 'invalid-address': invalid address format: address invalid-address: missing port in address",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.deps.Validate()

			if tc.wantErr == "" {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)
				require.EqualError(t, err, tc.wantErr)
			}
		})
	}
}
