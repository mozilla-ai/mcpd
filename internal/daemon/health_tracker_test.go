package daemon

import (
	"encoding/json"
	stdErrors "errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/mozilla-ai/mcpd/v2/internal/api"
	"github.com/mozilla-ai/mcpd/v2/internal/domain"
	"github.com/mozilla-ai/mcpd/v2/internal/errors"
)

func TestNewHealthTracker(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		serverNames []string
		wantLen     int
	}{
		{
			name:        "empty server list",
			serverNames: []string{},
			wantLen:     0,
		},
		{
			name:        "nil server list",
			serverNames: nil,
			wantLen:     0,
		},
		{
			name:        "single server",
			serverNames: []string{"server1"},
			wantLen:     1,
		},
		{
			name:        "multiple servers",
			serverNames: []string{"server1", "server2", "server3"},
			wantLen:     3,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tracker := NewHealthTracker(tc.serverNames)
			require.NotNil(t, tracker)
			require.Equal(t, tc.wantLen, len(tracker.statuses))

			// Verify all servers are initialized with unknown status
			for _, name := range tc.serverNames {
				health, exists := tracker.statuses[name]
				require.True(t, exists)
				require.Equal(t, name, health.Name)
				require.Equal(t, domain.HealthStatusUnknown, health.Status)
				require.Nil(t, health.Latency)
				require.Nil(t, health.LastChecked)
				require.Nil(t, health.LastSuccessful)
			}
		})
	}
}

func TestHealthTracker_Status(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		serverNames []string
		queryName   string
		wantError   bool
		wantStatus  domain.HealthStatus
	}{
		{
			name:        "existing server",
			serverNames: []string{"server1", "server2"},
			queryName:   "server1",
			wantError:   false,
			wantStatus:  domain.HealthStatusUnknown,
		},
		{
			name:        "non-existing server",
			serverNames: []string{"server1", "server2"},
			queryName:   "server3",
			wantError:   true,
		},
		{
			name:        "empty tracker",
			serverNames: []string{},
			queryName:   "server1",
			wantError:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tracker := NewHealthTracker(tc.serverNames)
			health, err := tracker.Status(tc.queryName)

			if tc.wantError {
				require.Error(t, err)
				require.True(t, stdErrors.Is(err, errors.ErrHealthNotTracked))
				require.Equal(t, domain.ServerHealth{}, health)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.queryName, health.Name)
				require.Equal(t, tc.wantStatus, health.Status)
			}
		})
	}
}

func TestHealthTracker_List(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		serverNames []string
		wantLen     int
	}{
		{
			name:        "empty tracker",
			serverNames: []string{},
			wantLen:     0,
		},
		{
			name:        "single server",
			serverNames: []string{"server1"},
			wantLen:     1,
		},
		{
			name:        "multiple servers",
			serverNames: []string{"server1", "server2", "server3"},
			wantLen:     3,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tracker := NewHealthTracker(tc.serverNames)
			list := tracker.List()

			require.Equal(t, tc.wantLen, len(list))

			// Verify all expected servers are in the list
			serverMap := make(map[string]bool)
			for _, health := range list {
				serverMap[health.Name] = true
			}

			for _, name := range tc.serverNames {
				require.True(t, serverMap[name], "server %s should be in the list", name)
			}
		})
	}
}

func TestHealthTracker_Update(t *testing.T) {
	t.Parallel()

	// Test basic update functionality
	t.Run("successful updates", func(t *testing.T) {
		t.Parallel()

		tracker := NewHealthTracker([]string{"server1", "server2"})
		latency := 50 * time.Millisecond

		tests := []struct {
			name         string
			serverName   string
			status       domain.HealthStatus
			latency      *time.Duration
			wantError    bool
			checkSuccess bool
		}{
			{
				name:         "update with OK status and latency",
				serverName:   "server1",
				status:       domain.HealthStatusOK,
				latency:      &latency,
				wantError:    false,
				checkSuccess: true,
			},
			{
				name:         "update with timeout status and latency",
				serverName:   "server1",
				status:       domain.HealthStatusTimeout,
				latency:      &latency,
				wantError:    false,
				checkSuccess: false,
			},
			{
				name:       "update with unreachable status and nil latency",
				serverName: "server1",
				status:     domain.HealthStatusUnreachable,
				latency:    nil,
				wantError:  false,
			},
			{
				name:       "update non-existing server",
				serverName: "server3",
				status:     domain.HealthStatusOK,
				latency:    &latency,
				wantError:  true,
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				beforeUpdate := time.Now().UTC()
				err := tracker.Update(tc.serverName, tc.status, tc.latency)

				if tc.wantError {
					require.Error(t, err)
					require.True(t, stdErrors.Is(err, errors.ErrHealthNotTracked))
					return
				}

				require.NoError(t, err)

				// Verify the update
				health, err := tracker.Status(tc.serverName)
				require.NoError(t, err)
				require.Equal(t, tc.serverName, health.Name)
				require.Equal(t, tc.status, health.Status)

				// Check LastChecked is set and recent
				require.NotNil(t, health.LastChecked)
				require.True(t, health.LastChecked.After(beforeUpdate) || health.LastChecked.Equal(beforeUpdate))
				require.True(t, health.LastChecked.Before(time.Now().UTC().Add(time.Second)))

				// Check latency
				if tc.latency != nil {
					require.NotNil(t, health.Latency)
					require.Equal(t, *tc.latency, *health.Latency)
				} else {
					require.Nil(t, health.Latency)
				}

				// Check LastSuccessful
				if tc.checkSuccess {
					require.NotNil(t, health.LastSuccessful)
					require.True(
						t,
						health.LastSuccessful.After(beforeUpdate) || health.LastSuccessful.Equal(beforeUpdate),
					)
				}
			})
		}
	})

	// Test LastSuccessful preservation
	t.Run("LastSuccessful preservation", func(t *testing.T) {
		t.Parallel()

		tracker := NewHealthTracker([]string{"server1"})
		latency := 50 * time.Millisecond

		// First update with OK status
		err := tracker.Update("server1", domain.HealthStatusOK, &latency)
		require.NoError(t, err)

		health, err := tracker.Status("server1")
		require.NoError(t, err)
		originalLastSuccessful := health.LastSuccessful
		require.NotNil(t, originalLastSuccessful)

		// Wait a bit to ensure time difference
		time.Sleep(10 * time.Millisecond)

		// Second update with non-OK status
		err = tracker.Update("server1", domain.HealthStatusTimeout, &latency)
		require.NoError(t, err)

		health, err = tracker.Status("server1")
		require.NoError(t, err)
		require.Equal(t, domain.HealthStatusTimeout, health.Status)
		require.Equal(t, originalLastSuccessful, health.LastSuccessful, "LastSuccessful should be preserved")
	})
}

