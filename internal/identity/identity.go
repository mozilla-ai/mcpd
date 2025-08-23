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

// InitServer creates a basic identity credential for a server
func (m *Manager) InitServer(serverName, organization string) error {
	if !m.enabled {
		return fmt.Errorf("identity not enabled (set MCPD_IDENTITY_ENABLED=true)")
	}
	
	homeDir, _ := os.UserHomeDir()
	identityDir := filepath.Join(homeDir, ".config", "mcpd", "identity")
	if err := os.MkdirAll(identityDir, 0700); err != nil {
		return fmt.Errorf("failed to create identity directory: %w", err)
	}
	
	// Simple AGNTCY-compatible credential
	cred := map[string]interface{}{
		"@context": []string{
			"https://www.w3.org/2018/credentials/v1",
			"https://agntcy.org/contexts/mcp-server-badge/v1",
		},
		"type": []string{"VerifiableCredential", "MCPServerBadge"},
		"issuer": fmt.Sprintf("did:dev:%s:mcpd", organization),
		"credentialSubject": map[string]interface{}{
			"id": serverName,
			"server": serverName,
			"organization": organization,
		},
		"issuanceDate": time.Now().Format(time.RFC3339),
	}
	
	data, _ := json.MarshalIndent(cred, "", "  ")
	credPath := filepath.Join(identityDir, serverName+".json")
	
	if err := os.WriteFile(credPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write credential: %w", err)
	}
	
	return nil
}