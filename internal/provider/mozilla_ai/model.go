package mozilla_ai

const (
	ArgumentEnv        ArgumentType = "environment"
	ArgumentValue      ArgumentType = "argument"
	ArgumentBool       ArgumentType = "argument_bool"
	ArgumentPositional ArgumentType = "argument_positional"
)

const (
	NPX Runtime = "npx"
	UVX Runtime = "uvx"
)

type ArgumentType string

type Runtime string

// MCPRegistry represents the root registry as a map of server IDs to server details.
type MCPRegistry map[string]Server

type Tools []Tool

type Arguments map[string]Argument

// Server represents a complete MCP server entry.
type Server struct {
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
	Installations map[string]Installation `json:"installations"`

	// Arguments specifies configurable arguments for the server.
	Arguments Arguments `json:"arguments,omitempty"`

	// IsOfficial indicates whether this is an officially supported server.
	IsOfficial bool `json:"isOfficial,omitempty"`

	// Deprecated indicates whether this server is deprecated and should not be used for new installations.
	Deprecated bool `json:"deprecated,omitempty"`

	// Meta is reserved by MCP to allow clients and servers to attach additional metadata.
	Meta map[string]any `json:"_meta,omitempty"` //nolint:tagliatelle
}

// Installation represents a method for installing and running an MCP server.
type Installation struct {
	// Runtime specifies the runtime type for this installation method.
	Runtime Runtime `json:"runtime"`

	// Package is the package name that will be executed.
	Package string `json:"package,omitempty"`

	// Version specifies the version for this installation method.
	Version string `json:"version"`

	// Description provides additional details about this installation method.
	Description string `json:"description,omitempty"`

	// Recommended indicates if this is the preferred installation method.
	Recommended bool `json:"recommended,omitempty"`

	// Deprecated indicates whether this installation method is deprecated.
	Deprecated bool `json:"deprecated,omitempty"`

	// Transports lists the supported transport mechanisms for this server.
	// Common transports include Stdio, SSE and Streamable HTTP.
	// If not specified, defaults to ["stdio"].
	Transports []string `json:"transports,omitempty"`

	// Repository optionally specifies a different source repository for this installation.
	Repository *Repository `json:"repository,omitempty"`
}

// Tool defines the structure for a tool that a client can call.
type Tool struct {
	// Name of the tool.
	// Display name precedence order for a tool is: title, annotations.title, then name.
	Name string `json:"name"`

	// Title is a human-readable and easily understood title for the tool.
	Title string `json:"title,omitempty"`

	// Description is a human-readable description of the tool.
	// This can be used by clients to improve the LLM's understanding of available tools.
	// It can be thought of like a "hint" to the model.
	Description string `json:"description"`

	// Annotations provide optional additional tool information.
	// Display name precedence order is: title, annotations.title when present, then tool name.
	Annotations *ToolAnnotations `json:"annotations,omitempty"`

	// Meta is reserved by MCP to allow clients and servers to attach additional metadata to their interactions.
	// See https://modelcontextprotocol.io/specification/2025-06-18/basic#general-fields for notes on _meta usage.
	Meta map[string]any `json:"_meta,omitempty"` //nolint:tagliatelle
}

// Argument represents a configurable argument for an MCP server.
type Argument struct {
	// Name is the reference for the argument.
	Name string `json:"name"`

	// Description provides a human-readable explanation of the argument's purpose.
	Description string `json:"description"`

	// Required indicates whether this argument is mandatory for server operation.
	Required bool `json:"required"`

	// Type specifies the argument type (environment variable, command-line argument, etc.).
	Type ArgumentType `json:"type"`

	// Example provides an example value for the argument.
	Example string `json:"example,omitempty"`

	// Position specifies the position for positional arguments (1-based index).
	// Only relevant when Type is ArgumentPositional.
	Position *int `json:"position,omitempty"`
}

// Repository represents a source code repository with version verification.
// When used per-installation, care should be taken to verify that different repositories
// for the same server implement identical functionality to prevent security issues.
type Repository struct {
	// Type specifies the repository type (e.g., "git", "github").
	Type string `json:"type"`

	// URL is the repository URL where the source code is hosted.
	URL string `json:"url"`

	// Commit is the specific commit hash corresponding to the version tag.
	// This provides version verification and prevents tag manipulation attacks.
	Commit string `json:"commit,omitempty"`
}

// Publisher represents the organization or individual that published the server.
type Publisher struct {
	// Name is the name of the publisher (organization or individual).
	Name string `json:"name"`

	// URL is an optional link to the publisher's website or profile.
	URL string `json:"url,omitempty"`
}

// ToolAnnotations provides additional properties describing a Tool to clients.
// NOTE: all properties in ToolAnnotations are **hints**.
// They are not guaranteed to provide a faithful description of tool behavior
// (including descriptive properties like `title`).
// Clients should never make tool use decisions based on ToolAnnotations received from untrusted servers.
type ToolAnnotations struct {
	// Title is a human-readable title for the tool.
	Title *string `json:"title,omitempty"`

	// ReadOnlyHint if true, the tool should not modify its environment.
	ReadOnlyHint *bool `json:"readOnlyHint,omitempty"`

	// DestructiveHint if true, the tool may perform destructive updates to its environment.
	// If false, the tool performs only additive updates.
	// (This property is meaningful only when ReadOnlyHint is false)
	DestructiveHint *bool `json:"destructiveHint,omitempty"`

	// IdempotentHint if true, calling the tool repeatedly with the same arguments
	// will have no additional effect on its environment.
	// (This property is meaningful only when ReadOnlyHint is false)
	IdempotentHint *bool `json:"idempotentHint,omitempty"`

	// OpenWorldHint if true, this tool may interact with an "open world" of external
	// entities. If false, the tool's domain of interaction is closed.
	// For example, the world of a web search tool is open, whereas that
	// of a memory tool is not.
	OpenWorldHint *bool `json:"openWorldHint,omitempty"`
}
