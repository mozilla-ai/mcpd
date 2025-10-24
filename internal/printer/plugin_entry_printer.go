package printer

import (
	"fmt"
	"io"

	"github.com/mozilla-ai/mcpd/v2/internal/cmd/output"
	"github.com/mozilla-ai/mcpd/v2/internal/config"
)

var _ output.Printer[PluginEntryResult] = (*PluginEntryPrinter)(nil)

// PluginEntryResult represents a single plugin entry output structure.
type PluginEntryResult struct {
	config.PluginEntry

	Category config.Category `json:"category" yaml:"category"`
}

// PluginEntryPrinter handles text output for plugin entries.
type PluginEntryPrinter struct {
	headerFunc output.WriteFunc[PluginEntryResult]
	footerFunc output.WriteFunc[PluginEntryResult]
}

// Header writes a custom header if one has been configured via SetHeader.
func (p *PluginEntryPrinter) Header(w io.Writer, count int) {
	if p.headerFunc != nil {
		p.headerFunc(w, count)
	}
}

// SetHeader configures a custom header function for the printer.
func (p *PluginEntryPrinter) SetHeader(fn output.WriteFunc[PluginEntryResult]) {
	p.headerFunc = fn
}

// Item writes a formatted plugin entry to the output.
func (p *PluginEntryPrinter) Item(w io.Writer, result PluginEntryResult) error {
	_, _ = fmt.Fprintf(w, "Plugin '%s' in category '%s':\n", result.Name, result.Category)
	_, _ = fmt.Fprintf(w, "  Flows: %s\n", formatFlows(result.Flows))
	_, _ = fmt.Fprintf(w, "  Required: %s\n", formatRequired(result.Required))
	if result.CommitHash != nil {
		_, _ = fmt.Fprintf(w, "  Commit Hash: %s\n", *result.CommitHash)
	}

	return nil
}

// Footer writes a custom footer if one has been configured via SetFooter.
func (p *PluginEntryPrinter) Footer(w io.Writer, count int) {
	if p.footerFunc != nil {
		p.footerFunc(w, count)
	}
}

// SetFooter configures a custom footer function for the printer.
func (p *PluginEntryPrinter) SetFooter(fn output.WriteFunc[PluginEntryResult]) {
	p.footerFunc = fn
}
