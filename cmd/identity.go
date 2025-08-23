package cmd

import (
	"fmt"
	
	"github.com/spf13/cobra"
	
	"github.com/mozilla-ai/mcpd/v2/internal/identity"
)

var identityCmd = &cobra.Command{
	Use:   "identity",
	Short: "Manage MCP server identities (AGNTCY-compatible)",
	Long:  `Optional identity management for MCP servers using AGNTCY standards.`,
}

var identityInitCmd = &cobra.Command{
	Use:   "init [server-name]",
	Short: "Initialize identity for an MCP server",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		serverName := args[0]
		organization, _ := cmd.Flags().GetString("org")
		
		manager := identity.NewManager(nil)
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
	
	identityInitCmd.Flags().StringP("org", "o", "mcpd", "Organization name")
}