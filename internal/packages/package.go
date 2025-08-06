package packages

// Package represents a canonical, flattened view of a discoverable MCP Server package.
type Package struct {
	ID            string        `json:"id"`
	Name          string        `json:"name"`
	DisplayName   string        `json:"displayName"`
	Description   string        `json:"description"`
	License       string        `json:"license"`
	Tools         Tools         `json:"tools"`
	Tags          []string      `json:"tags"`
	Categories    []string      `json:"categories"`
	Installations Installations `json:"installations"`
	Arguments     Arguments     `json:"arguments"`
	Source        string        `json:"source"`
	Transports    []Transport   `json:"transports"`
	IsOfficial    bool          `json:"isOfficial"`
	Deprecated    bool          `json:"deprecated"`
}
