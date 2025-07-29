package printer

import (
	"fmt"
	"io"

	"github.com/mozilla-ai/mcpd/v2/internal/cmd/output"
	"github.com/mozilla-ai/mcpd/v2/internal/packages"
)

type PackageResultsPrinter struct {
	headerFunc     output.WriteFunc[packages.Package]
	footerFunc     output.WriteFunc[packages.Package]
	PackagePrinter output.Printer[packages.Package]
}

func NewPackageResultsPrinter(prn output.Printer[packages.Package]) *PackageResultsPrinter {
	return &PackageResultsPrinter{
		headerFunc:     DefaultResultsHeader(),
		footerFunc:     DefaultResultsFooter(),
		PackagePrinter: prn,
	}
}

func (p *PackageResultsPrinter) Header(w io.Writer, count int) {
	if p.headerFunc != nil {
		p.headerFunc(w, count)
	}
}

func (p *PackageResultsPrinter) SetHeader(fn output.WriteFunc[packages.Package]) {
	p.headerFunc = fn
}

func (p *PackageResultsPrinter) Item(w io.Writer, pkg packages.Package) error {
	p.PackagePrinter.Header(w, 0)

	if err := p.PackagePrinter.Item(w, pkg); err != nil {
		return err
	}

	p.PackagePrinter.Footer(w, 0)

	return nil
}

func (p *PackageResultsPrinter) Footer(w io.Writer, count int) {
	if p.footerFunc != nil {
		p.footerFunc(w, count)
	}
}

func (p *PackageResultsPrinter) SetFooter(fn output.WriteFunc[packages.Package]) {
	p.footerFunc = fn
}

func DefaultResultsHeader() output.WriteFunc[packages.Package] {
	return func(w io.Writer, count int) {
		_, _ = fmt.Fprintln(w, "")
		_, _ = fmt.Fprintln(w, "🔎 Registry search results...")
		_, _ = fmt.Fprintln(w, "")
		_, _ = fmt.Fprintln(w, "────────────────────────────────────────────")
		_, _ = fmt.Fprintln(w, "")
	}
}

func DefaultResultsFooter() output.WriteFunc[packages.Package] {
	return func(w io.Writer, count int) {
		_, _ = fmt.Fprintf(w, "📦 Found %d package%s\n", count, map[bool]string{true: "s"}[count > 1])
		_, _ = fmt.Fprintln(w, "")
	}
}
