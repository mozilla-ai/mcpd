package daemon

import (
	"fmt"
	"slices"
	"sync"

	"github.com/mark3labs/mcp-go/client"

	"github.com/mozilla-ai/mcpd/internal/domain"
)

// ClientManager holds active client connections and their associated tool lists.
// It is safe for concurrent use by multiple goroutines.
// NewClientManager should be used to create instances of ClientManager.
type ClientManager struct {
	mu sync.RWMutex

	// servers is keyed by each server name's key, so a lookup does not depend on case.
	servers map[string]managedServer
}

// managedServer is what ClientManager knows about one running server.
type managedServer struct {
	client client.MCPClient

	// name is the server name as configured, which is the form reported to a client.
	name domain.ServerName

	// tools are the tool names the operator allowed for this server.
	tools []domain.ToolName
}

// NewClientManager creates an empty, concurrency-safe ClientManager.
func NewClientManager() *ClientManager {
	return &ClientManager{
		servers: make(map[string]managedServer),
	}
}

// Add registers a client and its tools under the given server name.
// The name is indexed by its key and stored as given, so a lookup does not depend on case while the
// configured name is what gets reported.
// This method is safe for concurrent use.
func (cm *ClientManager) Add(name domain.ServerName, c client.MCPClient, tools []domain.ToolName) {
	entry := managedServer{
		client: c,
		name:   name,
		tools:  slices.Clone(tools),
	}

	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.servers[name.Key()] = entry
}

// Client returns the client for the given server name.
// It returns a boolean to indicate whether the client was found.
// This method is safe for concurrent use.
func (cm *ClientManager) Client(name domain.ServerName) (client.MCPClient, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	entry, ok := cm.servers[name.Key()]

	return entry.client, ok
}

// List returns all known server names, each as it was configured.
// This method is safe for concurrent use.
func (cm *ClientManager) List() []domain.ServerName {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	names := make([]domain.ServerName, 0, len(cm.servers))
	for _, entry := range cm.servers {
		names = append(names, entry.name)
	}

	return names
}

// Remove deletes the client and its tools by server name.
// This method is safe for concurrent use.
func (cm *ClientManager) Remove(name domain.ServerName) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	delete(cm.servers, name.Key())
}

// Tools returns a copy of the allowed tool names for the given server, each as the operator configured it.
// It returns a boolean to indicate whether the tools were found.
// This method is safe for concurrent use.
func (cm *ClientManager) Tools(name domain.ServerName) ([]domain.ToolName, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	entry, ok := cm.servers[name.Key()]

	return slices.Clone(entry.tools), ok
}

// UpdateTools updates the tools list for an existing server without restarting the client.
// Returns an error if the server is not found.
// This method is safe for concurrent use.
func (cm *ClientManager) UpdateTools(name domain.ServerName, tools []domain.ToolName) error {
	key := name.Key()

	cm.mu.Lock()
	defer cm.mu.Unlock()

	entry, ok := cm.servers[key]
	if !ok {
		return fmt.Errorf("server '%s' not found", name)
	}

	entry.tools = slices.Clone(tools)
	cm.servers[key] = entry

	return nil
}
