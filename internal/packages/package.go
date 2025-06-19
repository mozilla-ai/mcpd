package packages

import "github.com/mozilla-ai/mcpd-cli/v2/internal/runtime"

// Package represents a canonical, flattened view of a discoverable MCP Server package.
type Package struct {
	ID                  string                           `json:"id"`
	Name                string                           `json:"name"`
	DisplayName         string                           `json:"displayName"`
	Description         string                           `json:"description"`
	License             string                           `json:"license"`
	Tools               []string                         `json:"tools"`
	Runtimes            []runtime.Runtime                `json:"runtimes"`
	InstallationDetails map[runtime.Runtime]Installation `json:"installationDetails"`
	Arguments           Arguments                        `json:"arguments"`
	ConfigurableEnvVars []string                         `json:"configurableEnvVars"`
	Source              string                           `json:"source"`
	Version             string                           `json:"version"`
}