func TestHealthTracker_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	tracker := NewHealthTracker([]string{"server1", "server2", "server3"})
	const numGoroutines = 100
	const numOperations = 10

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Test concurrent reads and writes
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			for j := 0; j < numOperations; j++ {
				serverName := fmt.Sprintf("server%d", (id%3)+1)
				latency := time.Duration(id*j) * time.Millisecond

				// Alternate between different operations
				switch j % 3 {
				case 0:
					err := tracker.Update(serverName, domain.HealthStatusOK, &latency)
					require.NoError(t, err)
				case 1:
					_, err := tracker.Status(serverName)
					require.NoError(t, err)
				case 2:
					list := tracker.List()
					require.GreaterOrEqual(t, len(list), 1)
				}
			}
		}(i)
	}

	wg.Wait()
}

func TestServerHealth_JSONSerialization(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	latency := 100 * time.Millisecond

	tests := []struct {
		name   string
		health domain.ServerHealth
	}{
		{
			name: "complete health record",
			health: domain.ServerHealth{
				Name:           "server1",
				Status:         domain.HealthStatusOK,
				Latency:        &latency,
				LastChecked:    &now,
				LastSuccessful: &now,
			},
		},
		{
			name: "minimal health record",
			health: domain.ServerHealth{
				Name:   "server2",
				Status: domain.HealthStatusUnknown,
			},
		},
		{
			name: "health record with nil latency",
			health: domain.ServerHealth{
				Name:        "server3",
				Status:      domain.HealthStatusTimeout,
				Latency:     nil,
				LastChecked: &now,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			res, err := api.DomainServerHealth(tc.health).ToAPIType()
			require.NoError(t, err)

			// Test marshaling only - unmarshalling requires UnmarshalJSON method
			data, err := json.Marshal(res)
			require.NoError(t, err)
			require.NotEmpty(t, data)

			// Verify it's valid JSON
			require.True(t, json.Valid(data))

			// Verify JSON structure contains expected fields
			var jsonMap map[string]any
			err = json.Unmarshal(data, &jsonMap)
			require.NoError(t, err)

			// Check required fields are present
			require.Contains(t, jsonMap, "name")
			require.Contains(t, jsonMap, "status")

			if tc.health.Latency != nil {
				require.Contains(t, jsonMap, "latency")
			} else {
				require.NotContains(t, jsonMap, "latency")
			}

			if tc.health.LastChecked != nil {
				require.Contains(t, jsonMap, "lastChecked")
			} else {
				require.NotContains(t, jsonMap, "lastChecked")
			}

			if tc.health.LastSuccessful != nil {
				require.Contains(t, jsonMap, "lastSuccessful")
			} else {
				require.NotContains(t, jsonMap, "lastSuccessful")
			}

			// Verify field values
			require.Equal(t, tc.health.Name, jsonMap["name"])
			require.Equal(t, string(tc.health.Status), jsonMap["status"])

			// Check latency field handling
			if tc.health.Latency != nil {
				require.NotEqual(t, nil, jsonMap["latency"])
				require.IsType(t, "", jsonMap["latency"])
				require.Equal(t, jsonMap["latency"].(string), "100ms")
			} else {
				require.Nil(t, jsonMap["latency"])
			}

			// Check timestamp fields
			if tc.health.LastChecked != nil {
				require.NotNil(t, jsonMap["lastChecked"])
				require.IsType(t, "", jsonMap["lastChecked"]) // Should be a string (RFC3339)
			} else {
				require.Nil(t, jsonMap["lastChecked"])
			}

			if tc.health.LastSuccessful != nil {
				require.NotNil(t, jsonMap["lastSuccessful"])
				require.IsType(t, "", jsonMap["lastSuccessful"]) // Should be a string (RFC3339)
			} else {
				require.Nil(t, jsonMap["lastSuccessful"])
			}
		})
	}
}

func TestHealthStatus_Constants(t *testing.T) {
	t.Parallel()

	// Test that all health status constants are defined correctly
	require.Equal(t, domain.HealthStatus("ok"), domain.HealthStatusOK)
	require.Equal(t, domain.HealthStatus("timeout"), domain.HealthStatusTimeout)
	require.Equal(t, domain.HealthStatus("unreachable"), domain.HealthStatusUnreachable)
	require.Equal(t, domain.HealthStatus("unknown"), domain.HealthStatusUnknown)
}
