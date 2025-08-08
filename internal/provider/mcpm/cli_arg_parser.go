package mcpm

import (
	"github.com/mozilla-ai/mcpd/v2/internal/packages"
	"github.com/mozilla-ai/mcpd/v2/internal/runtime"
)

// CLIArgParser encapsulates the state and logic for parsing CLI arguments.
type CLIArgParser struct {
	schema          Arguments
	spec            runtime.Spec
	result          map[string]packages.ArgumentMetadata
	positionalCount int // Counter for positional arguments
}

// NewCLIArgParser creates a new CLI argument parser
func NewCLIArgParser(schema Arguments, spec runtime.Spec) *CLIArgParser {
	return &CLIArgParser{
		schema: schema,
		spec:   spec,
		result: make(map[string]packages.ArgumentMetadata),
	}
}

// Parse processes all arguments in sequence and returns the result
func (p *CLIArgParser) Parse(args []string) map[string]packages.ArgumentMetadata {
	for i := 0; i < len(args); i++ {
		currentIndex := i
		arg := args[i]

		if isFlag(arg) {
			consumed := p.parseFlag(arg, args, currentIndex)
			if consumed {
				i++ // Skip the next argument as it was consumed as a value
			}
		} else {
			p.parsePositional(arg, currentIndex)
		}
	}
	return p.result
}

// parsePositional handles non-flag arguments that may contain placeholders
func (p *CLIArgParser) parsePositional(arg string, position int) {
	// Skip package names and other non-placeholder positional args
	placeholder := extractPlaceholder(arg)
	if placeholder == "" {
		return
	}

	// Only increment counter and store if the placeholder exists in schema
	if metadata, exists := p.schema[placeholder]; exists {
		// Increment the positional counter and use it as the logical position
		p.positionalCount++
		p.storeResultWithPosition(placeholder, packages.VariableTypeArgPositional, metadata, p.positionalCount)
	}
}

// parseFlag handles flag-style arguments and returns true if it consumed the next argument
func (p *CLIArgParser) parseFlag(arg string, args []string, currentIndex int) bool {
	flag := extractFlagName(arg)
	if flag == "" || p.shouldIgnoreFlag(flag) {
		return false
	}

	// Check for embedded value (--flag=value or --flag=${PLACEHOLDER})
	if embeddedValue := extractFlagValue(arg); embeddedValue != "" {
		p.parseFlagWithValue(flag, embeddedValue)
		return false // Didn't consume next argument
	}

	// Check if next argument is the value for this flag
	nextIndex := currentIndex + 1
	if nextIndex < len(args) && !isFlag(args[nextIndex]) {
		p.parseFlagWithValue(flag, args[nextIndex])
		return true // Consumed next argument
	} else {
		// Boolean flag (no value)
		var metadata Argument
		if m, exists := p.schema[flag]; exists {
			metadata = m
		}
		p.storeResult(flag, packages.VariableTypeArgBool, metadata)
		return false // Didn't consume next argument
	}
}

// shouldIgnoreFlag determines if a flag should be skipped
func (p *CLIArgParser) shouldIgnoreFlag(flag string) bool {
	return p.spec.ShouldIgnoreFlag != nil && p.spec.ShouldIgnoreFlag(flag)
}

// parseFlagWithValue handles a flag that has an associated value
func (p *CLIArgParser) parseFlagWithValue(flag, value string) {
	// Check if the value is a placeholder reference
	placeholder := extractPlaceholder(value)

	var metadata Argument
	if placeholder != "" {
		// Use schema metadata if placeholder exists in schema
		if m, exists := p.schema[placeholder]; exists {
			metadata = m
		}
	} else {
		// Try to find metadata using the flag name itself
		if m, exists := p.schema[flag]; exists {
			metadata = m
		}
	}

	p.storeResult(flag, packages.VariableTypeArg, metadata)
}

// storeResult centralizes the storage of argument metadata
func (p *CLIArgParser) storeResult(key string, varType packages.VariableType, metadata Argument) {
	p.result[key] = packages.ArgumentMetadata{
		Name:         key,
		VariableType: varType,
		Required:     metadata.Required,
		Description:  metadata.Description,
		Example:      metadata.Example,
	}
}

// storeResultWithPosition stores argument metadata with position information
func (p *CLIArgParser) storeResultWithPosition(
	key string,
	varType packages.VariableType,
	metadata Argument,
	position int,
) {
	p.result[key] = packages.ArgumentMetadata{
		Name:         key,
		VariableType: varType,
		Required:     metadata.Required,
		Description:  metadata.Description,
		Example:      metadata.Example,
		Position:     &position,
	}
}
