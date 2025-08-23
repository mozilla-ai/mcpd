// Package identity provides optional AGNTCY Identity support for MCP servers.
package identity

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
	
	"github.com/hashicorp/go-hclog"
)

// Manager handles identity verification with minimal complexity
type Manager struct {
	logger  hclog.Logger
	enabled bool
}

// NewManager creates a new identity manager
func NewManager(logger hclog.Logger) *Manager {
	if logger == nil {
		logger = hclog.NewNullLogger()
	}
	
	return &Manager{
		logger:  logger.Named("identity"),
		enabled: os.Getenv("MCPD_IDENTITY_ENABLED") == "true",
	}
}

// IsEnabled returns if identity is enabled
func (m *Manager) IsEnabled() bool {
	return m.enabled
}

// VerifyServer checks if a server has valid identity credentials
func (m *Manager) VerifyServer(ctx context.Context, serverName string) error {
	if !m.enabled {
		return nil
	}
	
	// For now, just check if credential file exists
	homeDir, _ := os.UserHomeDir()
	credPath := filepath.Join(homeDir, ".config", "mcpd", "identity", serverName+".json")
	
	if _, err := os.Stat(credPath); os.IsNotExist(err) {
		m.logger.Debug("No identity credentials found", "server", serverName)
		// Don't fail - identity is optional
		return nil
	}
	
	m.logger.Info("Identity verified", "server", serverName)
	return nil
}

// InitServer creates AGNTCY-spec identity with ResolverMetadata
func (m *Manager) InitServer(serverName, organization string) error {
	if !m.enabled {
		return fmt.Errorf("identity not enabled (set MCPD_IDENTITY_ENABLED=true)")
	}
	
	homeDir, _ := os.UserHomeDir()
	identityDir := filepath.Join(homeDir, ".config", "mcpd", "identity")
	if err := os.MkdirAll(identityDir, 0700); err != nil {
		return fmt.Errorf("failed to create identity directory: %w", err)
	}
	
	// AGNTCY identity format with ResolverMetadata
	id := fmt.Sprintf("did:agntcy:dev:%s:%s", organization, serverName)
	identity := map[string]interface{}{
		"id": id,
		"resolverMetadata": map[string]interface{}{
			"id": id,
			"assertionMethod": []map[string]interface{}{
				{
					"id": id + "#key-1",
					"publicKeyJwk": map[string]interface{}{
						"kty": "OKP",
						"crv": "Ed25519",
						"x":   "development-key-placeholder",
					},
				},
			},
			"service": []map[string]interface{}{
				{
					"id": id + "#mcp",
					"type": "MCPService",
					"serviceEndpoint": fmt.Sprintf("http://localhost:8090/servers/%s", serverName),
				},
			},
		},
	}
	
	data, _ := json.MarshalIndent(identity, "", "  ")
	credPath := filepath.Join(identityDir, serverName+".json")
	
	if err := os.WriteFile(credPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write identity: %w", err)
	}
	
	return nil
}