package daemon

import (
	"fmt"
	"strings"
	"sync"

	"github.com/mark3labs/mcp-go/client"

	"github.com/mozilla-ai/mcpd/internal/filter"
)

// ClientManager holds active client connections and their associated tool lists.
// It is safe for concurrent use by multiple goroutines.
// NewClientManager should be used to create instances of ClientManager.
type ClientManager struct {
	mu sync.RWMutex

	// servers is keyed by the normalized server name, so a lookup is case insensitive.
	servers map[string]serverEntry
}

// serverEntry is what ClientManager knows about one server.
type serverEntry struct {
	client client.MCPClient

	// name is the server name as configured, which is the form reported to a client.
	name string

	// tools are the allowed tool names in normalized form, for membership tests.
	tools []string
}

// NewClientManager creates an empty, concurrency-safe ClientManager.
func NewClientManager() *ClientManager {
	return &ClientManager{
		servers: make(map[string]serverEntry),
	}
}

// Add registers a client and its tools under the given server name.
// The name is stored as configured and indexed in normalized form, so a lookup is case insensitive while
// the configured name is what gets reported.
// Tool names are normalized, because they are only ever compared.
// This method is safe for concurrent use.
func (cm *ClientManager) Add(name string, c client.MCPClient, tools []string) {
	entry := serverEntry{
		client: c,
		name:   strings.TrimSpace(name),
		tools:  filter.NormalizeSlice(tools),
	}

	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.servers[filter.NormalizeString(name)] = entry
}

// Client returns the client for the given server name.
// The server name is normalized for case-insensitive lookup.
// It returns a boolean to indicate whether the client was found.
// This method is safe for concurrent use.
func (cm *ClientManager) Client(name string) (client.MCPClient, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	entry, ok := cm.servers[filter.NormalizeString(name)]

	return entry.client, ok
}

// List returns all known server names, each as it was configured.
// This method is safe for concurrent use.
func (cm *ClientManager) List() []string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	names := make([]string, 0, len(cm.servers))
	for _, entry := range cm.servers {
		names = append(names, entry.name)
	}

	return names
}

// Remove deletes the client and its tools by server name.
// The server name is normalized for case-insensitive lookup.
// This method is safe for concurrent use.
func (cm *ClientManager) Remove(name string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	delete(cm.servers, filter.NormalizeString(name))
}

// Tools returns the allowed tool names for the given server, in normalized form.
// The server name is normalized for case-insensitive lookup.
// It returns a boolean to indicate whether the tools were found.
// This method is safe for concurrent use.
//
// NOTE: The names are normalized, so a caller must compare rather than report them.
func (cm *ClientManager) Tools(name string) ([]string, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	entry, ok := cm.servers[filter.NormalizeString(name)]

	return entry.tools, ok
}

// UpdateTools updates the tools list for an existing server without restarting the client.
// The server name and tool names are normalized for consistent lookups.
// Returns an error if the server is not found.
// This method is safe for concurrent use.
func (cm *ClientManager) UpdateTools(name string, tools []string) error {
	key := filter.NormalizeString(name)

	cm.mu.Lock()
	defer cm.mu.Unlock()

	entry, ok := cm.servers[key]
	if !ok {
		return fmt.Errorf("server '%s' not found", name)
	}

	entry.tools = filter.NormalizeSlice(tools)
	cm.servers[key] = entry

	return nil
}
