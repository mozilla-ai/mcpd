package printer

import (
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/mozilla-ai/mcpd/internal/cmd/output"
	"github.com/mozilla-ai/mcpd/internal/config"
)

var _ output.Printer[PluginListResult] = (*PluginListPrinter)(nil)

// PluginListResult represents the plugin list output structure.
type PluginListResult struct {
	// Categories represent the configured plugins by category.
	Categories map[config.Category][]config.PluginEntry `json:"categories" yaml:"categories"`

	// TotalPlugins is the total number of distinct plugins configured across all categories.
	TotalPlugins int `json:"totalPlugins" yaml:"total_plugins"`
}

// PluginListPrinter handles text output for plugin lists.
type PluginListPrinter struct {
	// headerFunc is an optional custom header function.
	headerFunc output.WriteFunc[PluginListResult]

	// footerFunc is an optional custom footer function.
	footerFunc output.WriteFunc[PluginListResult]
}

// NewPluginListResult creates a new PluginListResult with the given categories.
// It automatically calculates the distinct plugin count across all categories.
func NewPluginListResult(categories map[config.Category][]config.PluginEntry) PluginListResult {
	// Count distinct plugin names across all categories.
	distinctPlugins := make(map[string]struct{})
	for _, plugins := range categories {
		for _, plugin := range plugins {
			distinctPlugins[plugin.Name] = struct{}{}
		}
	}

	return PluginListResult{
		Categories:   categories,
		TotalPlugins: len(distinctPlugins),
	}
}

// Header writes a custom header if one has been configured via SetHeader.
func (p *PluginListPrinter) Header(w io.Writer, count int) {
	if p.headerFunc != nil {
		p.headerFunc(w, count)
	}
}

// SetHeader configures a custom header function for the printer.
func (p *PluginListPrinter) SetHeader(fn output.WriteFunc[PluginListResult]) {
	p.headerFunc = fn
}

// Item writes a formatted plugin list to the output.
// It automatically detects single vs multi-category mode based on the number of categories.
func (p *PluginListPrinter) Item(w io.Writer, result PluginListResult) error {
	if len(result.Categories) == 0 {
		_, _ = fmt.Fprintln(w, "No plugins configured in any category")
		return nil
	}

	// Single category mode.
	if len(result.Categories) == 1 {
		for category, plugins := range result.Categories {
			_, _ = fmt.Fprintf(w, "Configured plugins in '%s' (%d total):\n", category, len(plugins))
			if len(plugins) == 0 {
				_, _ = fmt.Fprintln(w, "  (No plugins configured)")
			} else {
				p.printPluginEntries(w, plugins)
			}
		}
		return nil
	}

	// Multiple categories mode.
	_, _ = fmt.Fprintf(w, "Configured plugins (%d total):\n", result.TotalPlugins)
	for _, category := range config.OrderedCategories() {
		plugins, ok := result.Categories[category]
		if !ok || len(plugins) == 0 {
			continue
		}
		_, _ = fmt.Fprintf(w, "\n%s (%d total):\n", category, len(plugins))
		p.printPluginEntries(w, plugins)
	}

	return nil
}

// printPluginEntries writes formatted plugin entry details including flows, required status, and optional commit hash.
func (p *PluginListPrinter) printPluginEntries(w io.Writer, entries []config.PluginEntry) {
	for _, entry := range entries {
		_, _ = fmt.Fprintf(w, "  %s\n", entry.Name)
		_, _ = fmt.Fprintf(w, "    Flows: %s\n", formatFlows(entry.Flows))
		_, _ = fmt.Fprintf(w, "    Required: %v\n", formatRequired(entry.Required))
		if entry.CommitHash != nil {
			_, _ = fmt.Fprintf(w, "    Commit Hash: %s\n", *entry.CommitHash)
		}
	}
}

// Footer writes a custom footer if one has been configured via SetFooter.
func (p *PluginListPrinter) Footer(w io.Writer, count int) {
	if p.footerFunc != nil {
		p.footerFunc(w, count)
	}
}

// SetFooter configures a custom footer function for the printer.
func (p *PluginListPrinter) SetFooter(fn output.WriteFunc[PluginListResult]) {
	p.footerFunc = fn
}

// formatFlows converts a slice of flows to a sorted, comma-separated string.
func formatFlows(flows []config.Flow) string {
	tmp := slices.Clone(flows)

	slices.Sort(tmp)

	result := make([]string, len(tmp))
	for i, flow := range tmp {
		result[i] = string(flow)
	}

	return strings.Join(result, ", ")
}

// formatRequired converts a required pointer to a string, defaulting to "false" when nil.
func formatRequired(required *bool) string {
	if required == nil {
		return "false"
	}

	return fmt.Sprintf("%v", *required)
}
