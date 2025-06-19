package runtime

// Runtime represents the type of runtime an MCP server can be run under.
type Runtime string

const (
	// NPX represents the 'npx' Node package runner (Node Package Execute) for NodeJS packages.
	NPX Runtime = "npx"

	// UVX represents the 'uvx' UV runner for Python packages.
	UVX Runtime = "uvx"

	Python Runtime = "python"

	Docker Runtime = "docker"

	// TODO: Add other runtimes as required...
)
