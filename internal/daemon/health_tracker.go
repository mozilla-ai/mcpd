package daemon

import (
	"fmt"
	"maps"
	"slices"
	"sync"
	"time"
)

const (
	HealthStatusOK          HealthStatus = "ok"
	HealthStatusTimeout     HealthStatus = "timeout"
	HealthStatusUnreachable HealthStatus = "unreachable"
	HealthStatusUnknown     HealthStatus = "unknown"
)

type HealthStatus string

type Duration time.Duration

type ServerHealth struct {
	Name           string       `json:"name"`
	Status         HealthStatus `json:"status"`
	Latency        *Duration    `json:"latency"`
	LastChecked    *time.Time   `json:"last_checked"`
	LastSuccessful *time.Time   `json:"last_successful"`
}

type HealthTracker struct {
	mu       sync.RWMutex
	statuses map[string]ServerHealth
}

func (d *Duration) MarshalJSON() ([]byte, error) {
	if d == nil {
		return []byte("null"), nil
	}
	s := fmt.Sprintf(`"%s"`, time.Duration(*d).String())
	return []byte(s), nil
}

func NewHealthTracker(serverNames []string) *HealthTracker {
	statuses := make(map[string]ServerHealth, len(serverNames))
	for _, name := range serverNames {
		statuses[name] = ServerHealth{Name: name, Status: HealthStatusUnknown}
	}
	return &HealthTracker{
		statuses: statuses,
	}
}

// Status returns the health status for a single tracked server.
func (h *HealthTracker) Status(name string) (ServerHealth, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if health, ok := h.statuses[name]; ok {
		return health, nil
	}

	return ServerHealth{}, fmt.Errorf("%w: %s", ErrHealthNotTracked, name)
}

// List returns a copy of all known server health records.
func (h *HealthTracker) List() []ServerHealth {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return slices.Collect(maps.Values(h.statuses))
}

// Update records a health check for a tracked server.
// The current time is recorded as LastChecked, and LastSuccessful is updated only if status is HealthStatusOK.
// Latency can be nil if the ping failed or was not measured.
func (h *HealthTracker) Update(name string, status HealthStatus, latency *time.Duration) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	now := time.Now().UTC()

	prev, exists := h.statuses[name]
	if !exists {
		return fmt.Errorf("%w: %s", ErrHealthNotTracked, name)
	}

	var lastSuccessful *time.Time
	if status == HealthStatusOK {
		lastSuccessful = &now
	} else {
		lastSuccessful = prev.LastSuccessful
	}

	var duration *Duration
	if latency != nil {
		d := Duration(*latency)
		duration = &d
	}

	h.statuses[name] = ServerHealth{
		Name:           name,
		Status:         status,
		Latency:        duration,
		LastChecked:    &now,
		LastSuccessful: lastSuccessful,
	}

	return nil
}
