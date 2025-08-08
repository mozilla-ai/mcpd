package printer

import (
	"fmt"
	"io"
	"slices"
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
		"✓ Added server '%s' (version: %s)\n",
		elem.Name,
		elem.PackageVersion(),
	)

	slices.Sort(elem.Tools)
	_, _ = fmt.Fprintf(
		w,
		"  tools: %s\n",
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

	if len(elem.RequiredPositionalArgs) > 0 || len(elem.RequiredValueArgs) > 0 || len(elem.RequiredBoolArgs) > 0 {
		if len(elem.RequiredPositionalArgs) > 0 {
			_, _ = fmt.Fprintf(w, "\n❗ The following positional arguments are required for this server:\n\n")
			position := 0
			parts := make([]string, 0, len(elem.RequiredPositionalArgs))
			for _, posArg := range elem.RequiredPositionalArgs {
				position++
				parts = append(parts, fmt.Sprintf("  📍 (%d) %s", position, posArg))
			}
			_, _ = fmt.Fprintln(w, strings.Join(parts, "\n"))
		}

		if len(elem.RequiredValueArgs) > 0 {
			_, _ = fmt.Fprintf(
				w,
				"\n❗ The following command line arguments are required (along with values) for this server:\n\n",
			)
			parts := make([]string, 0, len(elem.RequiredValueArgs))
			for _, arg := range elem.RequiredValueArgs {
				parts = append(parts, fmt.Sprintf("  🚩 %s", arg))
			}
			_, _ = fmt.Fprintln(w, strings.Join(parts, "\n"))
		}

		if len(elem.RequiredBoolArgs) > 0 {
			_, _ = fmt.Fprintf(
				w,
				"\n❗ The following command line arguments are required (as boolean flags) for this server:\n\n",
			)
			parts := make([]string, 0, len(elem.RequiredBoolArgs))
			for _, arg := range elem.RequiredBoolArgs {
				parts = append(parts, fmt.Sprintf("  ✅ %s", arg))
			}
			_, _ = fmt.Fprintln(w, strings.Join(parts, "\n"))
		}

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
