package daemon

import (
	"encoding/json"
	stdErrors "errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/mozilla-ai/mcpd/internal/api"
	"github.com/mozilla-ai/mcpd/internal/domain"
	"github.com/mozilla-ai/mcpd/internal/errors"
)

func TestHealthTracker_Add(t *testing.T) {
	t.Parallel()

	t.Run("add new server", func(t *testing.T) {
		t.Parallel()
		tracker := NewHealthTracker([]string{"existing"})

		// Add a new server.
		tracker.Add("new-server")

		// Verify it was added.
		health, err := tracker.Status("new-server")
		require.NoError(t, err)
		require.Equal(t, "new-server", health.Name)
		require.Equal(t, domain.HealthStatusUnknown, health.Status)
		require.Nil(t, health.LastChecked)
		require.Nil(t, health.LastSuccessful)
	})

	t.Run("add existing server (no-op)", func(t *testing.T) {
		t.Parallel()
		tracker := NewHealthTracker([]string{"server1"})

		// Update the server's health.
		latency := 100 * time.Millisecond
		err := tracker.Update("server1", domain.HealthStatusOK, &latency)
		require.NoError(t, err)

		// Get the current state.
		healthBefore, err := tracker.Status("server1")
		require.NoError(t, err)

		// Try to add the same server again.
		tracker.Add("server1")

		// Verify the health data is preserved.
		healthAfter, err := tracker.Status("server1")
		require.NoError(t, err)
		require.Equal(t, healthBefore, healthAfter)
		require.NotNil(t, healthAfter.LastChecked)
		require.NotNil(t, healthAfter.LastSuccessful)
	})

	t.Run("concurrent adds", func(t *testing.T) {
		t.Parallel()
		tracker := NewHealthTracker([]string{})

		var wg sync.WaitGroup
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				tracker.Add(fmt.Sprintf("server-%d", i))
			}(i)
		}
		wg.Wait()

		// Verify all servers were added.
		servers := tracker.List()
		require.Len(t, servers, 10)
	})
}

func TestHealthTracker_Remove(t *testing.T) {
	t.Parallel()

	t.Run("remove existing server", func(t *testing.T) {
		t.Parallel()
		tracker := NewHealthTracker([]string{"server1", "server2", "server3"})

		// Remove a server.
		tracker.Remove("server2")

		// Verify it was removed.
		_, err := tracker.Status("server2")
		require.Error(t, err)
		require.True(t, stdErrors.Is(err, errors.ErrHealthNotTracked))

		// Verify other servers remain.
		servers := tracker.List()
		require.Len(t, servers, 2)
		serverNames := make([]string, len(servers))
		for i, s := range servers {
			serverNames[i] = s.Name
		}
		require.Contains(t, serverNames, "server1")
		require.Contains(t, serverNames, "server3")
		require.NotContains(t, serverNames, "server2")
	})

	t.Run("remove non-existent server (no-op)", func(t *testing.T) {
		t.Parallel()
		tracker := NewHealthTracker([]string{"server1"})

		// Remove a non-existent server.
		tracker.Remove("non-existent")

		// Verify existing server remains.
		servers := tracker.List()
		require.Len(t, servers, 1)
		require.Equal(t, "server1", servers[0].Name)
	})

	t.Run("concurrent removes", func(t *testing.T) {
		t.Parallel()
		serverNames := make([]string, 20)
		for i := 0; i < 20; i++ {
			serverNames[i] = fmt.Sprintf("server-%d", i)
		}
		tracker := NewHealthTracker(serverNames)

		var wg sync.WaitGroup
		// Remove even-numbered servers.
		for i := 0; i < 20; i += 2 {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				tracker.Remove(fmt.Sprintf("server-%d", i))
			}(i)
		}
		wg.Wait()

		// Verify only odd-numbered servers remain.
		servers := tracker.List()
		require.Len(t, servers, 10)
		for _, s := range servers {
			var num int
			_, err := fmt.Sscanf(s.Name, "server-%d", &num)
			require.NoError(t, err)
			require.Equal(t, 1, num%2, "Expected only odd-numbered servers")
		}
	})
}

func TestHealthTracker_AddRemoveIntegration(t *testing.T) {
	t.Parallel()

	tracker := NewHealthTracker([]string{"initial"})

	// Add servers.
	tracker.Add("server1")
	tracker.Add("server2")

	// Update health for server1.
	latency := 50 * time.Millisecond
	err := tracker.Update("server1", domain.HealthStatusOK, &latency)
	require.NoError(t, err)

	// Remove initial server.
	tracker.Remove("initial")

	// Add another server.
	tracker.Add("server3")

	// Remove server2.
	tracker.Remove("server2")

	// Verify final state.
	servers := tracker.List()
	require.Len(t, servers, 2)

	serverMap := make(map[string]domain.ServerHealth)
	for _, s := range servers {
		serverMap[s.Name] = s
	}

	// Verify server1 preserved its health data.
	require.Contains(t, serverMap, "server1")
	require.Equal(t, domain.HealthStatusOK, serverMap["server1"].Status)
	require.NotNil(t, serverMap["server1"].LastChecked)
	require.NotNil(t, serverMap["server1"].LastSuccessful)

	// Verify server3 is in unknown state.
	require.Contains(t, serverMap, "server3")
	require.Equal(t, domain.HealthStatusUnknown, serverMap["server3"].Status)
	require.Nil(t, serverMap["server3"].LastChecked)

	// Verify removed servers are gone.
	require.NotContains(t, serverMap, "initial")
	require.NotContains(t, serverMap, "server2")
}

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
