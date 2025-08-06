package config

import (
	"fmt"
	"slices"
	"strings"
)

const (
	// FlagPrefixLong represents the expected prefix for a long format flag (e.g. --flag).
	FlagPrefixLong = "--"

	// FlagPrefixShort represents the expected prefix for a short format flag (e.g. -f).
	FlagPrefixShort = "-"

	// FlagValueSeparator represents the value which is used to separate the flag name from the value.
	FlagValueSeparator = "="
)

// NormalizeArgs normalizes a slice of (CLI) arguments by extracting and formatting only flags.
//
// It transforms:
//
//	--flag value     -> --flag=value
//	-f value         -> -f=value
//	--flag=value     -> preserved as-is
//	-xyz             -> -x, -y, -z (expanded short flags)
//
// Positional arguments are excluded.
//
// This function is intended for internal normalization of flag arguments only.
func NormalizeArgs(rawArgs []string) []string {
	var normalized []string
	numArgs := len(rawArgs)

	for i := 0; i < numArgs; i++ {
		arg := strings.TrimSpace(rawArgs[i])

		nextIndex := i + 1
		hasNext := nextIndex < numArgs
		isShortFlag := strings.HasPrefix(arg, FlagPrefixShort) && !strings.HasPrefix(arg, FlagPrefixLong)
		containsValue := strings.Contains(arg, FlagValueSeparator)
		// isNotFlag returns true if the given string does not appear to be a flag
		isNotFlag := func(v string) bool {
			v = strings.TrimSpace(v)
			return !strings.HasPrefix(v, FlagPrefixShort) && !strings.HasPrefix(v, FlagPrefixLong)
		}

		// We shouldn't encounter args that aren't flags, because we look-ahead to extract arg values.
		if isNotFlag(arg) {
			continue
		}

		// -xyz => -x, -y, -z
		if isShortFlag && len(arg) > 2 && !containsValue {
			for _, c := range arg[1:] {
				normalized = append(normalized, fmt.Sprintf("-%c", c))
			}
			continue
		}

		// -f=value or --flag=value
		if containsValue {
			normalized = append(normalized, arg)
			continue
		}

		// -f or --flag
		// (handle the case where there's a 'next' arg which is a value that should be associated to this flag).
		if hasNext && isNotFlag(rawArgs[nextIndex]) {
			arg = arg + FlagValueSeparator + strings.TrimSpace(rawArgs[nextIndex])
			i++ // skip the next value as we've dealt with it.
		}
		normalized = append(normalized, arg)
	}

	return normalized
}

// RemoveMatchingFlags filters out all (CLI) flags from the input 'args' slice that match
// (based on a prefix and case) any of the specified flag names in 'toRemove'.
// The returned slice contains the filtered args with their order preserved.
func RemoveMatchingFlags(args []string, toRemove []string) []string {
	remove := make(map[string]struct{}, len(toRemove))
	for _, name := range toRemove {
		remove[name] = struct{}{}
	}

	filtered := make([]string, 0, len(args))
	for _, arg := range args {
		drop := false
		for flag := range remove {
			if arg == flag || strings.HasPrefix(arg, flag+FlagValueSeparator) {
				drop = true
				break
			}
		}
		if !drop {
			filtered = append(filtered, arg)
		}
	}

	return filtered
}

// MergeArgs merges all args present in 'b' into 'a', overwriting collisions.
// Any value originally in 'a' but not in 'b' are preserved.
// Supports args in the format --arg1 (bool flags) and --arg1=value1 (key/value flags).
// The returned slice preserves the order of 'a', and appends new flags from 'b' in order.
func MergeArgs(a, b []string) []string {
	// Handle early returns if we don't have work to do.
	if len(b) == 0 {
		return slices.Clone(a)
	}
	if len(a) == 0 {
		return slices.Clone(b)
	}

	overrides := parseArgs(b)
	result := make([]string, 0, len(a)+len(b))
	processed := make(map[string]struct{}, len(a)+len(b))

	// Process args from 'a', applying overrides from 'b'.
	for _, arg := range a {
		entry := parseArg(arg)
		if override, exists := overrides[entry.key]; exists {
			arg = override.String()
		}

		result = append(result, arg)
		processed[entry.key] = struct{}{}
	}

	// Append new args from 'b' that weren't in 'a'.
	for _, arg := range b {
		entry := parseArg(arg)
		if _, seen := processed[entry.key]; !seen {
			result = append(result, arg)
		}
	}

	return result
}

// ProcessAllArgs processes a slice of arguments, normalizing flags while preserving positional arguments.
// It processes arguments sequentially, normalizing flag groups as it encounters them,
// and returns them in their original relative order.
//
// Examples:
//   - ["--flag", "value", "pos1", "pos2"] -> ["--flag=value", "pos1", "pos2"]
//   - ["pos1", "--flag=value", "pos2"] -> ["pos1", "--flag=value", "pos2"]
//   - ["/path/to/dir", "--verbose"] -> ["/path/to/dir", "--verbose"]
func ProcessAllArgs(rawArgs []string) []string {
	if len(rawArgs) == 0 {
		return []string{}
	}

	var result []string

	// Process arguments sequentially
	for i := 0; i < len(rawArgs); i++ {
		arg := strings.TrimSpace(rawArgs[i])

		isFlag := strings.HasPrefix(arg, FlagPrefixShort) || strings.HasPrefix(arg, FlagPrefixLong)
		if isFlag {
			// Check for combined short flags (like -xyz) which should not consume next arg as value
			isCombinedShortFlag := strings.HasPrefix(arg, FlagPrefixShort) &&
				!strings.HasPrefix(arg, FlagPrefixLong) &&
				len(arg) > 2 &&
				!strings.Contains(arg, FlagValueSeparator)

			// Collect this flag and potentially its value for normalization
			flagGroup := []string{arg}

			// Check if next argument is a value for this flag (not embedded with =)
			// Don't consume next arg for combined short flags
			if !isCombinedShortFlag && !strings.Contains(arg, FlagValueSeparator) && i+1 < len(rawArgs) {
				nextArg := strings.TrimSpace(rawArgs[i+1])
				nextIsFlag := strings.HasPrefix(nextArg, FlagPrefixShort) || strings.HasPrefix(nextArg, FlagPrefixLong)
				if !nextIsFlag {
					// Include the value
					flagGroup = append(flagGroup, nextArg)
					i++ // Skip the next argument since we processed it
				}
			}

			// Normalize this flag group and add to result
			normalizedFlags := NormalizeArgs(flagGroup)
			result = append(result, normalizedFlags...)
		} else {
			// Positional argument - add as-is
			result = append(result, arg)
		}
	}

	return result
}

// parseArgs converts a slice of argument strings into a map of argEntry
func parseArgs(args []string) map[string]argEntry {
	result := make(map[string]argEntry, len(args))
	for _, arg := range args {
		entry := parseArg(arg)
		result[entry.key] = entry
	}
	return result
}

// parseArg extracts the key and value from a command line argument
func parseArg(arg string) argEntry {
	parts := strings.SplitN(arg, FlagValueSeparator, 2)
	entry := argEntry{
		key: strings.TrimSpace(parts[0]),
	}

	if len(parts) == 2 {
		entry.value = strings.TrimSpace(parts[1])
	}

	return entry
}
