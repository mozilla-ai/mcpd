package runtime

import "strings"

// Runtime represents the type of runtime an MCP server can be run under.
type Runtime string

const (
	// NPX represents the 'npx' Node package runner (Node Package Execute) for NodeJS packages.
	NPX Runtime = "npx"

	// UVX represents the 'uvx' UV runner for Python packages.
	UVX Runtime = "uvx"

	Python Runtime = "python"

	Docker Runtime = "docker"

	// TODO: Add other runtimes as required...
)

// AnyIntersection returns true if any value in a is also in b.
func AnyIntersection(a []Runtime, b []Runtime) bool {
	if a == nil || b == nil || len(a) == 0 || len(b) == 0 {
		return false
	}

	set := map[Runtime]struct{}{}
	for _, v := range b {
		set[v] = struct{}{}
	}
	for _, v := range a {
		if _, ok := set[v]; ok {
			return true
		}
	}
	return false
}

func Join(runtimes []Runtime, sep string) string {
	res := make([]string, len(runtimes))
	for i, r := range runtimes {
		res[i] = string(r)
	}
	return strings.Join(res, sep)
}
