package cmd

import (
	"fmt"
	
	"github.com/hashicorp/go-hclog"
	"github.com/spf13/cobra"
	
	"github.com/mozilla-ai/mcpd/v2/internal/identity"
)

var identityCmd = &cobra.Command{
	Use:   "identity",
	Short: "Manage MCP server identities",
	Long: `Manage AGNTCY-compliant identities for MCP servers.

Identity support is optional and disabled by default. Enable with:
  export MCPD_IDENTITY_ENABLED=true

Identities follow the AGNTCY Identity specification:
  https://spec.identity.agntcy.org/docs/id/definitions`,
}

var identityInitCmd = &cobra.Command{
	Use:   "init [server-name]",
	Short: "Initialize identity for an MCP server",
	Long: `Initialize an AGNTCY-compliant identity for an MCP server.

This creates a development identity with:
  - DID format: did:agntcy:dev:{organization}:{server}
  - ResolverMetadata with assertion methods
  - Service endpoints for MCP

The identity is stored in ~/.config/mcpd/identity/{server}.json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		serverName := args[0]
		organization, _ := cmd.Flags().GetString("org")
		
		// Create logger based on verbosity
		logger := hclog.NewNullLogger()
		if verbose, _ := cmd.Flags().GetBool("verbose"); verbose {
			logger = hclog.New(&hclog.LoggerOptions{
				Name:  "identity",
				Level: hclog.Debug,
			})
		}
		
		manager := identity.NewManager(logger)
		if err := manager.InitServer(serverName, organization); err != nil {
			return err
		}
		
		fmt.Printf("Created identity for server '%s'\n", serverName)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(identityCmd)
	identityCmd.AddCommand(identityInitCmd)
	
	// Flags for identity init
	identityInitCmd.Flags().StringP("org", "o", "mcpd", "Organization name for the identity")
	identityInitCmd.Flags().BoolP("verbose", "v", false, "Enable verbose logging")
}