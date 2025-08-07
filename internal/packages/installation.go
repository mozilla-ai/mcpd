package packages

import "github.com/mozilla-ai/mcpd/v2/internal/runtime"

type Installations map[runtime.Runtime]Installation

type Installation struct {
	// Runtime specifies the runtime type for this installation method.
	Runtime runtime.Runtime `json:"runtime"`

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
