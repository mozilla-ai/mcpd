package daemon

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDuration_MarshalJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		duration *Duration
		want     string
	}{
		{
			name:     "nil duration",
			duration: nil,
			want:     "null",
		},
		{
			name:     "zero duration",
			duration: func() *Duration { d := Duration(0); return &d }(),
			want:     `"0s"`,
		},
		{
			name:     "positive duration",
			duration: func() *Duration { d := Duration(100 * time.Millisecond); return &d }(),
			want:     `"100ms"`,
		},
		{
			name:     "complex duration",
			duration: func() *Duration { d := Duration(1*time.Hour + 30*time.Minute + 45*time.Second); return &d }(),
			want:     `"1h30m45s"`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := tc.duration.MarshalJSON()
			require.NoError(t, err)
			require.Equal(t, tc.want, string(got))
		})
	}
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
				require.Equal(t, HealthStatusUnknown, health.Status)
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
		wantStatus  HealthStatus
	}{
		{
			name:        "existing server",
			serverNames: []string{"server1", "server2"},
			queryName:   "server1",
			wantError:   false,
			wantStatus:  HealthStatusUnknown,
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
				require.True(t, errors.Is(err, ErrHealthNotTracked))
				require.Equal(t, ServerHealth{}, health)
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
			status       HealthStatus
			latency      *time.Duration
			wantError    bool
			checkSuccess bool
		}{
			{
				name:         "update with OK status and latency",
				serverName:   "server1",
				status:       HealthStatusOK,
				latency:      &latency,
				wantError:    false,
				checkSuccess: true,
			},
			{
				name:         "update with timeout status and latency",
				serverName:   "server1",
				status:       HealthStatusTimeout,
				latency:      &latency,
				wantError:    false,
				checkSuccess: false,
			},
			{
				name:       "update with unreachable status and nil latency",
				serverName: "server1",
				status:     HealthStatusUnreachable,
				latency:    nil,
				wantError:  false,
			},
			{
				name:       "update non-existing server",
				serverName: "server3",
				status:     HealthStatusOK,
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
					require.True(t, errors.Is(err, ErrHealthNotTracked))
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
					require.Equal(t, Duration(*tc.latency), *health.Latency)
				} else {
					require.Nil(t, health.Latency)
				}

				// Check LastSuccessful
				if tc.checkSuccess {
					require.NotNil(t, health.LastSuccessful)
					require.True(t, health.LastSuccessful.After(beforeUpdate) || health.LastSuccessful.Equal(beforeUpdate))
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
		err := tracker.Update("server1", HealthStatusOK, &latency)
		require.NoError(t, err)

		health, err := tracker.Status("server1")
		require.NoError(t, err)
		originalLastSuccessful := health.LastSuccessful
		require.NotNil(t, originalLastSuccessful)

		// Wait a bit to ensure time difference
		time.Sleep(10 * time.Millisecond)

		// Second update with non-OK status
		err = tracker.Update("server1", HealthStatusTimeout, &latency)
		require.NoError(t, err)

		health, err = tracker.Status("server1")
		require.NoError(t, err)
		require.Equal(t, HealthStatusTimeout, health.Status)
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
					err := tracker.Update(serverName, HealthStatusOK, &latency)
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
	latency := Duration(100 * time.Millisecond)

	tests := []struct {
		name   string
		health ServerHealth
	}{
		{
			name: "complete health record",
			health: ServerHealth{
				Name:           "server1",
				Status:         HealthStatusOK,
				Latency:        &latency,
				LastChecked:    &now,
				LastSuccessful: &now,
			},
		},
		{
			name: "minimal health record",
			health: ServerHealth{
				Name:   "server2",
				Status: HealthStatusUnknown,
			},
		},
		{
			name: "health record with nil latency",
			health: ServerHealth{
				Name:        "server3",
				Status:      HealthStatusTimeout,
				Latency:     nil,
				LastChecked: &now,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Test marshaling only - unmarshaling requires UnmarshalJSON method
			data, err := json.Marshal(tc.health)
			require.NoError(t, err)
			require.NotEmpty(t, data)

			// Verify it's valid JSON
			require.True(t, json.Valid(data))

			// Verify JSON structure contains expected fields
			var jsonMap map[string]interface{}
			err = json.Unmarshal(data, &jsonMap)
			require.NoError(t, err)

			// Check required fields are present
			require.Contains(t, jsonMap, "name")
			require.Contains(t, jsonMap, "status")
			require.Contains(t, jsonMap, "latency")
			require.Contains(t, jsonMap, "last_checked")
			require.Contains(t, jsonMap, "last_successful")

			// Verify field values
			require.Equal(t, tc.health.Name, jsonMap["name"])
			require.Equal(t, string(tc.health.Status), jsonMap["status"])

			// Check latency field handling
			if tc.health.Latency != nil {
				require.NotEqual(t, nil, jsonMap["latency"])
				require.IsType(t, "", jsonMap["latency"])
				require.Contains(t, jsonMap["latency"].(string), "ms") // Should contain duration unit
			} else {
				require.Nil(t, jsonMap["latency"])
			}

			// Check timestamp fields
			if tc.health.LastChecked != nil {
				require.NotNil(t, jsonMap["last_checked"])
				require.IsType(t, "", jsonMap["last_checked"]) // Should be a string (RFC3339)
			} else {
				require.Nil(t, jsonMap["last_checked"])
			}

			if tc.health.LastSuccessful != nil {
				require.NotNil(t, jsonMap["last_successful"])
				require.IsType(t, "", jsonMap["last_successful"]) // Should be a string (RFC3339)
			} else {
				require.Nil(t, jsonMap["last_successful"])
			}
		})
	}
}

func TestHealthStatus_Constants(t *testing.T) {
	t.Parallel()

	// Test that all health status constants are defined correctly
	require.Equal(t, HealthStatus("ok"), HealthStatusOK)
	require.Equal(t, HealthStatus("timeout"), HealthStatusTimeout)
	require.Equal(t, HealthStatus("unreachable"), HealthStatusUnreachable)
	require.Equal(t, HealthStatus("unknown"), HealthStatusUnknown)
}
