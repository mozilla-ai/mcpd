package daemon

import (
	"fmt"
	"maps"
	"slices"
	"sync"
	"time"

	"github.com/mozilla-ai/mcpd/internal/domain"
	"github.com/mozilla-ai/mcpd/internal/errors"
)

// HealthTracker is used to track the health of registered MCP servers.
// NewHealthTracker should be used to initialize this type.
type HealthTracker struct {
	mu sync.RWMutex

	// statuses is keyed by the normalized server name, so a lookup is case insensitive.
	// Each record carries the server's configured name, which is the form reported to a client.
	statuses map[string]domain.ServerHealth
}

// NewHealthTracker creates a HealthTracker which tracks the specified MCP server names.
func NewHealthTracker(serverNames []domain.ServerName) *HealthTracker {
	statuses := make(map[string]domain.ServerHealth, len(serverNames))
	for _, name := range serverNames {
		statuses[name.Key()] = domain.ServerHealth{
			Name:   name,
			Status: domain.HealthStatusUnknown,
		}
	}
	return &HealthTracker{
		statuses: statuses,
	}
}

// Add registers a new server for health tracking, keeping the name as configured.
// If the server is already being tracked under any casing, this is a no-op.
func (h *HealthTracker) Add(name domain.ServerName) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Only add if not already tracked.
	key := name.Key()
	if _, exists := h.statuses[key]; !exists {
		h.statuses[key] = domain.ServerHealth{
			Name:   name,
			Status: domain.HealthStatusUnknown,
		}
	}
}

// List returns a copy of all known server health records.
func (h *HealthTracker) List() []domain.ServerHealth {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return slices.Collect(maps.Values(h.statuses))
}

// Remove stops tracking health for a server and removes its health data.
// The server name is normalized for case-insensitive lookup.
func (h *HealthTracker) Remove(name domain.ServerName) {
	h.mu.Lock()
	defer h.mu.Unlock()

	delete(h.statuses, name.Key())
}

// Status returns the health status for a single tracked server.
// The server name is normalized for case-insensitive lookup.
func (h *HealthTracker) Status(name domain.ServerName) (domain.ServerHealth, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if health, ok := h.statuses[name.Key()]; ok {
		return health, nil
	}

	return domain.ServerHealth{}, fmt.Errorf("%w: %s", errors.ErrHealthNotTracked, name)
}

// Update records a health check for a tracked server.
// The server name is normalized for case-insensitive lookup, and the record keeps the configured name.
// The current time is recorded as LastChecked, and LastSuccessful is updated only if status is HealthStatusOK.
// Latency can be nil if the ping failed or was not measured.
func (h *HealthTracker) Update(
	name domain.ServerName,
	status domain.HealthStatus,
	latency *time.Duration,
) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	now := time.Now().UTC()

	key := name.Key()

	prev, exists := h.statuses[key]
	if !exists {
		return fmt.Errorf("%w: %s", errors.ErrHealthNotTracked, name)
	}

	var lastSuccessful *time.Time
	if status == domain.HealthStatusOK {
		lastSuccessful = &now
	} else {
		lastSuccessful = prev.LastSuccessful
	}

	h.statuses[key] = domain.ServerHealth{
		Name:           prev.Name,
		Status:         status,
		Latency:        latency,
		LastChecked:    &now,
		LastSuccessful: lastSuccessful,
	}

	return nil
}
