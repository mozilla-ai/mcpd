package packages

import (
	"cmp"
	"maps"
	"regexp"
	"slices"
	"strings"
)

const (
	// VariableTypeEnv represents an environment variable.
	VariableTypeEnv VariableType = "environment"

	// VariableTypeArg represents a command line argument which requires a value.
	VariableTypeArg VariableType = "argument"

	// VariableTypeArgBool represents a command line argument that is a boolean flag (doesn't have a value).
	VariableTypeArgBool VariableType = "argument_bool"

	// VariableTypePositionalArg represents a positional command line argument.
	VariableTypePositionalArg VariableType = "positional_argument"
)

// EnvVarPlaceholderRegex is used to find environment variable placeholders like ${VAR_NAME}.
var EnvVarPlaceholderRegex = regexp.MustCompile(`\$\{(\w+)}`)

// VariableType represents the type of variable an MCP server package can utilize.
type VariableType string

type Arguments map[string]ArgumentMetadata

// ArgumentMetadata represents metadata about an argument/variable
type ArgumentMetadata struct {
	Name         string       `json:"name"`
	Description  string       `json:"description"`
	Required     bool         `json:"required"`
	Example      string       `json:"example"`
	VariableType VariableType `json:"type"`
	Position     *int         `json:"position,omitempty"` // Position in args array for positional arguments
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

// Ordered returns all arguments with positional arguments first (in position order),
// followed by all other arguments in alphabetical order by name.
func (a Arguments) Ordered() []ArgumentMetadata {
	var positional []ArgumentMetadata
	var others []ArgumentMetadata

	for name, meta := range a {
		// Ensure name is set in the metadata
		meta.Name = name

		if meta.VariableType == VariableTypePositionalArg && meta.Position != nil {
			positional = append(positional, meta)
		} else {
			others = append(others, meta)
		}
	}

	// Sort positional by position
	slices.SortFunc(positional, func(a, b ArgumentMetadata) int {
		return *a.Position - *b.Position
	})

	// Sort others alphabetically by name
	slices.SortFunc(others, func(a, b ArgumentMetadata) int {
		return cmp.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	})

	// Combine: positional first, then others
	result := make([]ArgumentMetadata, 0, len(positional)+len(others))
	result = append(result, positional...)
	result = append(result, others...)

	return result
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
	return ValueArgument(s, data) || BoolArgument(s, data) || PositionalArgument(s, data)
}

// ValueArgument is a predicate that requires the argument is a command line argument which requires a value.
func ValueArgument(_ string, data ArgumentMetadata) bool {
	return data.VariableType == VariableTypeArg
}

// BoolArgument is a predicate that requires the argument is a command line argument which is a boolean flag.
func BoolArgument(_ string, data ArgumentMetadata) bool {
	return data.VariableType == VariableTypeArgBool
}

// PositionalArgument is a predicate that requires the argument is a positional command line argument.
func PositionalArgument(_ string, data ArgumentMetadata) bool {
	return data.VariableType == VariableTypePositionalArg
}

// NonPositionalArgument is a predicate that requires the argument is not a positional command line argument.
func NonPositionalArgument(s string, data ArgumentMetadata) bool {
	return !PositionalArgument(s, data)
}
