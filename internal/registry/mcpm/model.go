package mcpm

// ServerMap represents the root JSON object, which is a map of server IDs to ServerDetails.
type ServerMap map[string]ServerDetails

// ServerDetails represents the detailed information for a single server.
type ServerDetails struct {
	Name          string                  `json:"name"`
	DisplayName   string                  `json:"display_name"`
	Description   string                  `json:"description"`
	License       string                  `json:"license"`
	Arguments     map[string]Argument     `json:"arguments"`
	Installations map[string]Installation `json:"installations"`
	Tools         []Tool                  `json:"tools,omitempty"`
	IsOfficial    bool                    `json:"is_official"`
}

// Argument defines a command-line argument for the server.
type Argument struct {
	Description string `json:"description"`
	Required    bool   `json:"required"`
	Example     string `json:"example,omitempty"`
}

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
	Name           string         `json:"name"`
	Description    string         `json:"description"`
	InputSchema    map[string]any `json:"inputSchema"`
	RequiredInputs []string       `json:"required"`
}
