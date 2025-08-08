package daemon

import (
	"sync"

	"github.com/mark3labs/mcp-go/client"
)

// ClientManager holds active client connections and their associated tool lists.
// It is safe for concurrent use by multiple goroutines.
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
// This method is safe for concurrent use.
func (cm *ClientManager) Add(name string, c client.MCPClient, tools []string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.clients[name] = c
	cm.serverTools[name] = tools
}

// Client returns the client for the given server name.
// It returns a boolean to indicate whether the client was found.
// This method is safe for concurrent use.
func (cm *ClientManager) Client(name string) (client.MCPClient, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	c, ok := cm.clients[name]
	return c, ok
}

// Tools returns the tools for the given server name.
// It returns a boolean to indicate whether the tools were found.
// This method is safe for concurrent use.
func (cm *ClientManager) Tools(name string) ([]string, bool) {
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

// Remove deletes the client and its tools by server name.
// This method is safe for concurrent use.
func (cm *ClientManager) Remove(name string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	delete(cm.clients, name)
	delete(cm.serverTools, name)
}
