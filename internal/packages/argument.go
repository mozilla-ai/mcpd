package packages

import (
	"regexp"
)

const (
	// VariableTypeEnv represents an environment variable.
	VariableTypeEnv = "environment"

	// VariableTypeArg represents a command line argument.
	VariableTypeArg = "argument"
)

// EnvVarPlaceholderRegex is used to find environment variable placeholders like ${VAR_NAME}.
var EnvVarPlaceholderRegex = regexp.MustCompile(`\$\{(\w+)}`)

// VariableType represents the type of variable an MCP server package can utilize.
type VariableType string

type Arguments map[string]ArgumentMetadata

// ArgumentMetadata represents metadata about an argument/variable
type ArgumentMetadata struct {
	Description  string       `json:"description"`
	Required     bool         `json:"required"`
	VariableType VariableType `json:"type"`
}

func (a *Arguments) byVariableType(vt VariableType) map[string]ArgumentMetadata {
	args := *a
	result := make(map[string]ArgumentMetadata, len(args))
	for name, arg := range args {
		if arg.VariableType == vt {
			result[name] = arg
		}
	}
	return result
}

func (a *Arguments) EnvVars() Arguments {
	return a.byVariableType(VariableTypeEnv)
}

func (a *Arguments) EnvVarNames() []string {
	var ns []string
	for k, v := range *a {
		if v.IsEnvironmentVariable() {
			ns = append(ns, k)
		}
	}
	return ns
}

func (a *Arguments) Args() Arguments {
	return a.byVariableType(VariableTypeArg)
}

func (a *Arguments) ArgNames() []string {
	var ns []string
	for k, v := range *a {
		if v.IsCommandLineArgument() {
			ns = append(ns, k)
		}
	}
	return ns
}

// IsEnvironmentVariable returns true if this argument is primarily used as an environment variable
func (am ArgumentMetadata) IsEnvironmentVariable() bool {
	return am.VariableType == VariableTypeEnv
}

// IsCommandLineArgument returns true if this argument is primarily used as a command line argument
func (am ArgumentMetadata) IsCommandLineArgument() bool {
	return am.VariableType == VariableTypeArg
}
