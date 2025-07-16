package packages

import "github.com/mozilla-ai/mcpd/v2/internal/runtime"

// Package represents a canonical, flattened view of a discoverable MCP Server package.
type Package struct {
	ID                  string                           `json:"id"`
	Name                string                           `json:"name"`
	DisplayName         string                           `json:"display_name"`
	Description         string                           `json:"description"`
	License             string                           `json:"license"`
	Tools               Tools                            `json:"tools"`
	Tags                []string                         `json:"tags"`
	Categories          []string                         `json:"categories"`
	Runtimes            []runtime.Runtime                `json:"runtimes"`
	InstallationDetails map[runtime.Runtime]Installation `json:"installations"`
	Arguments           Arguments                        `json:"arguments"`
	Source              string                           `json:"source"`
	Version             string                           `json:"version"`
	Transport           string                           `json:"transport"`   // TODO: Default to stdio.
	IsOfficial          bool                             `json:"is_official"` // TODO: Not all registries support this.
}
