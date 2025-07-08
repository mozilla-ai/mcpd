package daemon

import (
	"fmt"
	"maps"
	"slices"
	"sync"
	"time"

	"github.com/mozilla-ai/mcpd/v2/internal/domain"
	"github.com/mozilla-ai/mcpd/v2/internal/errors"
)

// HealthTracker is used to track the health of registered MCP servers.
// NewHealthTracker should be used to initialize this type.
type HealthTracker struct {
	mu       sync.RWMutex
	statuses map[string]domain.ServerHealth
}

// NewHealthTracker creates a HealthTracker which tracks the specified MCP server names.
func NewHealthTracker(serverNames []string) *HealthTracker {
	statuses := make(map[string]domain.ServerHealth, len(serverNames))
	for _, name := range serverNames {
		statuses[name] = domain.ServerHealth{Name: name, Status: domain.HealthStatusUnknown}
	}
	return &HealthTracker{
		statuses: statuses,
	}
}

// Status returns the health status for a single tracked server.
func (h *HealthTracker) Status(name string) (domain.ServerHealth, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if health, ok := h.statuses[name]; ok {
		return health, nil
	}

	return domain.ServerHealth{}, fmt.Errorf("%w: %s", errors.ErrHealthNotTracked, name)
}

// List returns a copy of all known server health records.
func (h *HealthTracker) List() []domain.ServerHealth {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return slices.Collect(maps.Values(h.statuses))
}

// Update records a health check for a tracked server.
// The current time is recorded as LastChecked, and LastSuccessful is updated only if status is HealthStatusOK.
// Latency can be nil if the ping failed or was not measured.
func (h *HealthTracker) Update(name string, status domain.HealthStatus, latency *time.Duration) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	now := time.Now().UTC()

	prev, exists := h.statuses[name]
	if !exists {
		return fmt.Errorf("%w: %s", errors.ErrHealthNotTracked, name)
	}

	var lastSuccessful *time.Time
	if status == domain.HealthStatusOK {
		lastSuccessful = &now
	} else {
		lastSuccessful = prev.LastSuccessful
	}

	h.statuses[name] = domain.ServerHealth{
		Name:           name,
		Status:         status,
		Latency:        latency,
		LastChecked:    &now,
		LastSuccessful: lastSuccessful,
	}

	return nil
}
