// Package apiref renders a GitBook API reference page from a set of OpenAPI
// operations.
//
// Each operation becomes an openapi-operation block that references a
// specification registered in GitBook by name, so GitBook renders it as an
// interactive, auto-updating widget while non-GitBook viewers fall back to the
// spec link. The page is generated into the docs source so it flows one-way
// through Git Sync and is reproduced on every build.
package apiref

import (
	"fmt"
	"maps"
	"slices"
	"strings"
)

const (
	// SpecName is the name the OpenAPI specification is registered under in GitBook.
	// It must match the specification name in the GitBook organization.
	SpecName = "mcpd-openapi-spec"

	// SpecURL is the published location of the specification, shown as fallback
	// content for viewers that do not render GitBook blocks.
	SpecURL = "https://raw.githubusercontent.com/mozilla-ai/mcpd/gitbook-docs/api/openapi.yaml"

	// untaggedGroup titles the section for operations without a tag.
	untaggedGroup = "Other"
)

// Operation identifies a single API operation to document.
type Operation struct {
	Tag    string
	Method string
	Path   string
}

// Render returns the API reference page: operations grouped by tag (tags
// alphabetical, operations within a tag sorted by path then method), each as an
// openapi-operation block referencing the registered specification.
func Render(ops []Operation) string {
	byTag := map[string][]Operation{}
	for _, op := range ops {
		tag := op.Tag
		if tag == "" {
			tag = untaggedGroup
		}
		byTag[tag] = append(byTag[tag], op)
	}

	var b strings.Builder
	b.WriteString("# API Reference\n\n")
	b.WriteString("{% hint style=\"info\" %}\n")
	b.WriteString("Interactive reference generated from the mcpd OpenAPI specification.\n")
	b.WriteString("{% endhint %}\n")

	for _, tag := range slices.Sorted(maps.Keys(byTag)) {
		group := byTag[tag]
		slices.SortFunc(group, func(a, b Operation) int {
			if a.Path != b.Path {
				return strings.Compare(a.Path, b.Path)
			}
			return strings.Compare(a.Method, b.Method)
		})

		fmt.Fprintf(&b, "\n## %s\n\n", tag)
		for i, op := range group {
			if i > 0 {
				b.WriteString("\n")
			}
			fmt.Fprintf(&b, "{%% openapi-operation spec=%q path=%q method=%q %%}\n", SpecName, op.Path, op.Method)
			fmt.Fprintf(&b, "[OpenAPI %s](%s)\n", SpecName, SpecURL)
			b.WriteString("{% endopenapi-operation %}\n")
		}
	}

	return b.String()
}
