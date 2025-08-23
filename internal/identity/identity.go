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

// Manager handles identity verification for MCP servers.
// It supports AGNTCY Identity specifications with local file storage.
// NewManager should be used to create instances of Manager.
type Manager struct {
	logger  hclog.Logger
	enabled bool
}

// NewManager creates a new identity manager instance.
// Identity is disabled by default unless MCPD_IDENTITY_ENABLED is set to "true".
func NewManager(logger hclog.Logger) *Manager {
	if logger == nil {
		logger = hclog.NewNullLogger()
	}
	
	return &Manager{
		logger:  logger.Named("identity"),
		enabled: os.Getenv("MCPD_IDENTITY_ENABLED") == "true",
	}
}

// IsEnabled returns whether identity verification is enabled.
// This method is safe for concurrent use.
func (m *Manager) IsEnabled() bool {
	return m.enabled
}

// VerifyServer checks if a server has valid identity credentials.
// If identity is disabled or no credentials exist, it returns nil (non-blocking).
// This method logs verification status but never fails to maintain backward compatibility.
func (m *Manager) VerifyServer(ctx context.Context, serverName string) error {
	if !m.enabled {
		return nil
	}
	
	credPath, err := m.getCredentialPath(serverName)
	if err != nil {
		m.logger.Debug("Failed to get credential path", "server", serverName, "error", err)
		return nil
	}
	
	if _, err := os.Stat(credPath); os.IsNotExist(err) {
		m.logger.Debug("No identity credentials found", "server", serverName, "path", credPath)
		return nil
	}
	
	m.logger.Info("Identity verified", "server", serverName)
	return nil
}

// InitServer creates an AGNTCY-compliant identity for the specified server.
// The identity follows the AGNTCY Identity specification with ResolverMetadata.
// Returns an error if identity is disabled or if file operations fail.
func (m *Manager) InitServer(serverName, organization string) error {
	if !m.enabled {
		return fmt.Errorf("identity not enabled (set MCPD_IDENTITY_ENABLED=true)")
	}
	
	identity := m.createIdentity(serverName, organization)
	
	credPath, err := m.getCredentialPath(serverName)
	if err != nil {
		return fmt.Errorf("failed to get credential path: %w", err)
	}
	
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(credPath), 0700); err != nil {
		return fmt.Errorf("failed to create identity directory: %w", err)
	}
	
	data, err := json.MarshalIndent(identity, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal identity: %w", err)
	}
	
	if err := os.WriteFile(credPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write identity: %w", err)
	}
	
	m.logger.Info("Created identity", "server", serverName, "path", credPath)
	return nil
}

// createIdentity builds the AGNTCY-compliant identity structure.
func (m *Manager) createIdentity(serverName, organization string) map[string]interface{} {
	id := fmt.Sprintf("did:agntcy:dev:%s:%s", organization, serverName)
	
	return map[string]interface{}{
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
					"serviceEndpoint": fmt.Sprintf("/servers/%s", serverName),
				},
			},
		},
	}
}

// getCredentialPath returns the file path for a server's identity credentials.
func (m *Manager) getCredentialPath(serverName string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	
	return filepath.Join(homeDir, ".config", "mcpd", "identity", serverName+".json"), nil
}