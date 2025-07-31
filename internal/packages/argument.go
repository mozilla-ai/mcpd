package packages

import (
	"maps"
	"regexp"
	"slices"
)

const (
	// VariableTypeEnv represents an environment variable.
	VariableTypeEnv VariableType = "environment"

	// VariableTypeArg represents a command line argument which requires a value.
	VariableTypeArg VariableType = "argument"

	// VariableTypeArgBool represents a command line argument that is a boolean flag (doesn't have a value).
	VariableTypeArgBool VariableType = "argument_bool"
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

// FilterBy allows filtering of Arguments using predicates.
// All predicates must be true in order for an argument to be included in the results.
func (a Arguments) FilterBy(predicate ...func(name string, data ArgumentMetadata) bool) Arguments {
	return FilterArguments(a, predicate...)
}

// Names returns the names of the Arguments.
func (a Arguments) Names() []string {
	return slices.Collect(maps.Keys(a))
}

// FilterArguments allows Arguments to be filtered using any number of predicates.
// All predicates must be true in order for an argument to be included in the results.
func FilterArguments(args Arguments, predicate ...func(name string, data ArgumentMetadata) bool) Arguments {
	result := make(Arguments)
next:
	for name, arg := range args {
		for _, p := range predicate {
			if !p(name, arg) {
				continue next
			}
		}
		result[name] = arg
	}
	return result
}

// Required is a predicate that requires the argument is required.
func Required(_ string, data ArgumentMetadata) bool {
	return data.Required
}

// EnvVar is a predicate that requires the argument is an environment variable.
func EnvVar(_ string, data ArgumentMetadata) bool {
	return data.VariableType == VariableTypeEnv
}

// Argument is a predicate that requires the argument is a command line argument.
func Argument(s string, data ArgumentMetadata) bool {
	return ValueArgument(s, data) || BoolArgument(s, data)
}

// ValueArgument is a predicate that requires the argument is a command line argument which requires a value.
func ValueArgument(_ string, data ArgumentMetadata) bool {
	return data.VariableType == VariableTypeArg
}

// BoolArgument is a predicate that requires the argument is a command line argument which is a boolean flag.
func BoolArgument(_ string, data ArgumentMetadata) bool {
	return data.VariableType == VariableTypeArgBool
}
