package mcpm

// MCPServers represents the root JSON object, which is a map of MCP server IDs to MCPServer.
type MCPServers map[string]MCPServer

// Tools is a wrapper for a collection of Tool.
// This allows for receivers to be declared that operate on the collection.
type Tools []Tool

// MCPServer represents the detailed information for an MCP single server.
// NOTE: Based on mcpm server schema: https://github.com/pathintegral-institute/mcpm.sh/blob/8edbd723cf3c35433739afb27a723fdcdf763c23/mcp-registry/schema/server-schema.json
type MCPServer struct {
	Name          string        `json:"name"`
	DisplayName   string        `json:"display_name"`
	Description   string        `json:"description,omitempty"`
	License       string        `json:"license"`
	Arguments     Arguments     `json:"arguments"`
	Installations Installations `json:"installations"`
	Tools         Tools         `json:"tools,omitempty"`
	IsOfficial    bool          `json:"is_official"`
	Repository    Repository    `json:"repository,omitempty"`
	Homepage      string        `json:"homepage,omitempty"`
	Author        Author        `json:"author,omitempty"`
	Tags          []string      `json:"tags,omitempty"`
	Categories    []string      `json:"categories,omitempty"`
	Examples      []Example     `json:"examples,omitempty"`
}

type Arguments map[string]Argument

// Argument defines a command-line argument for the server.
type Argument struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
	Example     string `json:"example,omitempty"`
}

type Installations map[string]Installation

// Installation defines a method for installing and running the server.
type Installation struct {
	Type        string            `json:"type"`
	Command     string            `json:"command"`
	Args        []string          `json:"args"`
	Package     string            `json:"package,omitempty"`
	Env         map[string]string `json:"env,omitempty"`
	Description string            `json:"description,omitempty"`
	Recommended bool              `json:"recommended,omitempty"`
}

// Tool defines a specific function or capability exposed by the server.
// This struct is used for tools that have detailed schema (e.g., in other registries),
// but the MCPM 'tools' field itself is a list of strings.
type Tool struct {
	Name           string     `json:"name"`
	Title          string     `json:"title"`
	Description    string     `json:"description"`
	InputSchema    JSONSchema `json:"inputSchema"`
	RequiredInputs []string   `json:"required"`
}

// JSONSchema defines the structure for a JSON schema object.
type JSONSchema struct {
	// Type defines the type for this schema, e.g. "object".
	Type string `json:"type"`

	// Properties represents a property name and associated object definition.
	Properties map[string]any `json:"properties,omitempty"`

	// Required lists the (keys of) Properties that are required.
	Required []string `json:"required,omitempty"`
}

type Repository struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

type Author struct {
	Name string `json:"name"`
}

type Example struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Prompt      string `json:"prompt"`
}
