package packages

// Server represents a canonical, flattened view of a discoverable MCP Server package.
type Server struct {
	// Source is the source registry where this server data originated.
	Source string `json:"source"`

	// ID is a unique identifier for the server.
	ID string `json:"id"`

	// Name is the canonical name of the server.
	Name string `json:"name"`

	// DisplayName is a human-readable display name for the server.
	DisplayName string `json:"displayName,omitempty"`

	// Description provides a detailed description of the server's capabilities.
	Description string `json:"description"`

	// License specifies the SPDX license identifier for the server.
	License string `json:"license"`

	// Categories lists the functional categories this server belongs to.
	Categories []string `json:"categories,omitempty"`

	// Tags provides searchable keywords for the server.
	Tags []string `json:"tags,omitempty"`

	// Homepage is the URL to the server's homepage or documentation.
	Homepage string `json:"homepage,omitempty"`

	// Publisher identifies the organization or individual that published the server.
	Publisher Publisher `json:"publisher,omitempty"`

	// Tools lists all tools provided by this server.
	Tools Tools `json:"tools"`

	// Installations defines the available methods for installing and running the server.
	Installations Installations `json:"installations"`

	// Arguments specifies configurable arguments for the server.
	Arguments Arguments `json:"arguments,omitempty"`

	// IsOfficial indicates whether this is an officially supported server.
	IsOfficial bool `json:"isOfficial,omitempty"`

	// Deprecated indicates whether this server is deprecated and should not be used for new installations.
	Deprecated bool `json:"deprecated,omitempty"`

	// Meta is reserved by MCP to allow clients and servers to attach additional metadata.
	Meta map[string]any `json:"_meta,omitempty"` //nolint:tagliatelle
}
