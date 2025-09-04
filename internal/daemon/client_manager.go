package daemon

import (
	"fmt"
	"sync"

	"github.com/mark3labs/mcp-go/client"

	"github.com/mozilla-ai/mcpd/v2/internal/filter"
)

// ClientManager holds active client connections and their associated tool lists.
// It is safe for concurrent use by multiple goroutines.
// NewClientManager should be used to create instances of ClientManager.
type ClientManager struct {
	mu          sync.RWMutex
	clients     map[string]client.MCPClient
	serverTools map[string][]string
}

// NewClientManager creates an empty, concurrency-safe ClientManager.
func NewClientManager() *ClientManager {
	return &ClientManager{
		clients:     make(map[string]client.MCPClient),
		serverTools: make(map[string][]string),
	}
}

// Add registers a client and its tools by server name.
// The server name and tool names are normalized (lowercase, trimmed) for consistent lookups.
// This method is safe for concurrent use.
func (cm *ClientManager) Add(name string, c client.MCPClient, tools []string) {
	name = filter.NormalizeString(name)
	tools = filter.NormalizeSlice(tools)
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.clients[name] = c
	cm.serverTools[name] = tools
}

// Client returns the client for the given server name.
// The server name is normalized for case-insensitive lookup.
// It returns a boolean to indicate whether the client was found.
// This method is safe for concurrent use.
func (cm *ClientManager) Client(name string) (client.MCPClient, bool) {
	name = filter.NormalizeString(name)
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	c, ok := cm.clients[name]
	return c, ok
}

// Tools returns the tools for the given server name.
// The server name is normalized for case-insensitive lookup.
// It returns a boolean to indicate whether the tools were found.
// This method is safe for concurrent use.
func (cm *ClientManager) Tools(name string) ([]string, bool) {
	name = filter.NormalizeString(name)
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	t, ok := cm.serverTools[name]
	return t, ok
}

// List returns all known server names.
// This method is safe for concurrent use.
func (cm *ClientManager) List() []string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	names := make([]string, 0, len(cm.clients))
	for name := range cm.clients {
		names = append(names, name)
	}
	return names
}

// UpdateTools updates the tools list for an existing server without restarting the client.
// The server name and tool names are normalized for consistent lookups.
// Returns an error if the server is not found.
// This method is safe for concurrent use.
func (cm *ClientManager) UpdateTools(name string, tools []string) error {
	name = filter.NormalizeString(name)
	tools = filter.NormalizeSlice(tools)
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Check if the server exists.
	if _, ok := cm.clients[name]; !ok {
		return fmt.Errorf("server '%s' not found", name)
	}

	// Update the tools list.
	cm.serverTools[name] = tools
	return nil
}

// Remove deletes the client and its tools by server name.
// The server name is normalized for case-insensitive lookup.
// This method is safe for concurrent use.
func (cm *ClientManager) Remove(name string) {
	name = filter.NormalizeString(name)
	cm.mu.Lock()
	defer cm.mu.Unlock()
	delete(cm.clients, name)
	delete(cm.serverTools, name)
}
