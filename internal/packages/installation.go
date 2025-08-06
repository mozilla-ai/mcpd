package packages

import "github.com/mozilla-ai/mcpd/v2/internal/runtime"

type Installations map[runtime.Runtime]Installation

type Installation struct {
	Command     string            `json:"command"`
	Args        []string          `json:"args"`
	Package     string            `json:"package,omitempty"`
	Env         map[string]string `json:"env,omitempty"`
	Description string            `json:"description,omitempty"`
	Recommended bool              `json:"recommended,omitempty"`
	Deprecated  bool              `json:"deprecated,omitempty"`
}

// AnyDeprecated can be used to determine if any of the installations are deprecated.
func (i Installations) AnyDeprecated() bool {
	for _, installation := range i {
		if installation.Deprecated {
			return true
		}
	}

	return false
}

// AllDeprecated can be used to determine if all the installations are deprecated.
func (i Installations) AllDeprecated() bool {
	for _, installation := range i {
		if !installation.Deprecated {
			return false
		}
	}

	return true
}
