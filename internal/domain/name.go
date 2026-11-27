package domain

import (
	"strings"

	"github.com/mozilla-ai/mcpd/internal/filter"
)

// ServerName is an MCP server's name as the operator configured it.
// Two names that differ only by case or by surrounding whitespace identify one server, so a name is
// indexed and compared by Key and reported by String.
type ServerName string

// ToolName is a tool's name as the MCP server reports it.
// Matching a name against a configured allow list ignores case, so a name is compared by Key. MCP tool
// names are opaque and a server may be case sensitive, so a name is reported and sent upstream by String.
type ToolName string

// Key returns the form under which the name is indexed and compared.
func (n ServerName) Key() string {
	return filter.NormalizeString(string(n))
}

// String returns the name as it was given, without surrounding whitespace.
func (n ServerName) String() string {
	return strings.TrimSpace(string(n))
}

// Key returns the form under which the name is compared.
func (n ToolName) Key() string {
	return filter.NormalizeString(string(n))
}

// String returns the name as it was given, without surrounding whitespace.
func (n ToolName) String() string {
	return strings.TrimSpace(string(n))
}
