package domain

import (
	"strings"
)

// ServerName is an MCP server's name as the operator configured it.
// Two names that differ only by case or by surrounding whitespace identify one server, so a name is
// indexed and compared by Key and reported by String.
type ServerName string

// ToolName is a tool's name as the MCP server reports it.
// MCP tool names are case-sensitive, so two names that differ only by case identify two tools. A name is
// compared, reported and sent upstream by String.
type ToolName string

// Key returns the form under which the name is indexed and compared.
func (n ServerName) Key() string {
	return strings.ToLower(n.String())
}

// String returns the name as it was given, without surrounding whitespace.
func (n ServerName) String() string {
	return strings.TrimSpace(string(n))
}

// String returns the name as it was given, without surrounding whitespace.
func (n ToolName) String() string {
	return strings.TrimSpace(string(n))
}
