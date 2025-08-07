package printer

import (
	"fmt"
	"io"

	"github.com/mozilla-ai/mcpd/v2/internal/cmd/output"
	"github.com/mozilla-ai/mcpd/v2/internal/packages"
)

type ServerResultsPrinter struct {
	headerFunc output.WriteFunc[packages.Server]
	footerFunc output.WriteFunc[packages.Server]
	Printer    output.Printer[packages.Server]
}

func NewServerResultsPrinter(prn output.Printer[packages.Server]) *ServerResultsPrinter {
	return &ServerResultsPrinter{
		headerFunc: DefaultResultsHeader(),
		footerFunc: DefaultResultsFooter(),
		Printer:    prn,
	}
}

func (p *ServerResultsPrinter) Header(w io.Writer, count int) {
	if p.headerFunc != nil {
		p.headerFunc(w, count)
	}
}

func (p *ServerResultsPrinter) SetHeader(fn output.WriteFunc[packages.Server]) {
	p.headerFunc = fn
}

func (p *ServerResultsPrinter) Item(w io.Writer, pkg packages.Server) error {
	p.Printer.Header(w, 0)

	if err := p.Printer.Item(w, pkg); err != nil {
		return err
	}

	p.Printer.Footer(w, 0)

	return nil
}

func (p *ServerResultsPrinter) Footer(w io.Writer, count int) {
	if p.footerFunc != nil {
		p.footerFunc(w, count)
	}
}

func (p *ServerResultsPrinter) SetFooter(fn output.WriteFunc[packages.Server]) {
	p.footerFunc = fn
}

func DefaultResultsHeader() output.WriteFunc[packages.Server] {
	return func(w io.Writer, count int) {
		_, _ = fmt.Fprintln(w, "")
		_, _ = fmt.Fprintln(w, "🔎 Registry search results...")
		_, _ = fmt.Fprintln(w, "")
		_, _ = fmt.Fprintln(w, "────────────────────────────────────────────")
		_, _ = fmt.Fprintln(w, "")
	}
}

func DefaultResultsFooter() output.WriteFunc[packages.Server] {
	return func(w io.Writer, count int) {
		_, _ = fmt.Fprintf(w, "📦 Found %d servers%s\n", count, map[bool]string{true: "s"}[count > 1])
		_, _ = fmt.Fprintln(w, "")
	}
}
