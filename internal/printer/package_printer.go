package printer

import (
	"cmp"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/mozilla-ai/mcpd/v2/internal/cmd/output"
	"github.com/mozilla-ai/mcpd/v2/internal/packages"
)

var _ output.Printer[packages.Package] = (*PackagePrinter)(nil)

func DefaultPackageHeader() output.WriteFunc[packages.Package] {
	return nil
}

func DefaultPackageFooter() output.WriteFunc[packages.Package] {
	return func(w io.Writer, _ int) {
		_, _ = fmt.Fprintln(w, "")
		_, _ = fmt.Fprintln(w, "────────────────────────────────────────────")
		_, _ = fmt.Fprintln(w, "")
	}
}

type PackagePrinter struct {
	headerFunc output.WriteFunc[packages.Package]
	footerFunc output.WriteFunc[packages.Package]
}

func NewPackagePrinter() *PackagePrinter {
	return &PackagePrinter{
		headerFunc: DefaultPackageHeader(),
		footerFunc: DefaultPackageFooter(),
	}
}

func (p *PackagePrinter) Header(w io.Writer, count int) {
	if p.headerFunc != nil {
		p.headerFunc(w, count)
	}
}

func (p *PackagePrinter) SetHeader(fn output.WriteFunc[packages.Package]) {
	p.headerFunc = fn
}

func (p *PackagePrinter) Footer(w io.Writer, count int) {
	if p.footerFunc != nil {
		p.footerFunc(w, count)
	}
}

func (p *PackagePrinter) SetFooter(fn output.WriteFunc[packages.Package]) {
	p.footerFunc = fn
}

// Item outputs a single package entry.
func (p *PackagePrinter) Item(w io.Writer, pkg packages.Package) error {
	parts := []string{
		fmt.Sprintf("  🆔 %s", pkg.ID),
	}

	if pkg.IsOfficial {
		parts = append(parts, "✓ (Official)")
	}

	if pkg.Installations.AnyDeprecated() {
		parts = append(parts, "⚠️ (Deprecated)")
	}

	_, _ = fmt.Fprintf(w, "%s\n", strings.Join(parts, " "))
	_, _ = fmt.Fprintf(w, "  Source: %s\n", pkg.Source)

	if strings.TrimSpace(pkg.Description) != "" {
		_, _ = fmt.Fprintf(w, "  Description: %s\n", pkg.Description)
	}

	if strings.TrimSpace(pkg.License) != "" {
		_, _ = fmt.Fprintf(w, "  License: %s\n", pkg.License)
	}

	if len(pkg.Categories) > 0 {
		slices.SortFunc(pkg.Categories, func(a, b string) int {
			return strings.Compare(strings.ToLower(a), strings.ToLower(b))
		})
		_, _ = fmt.Fprintf(w, "  Categories: %s\n", strings.Join(pkg.Categories, ", "))
	}

	if len(pkg.Tags) > 0 {
		slices.SortFunc(pkg.Tags, func(a, b string) int {
			return strings.Compare(strings.ToLower(a), strings.ToLower(b))
		})
		_, _ = fmt.Fprintf(w, "  Tags: %s\n", strings.Join(pkg.Tags, ", "))
	}

	if len(pkg.Runtimes) > 0 {
		runtimes := make([]string, len(pkg.Runtimes))
		for i, r := range pkg.Runtimes {
			runtimes[i] = string(r)
		}
		_, _ = fmt.Fprintf(w, "  Runtimes: %s\n", strings.Join(runtimes, ", "))
	} else {
		_, _ = fmt.Fprintf(w, "  ⚠️ Warning: No supported runtimes found in package description\n")
	}

	if len(pkg.Tools) > 0 {
		p.printTools(w, pkg)
	} else {
		_, _ = fmt.Fprintf(w, "  ⚠️ Warning: No tools found in package description\n")
	}

	if len(pkg.Arguments) > 0 {
		p.printEnvVars(w, pkg)
		p.printPositionalArgs(w, pkg)
		p.printValueFlags(w, pkg)
		p.printBoolFlags(w, pkg)
		// Required.
		p.printRequiredEnvs(w, pkg)
		p.printRequiredArgs(w, pkg)
	}

	return nil
}

func pad(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

func (p *PackagePrinter) printTools(w io.Writer, pkg packages.Package) {
	slices.SortFunc(pkg.Tools, func(a, b packages.Tool) int {
		return cmp.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	})

	_, _ = fmt.Fprintln(w, "  🔨 Tools:")
	for _, t := range pkg.Tools {
		_, _ = fmt.Fprintf(w, "\t- %s\n", t.Name)
	}
}

func (p *PackagePrinter) printEnvVars(w io.Writer, pkg packages.Package) {
	envs := pkg.Arguments.FilterBy(packages.EnvVar).Ordered()
	if len(envs) == 0 {
		return
	}

	_, _ = fmt.Fprintln(w, "  🌍 Environment variables:")
	for _, env := range envs {
		_, _ = fmt.Fprintf(w, "\t%s\t%s\n", pad(env.Name, 24), env.Description)
	}
}

func (p *PackagePrinter) printRequiredEnvs(w io.Writer, pkg packages.Package) {
	envs := pkg.Arguments.FilterBy(packages.EnvVar, packages.Required).Names()
	if len(envs) == 0 {
		return
	}

	slices.Sort(envs)

	_, _ = fmt.Fprintln(w, "  ❗ Required env vars:")
	for _, env := range envs {
		_, _ = fmt.Fprintf(w, "\t- %s\n", env)
	}
}

func (p *PackagePrinter) printPositionalArgs(w io.Writer, pkg packages.Package) {
	args := pkg.Arguments.FilterBy(packages.PositionalArgument).Ordered()
	if len(args) == 0 {
		return
	}

	_, _ = fmt.Fprintln(w, "  📍 Positional arguments:")
	for _, arg := range args {
		_, _ = fmt.Fprintf(w, "\t(%d) %s\t%s\n", *arg.Position, pad(arg.Name, 24), arg.Description)
	}
}

func (p *PackagePrinter) printValueFlags(w io.Writer, pkg packages.Package) {
	args := pkg.Arguments.FilterBy(packages.ValueArgument).Ordered()
	if len(args) == 0 {
		return
	}

	_, _ = fmt.Fprintln(w, "  🚩 Value flags:")
	for _, arg := range args {
		_, _ = fmt.Fprintf(w, "\t%s\t%s\n", pad(arg.Name, 24), arg.Description)
	}
}

func (p *PackagePrinter) printBoolFlags(w io.Writer, pkg packages.Package) {
	args := pkg.Arguments.FilterBy(packages.BoolArgument).Ordered()
	if len(args) == 0 {
		return
	}

	_, _ = fmt.Fprintln(w, "  ✅ Boolean flags:")
	for _, arg := range args {
		_, _ = fmt.Fprintf(w, "\t%s\t%s\n", pad(arg.Name, 24), arg.Description)
	}
}

func (p *PackagePrinter) printRequiredArgs(w io.Writer, pkg packages.Package) {
	args := pkg.Arguments.FilterBy(packages.Argument, packages.Required).Ordered()
	if len(args) == 0 {
		return
	}

	_, _ = fmt.Fprintln(w, "  ❗ Required args:")
	for _, arg := range args {
		_, _ = fmt.Fprintf(w, "\t%s\n", arg.Name)
	}
}
