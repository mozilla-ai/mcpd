package types

import (
	"regexp"
)

type Runtime string

const (
	RuntimeNpx Runtime = "npx"
	RuntimeUvx Runtime = "uvx"
)

// PackageResult represents a canonical, flattened view of a discoverable package or server.
type PackageResult struct {
	ID                  string
	Name                string
	DisplayName         string
	Description         string
	License             string
	Tools               []string
	Runtimes            []string
	InstallationDetails map[string]Installation
	Arguments           map[string]ArgumentMetadata
	ConfigurableEnvVars []string
	Source              string // The source for this package
	Version             string // TODO: Version for this package.
}

type Installation struct {
	Args        []string          `json:"args"`
	Package     string            `json:"package,omitempty"`
	Env         map[string]string `json:"env,omitempty"`
	Description string            `json:"description,omitempty"`
	Recommended bool              `json:"recommended,omitempty"`
}

type ArgumentMetadata struct {
	Description string `json:"description"`
	Required    bool   `json:"required"`
	Example     string `json:"example,omitempty"`
}

// EnvVarPlaceholderRegex is used to find environment variable placeholders like ${VAR_NAME}.
var EnvVarPlaceholderRegex = regexp.MustCompile(`\$\{(\w+)}`)
