package printer

import (
	"fmt"
	"io"

	"github.com/mozilla-ai/mcpd/internal/cmd/output"
)

var _ output.Printer[ToolsListResult] = (*ToolsListPrinter)(nil)

// ToolsListResult represents the tools list output structure.
type ToolsListResult struct {
	Server string   `json:"server" yaml:"server"`
	Tools  []string `json:"tools"  yaml:"tools"`
	Count  int      `json:"count"  yaml:"count"`
}

type ToolsListPrinter struct {
	headerFunc output.WriteFunc[ToolsListResult]
	footerFunc output.WriteFunc[ToolsListResult]
}

func (p *ToolsListPrinter) Header(w io.Writer, count int) {
	if p.headerFunc != nil {
		p.headerFunc(w, count)
	}
}

func (p *ToolsListPrinter) SetHeader(fn output.WriteFunc[ToolsListResult]) {
	p.headerFunc = fn
}

func (p *ToolsListPrinter) Item(w io.Writer, result ToolsListResult) error {
	_, _ = fmt.Fprintf(w, "Tools for '%s' (%d total):\n", result.Server, result.Count)

	if len(result.Tools) == 0 {
		_, _ = fmt.Fprintln(w, "  (No tools configured)")
	} else {
		// Tools should already be sorted.
		for _, tool := range result.Tools {
			_, _ = fmt.Fprintf(w, "  %s\n", tool)
		}
	}

	return nil
}

func (p *ToolsListPrinter) Footer(w io.Writer, count int) {
	if p.footerFunc != nil {
		p.footerFunc(w, count)
	}
}

func (p *ToolsListPrinter) SetFooter(fn output.WriteFunc[ToolsListResult]) {
	p.footerFunc = fn
}
