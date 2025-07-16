package printer

import (
	"fmt"
	"io"

	"github.com/mozilla-ai/mcpd/v2/internal/packages"
)

type PackageListPrinter struct {
	inner Printer
}

func NewPackageListPrinter(inner Printer) *PackageListPrinter {
	return &PackageListPrinter{inner: inner}
}

func (p *PackageListPrinter) Header(w io.Writer, _ int) {
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "🔎 Registry search results...")
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "────────────────────────────────────────────")
	_, _ = fmt.Fprintln(w, "")
}

func (p *PackageListPrinter) Item(_ io.Writer, pkg packages.Package) error {
	return p.inner.PrintPackage(pkg)
}

func (p *PackageListPrinter) Footer(w io.Writer, count int) {
	_, _ = fmt.Fprintf(w, "📦 Found %d package%s\n", count, map[bool]string{true: "s"}[count > 1])
	_, _ = fmt.Fprintln(w, "")
}
