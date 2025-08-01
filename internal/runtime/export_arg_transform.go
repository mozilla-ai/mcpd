package runtime

import (
	"fmt"
	"strings"
)

// valueArgTransformation holds different representations of a command line argument for export to configuration files:
// the original form, environment variable name, shell reference, and the formatted argument.
type valueArgTransformation struct {
	Raw             string // --foo=bar
	Name            string // foo
	EnvVarName      string // MCPD__TEST__FOO
	EnvVarReference string // ${MCPD__TEST__FOO}
	FormattedArg    string // --foo=${MCPD__TEST__FOO}
}

// normalizeForEnvVarName converts string name (e.g. app name, env var name) to uppercase with underscores
func normalizeForEnvVarName(s string) string {
	return strings.ToUpper(strings.ReplaceAll(s, "-", "_"))
}

// extractArgName extracts the argument name from --foo or --foo=bar, returning just "foo"
func extractArgName(rawArg string) string {
	// Remove leading dashes
	arg := strings.TrimLeft(rawArg, "-")

	// If there's an equals sign, take only the part before it
	if idx := strings.IndexByte(arg, '='); idx != -1 {
		arg = arg[:idx]
	}

	return arg
}

// extractArgNameWithPrefix extracts the argument name including prefix from --foo or --foo=bar, returning "--foo"
func extractArgNameWithPrefix(rawArg string) string {
	if idx := strings.IndexByte(rawArg, '='); idx != -1 {
		return rawArg[:idx]
	}
	return rawArg
}

// buildEnvVarName creates the environment variable name: MCPD__TEST__FOO
func buildEnvVarName(appName, serverName, argName string) string {
	sanitizedAppName := normalizeForEnvVarName(appName)
	sanitizedServer := normalizeForEnvVarName(serverName)
	sanitizedArg := strings.ToUpper(strings.ReplaceAll(argName, "-", "_"))
	return fmt.Sprintf("%s__%s__%s", sanitizedAppName, sanitizedServer, sanitizedArg)
}

// transformValueArg converts a raw CLI argument into its transformed representations.
// Only use this for value-type arguments (e.g. --foo=bar or ["--foo", "bar"]).
// Do not use for boolean flags (e.g. --enable-feature) that do not take a value.
func transformValueArg(appName, serverName, rawArg string) *valueArgTransformation {
	argName := extractArgName(rawArg)
	envVarName := buildEnvVarName(appName, serverName, argName)
	argPrefix := extractArgNameWithPrefix(rawArg)

	// Value argument: --foo=bar => --foo=${MCPD__TEST__FOO}
	formattedArg := fmt.Sprintf("%s=${%s}", argPrefix, envVarName)

	return &valueArgTransformation{
		Raw:             rawArg,
		Name:            argName,
		EnvVarName:      envVarName,
		EnvVarReference: fmt.Sprintf("${%s}", envVarName),
		FormattedArg:    formattedArg,
	}
}
