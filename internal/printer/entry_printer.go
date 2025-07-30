package printer

import (
	"fmt"
	"io"
	"strings"

	"github.com/mozilla-ai/mcpd/v2/internal/cmd/output"
	"github.com/mozilla-ai/mcpd/v2/internal/config"
)

var _ output.Printer[config.ServerEntry] = (*ServerEntryPrinter)(nil)

type ServerEntryPrinter struct {
	headerFunc output.WriteFunc[config.ServerEntry]
	footerFunc output.WriteFunc[config.ServerEntry]
}

func (p *ServerEntryPrinter) Header(w io.Writer, count int) {
	if p.headerFunc != nil {
		p.headerFunc(w, count)
	}
}

func (p *ServerEntryPrinter) SetHeader(fn output.WriteFunc[config.ServerEntry]) {
	p.headerFunc = fn
}

func (p *ServerEntryPrinter) Item(w io.Writer, elem config.ServerEntry) error {
	_, _ = fmt.Fprintf(
		w,
		"✓ Added server '%s' (version: %s)\n  tools: %s\n",
		elem.Name,
		elem.PackageVersion(),
		strings.Join(elem.Tools, ", "),
	)

	if len(elem.RequiredEnvVars) > 0 {
		_, _ = fmt.Fprintf(
			w,
			"\n! The following environment variables are required for this server:\n\n  %s\n",
			strings.Join(elem.RequiredEnvVars, "\n  "),
		)

		_, _ = fmt.Fprint(w, "\nsee: mcpd config env set --help\n")
	}

	if len(elem.RequiredArgs) > 0 {
		_, _ = fmt.Fprintf(
			w,
			"\n! The following command line arguments are required for this server:\n\n  %s\n",
			strings.Join(elem.RequiredArgs, "\n  "),
		)

		_, _ = fmt.Fprint(w, "\nsee: mcpd config args set --help\n")
	}
	return nil
}

func (p *ServerEntryPrinter) Footer(w io.Writer, count int) {
	if p.footerFunc != nil {
		p.footerFunc(w, count)
	}
}

func (p *ServerEntryPrinter) SetFooter(fn output.WriteFunc[config.ServerEntry]) {
	p.footerFunc = fn
}
