package printer

import (
	"fmt"
	"io"

	"github.com/mozilla-ai/mcpd/internal/cmd/output"
)

var _ output.Printer[PluginConfigResult] = (*PluginConfigPrinter)(nil)

// PluginConfigResult represents plugin-level configuration.
type PluginConfigResult struct {
	Dir string `json:"dir,omitempty" yaml:"dir,omitempty"`
}

// PluginConfigPrinter handles text output for plugin-level config.
type PluginConfigPrinter struct {
	headerFunc output.WriteFunc[PluginConfigResult]
	footerFunc output.WriteFunc[PluginConfigResult]
}

// Header writes a custom header if one has been configured via SetHeader.
func (p *PluginConfigPrinter) Header(w io.Writer, count int) {
	if p.headerFunc != nil {
		p.headerFunc(w, count)
	}
}

// SetHeader configures a custom header function for the printer.
func (p *PluginConfigPrinter) SetHeader(fn output.WriteFunc[PluginConfigResult]) {
	p.headerFunc = fn
}

// Item writes formatted plugin-level configuration to the output.
func (p *PluginConfigPrinter) Item(w io.Writer, result PluginConfigResult) error {
	_, _ = fmt.Fprintln(w, "Plugin Configuration:")

	dir := result.Dir
	if dir == "" {
		dir = "(not configured)"
	}
	_, _ = fmt.Fprintf(w, "  Directory: %s\n", dir)

	return nil
}

// Footer writes a custom footer if one has been configured via SetFooter.
func (p *PluginConfigPrinter) Footer(w io.Writer, count int) {
	if p.footerFunc != nil {
		p.footerFunc(w, count)
	}
}

// SetFooter configures a custom footer function for the printer.
func (p *PluginConfigPrinter) SetFooter(fn output.WriteFunc[PluginConfigResult]) {
	p.footerFunc = fn
}
