package printer

import (
	"cmp"
	"fmt"
	"io"
	"maps"
	"slices"
	"strings"

	"github.com/mozilla-ai/mcpd/v2/internal/cmd/output"
	"github.com/mozilla-ai/mcpd/v2/internal/packages"
	"github.com/mozilla-ai/mcpd/v2/internal/runtime"
)

var _ output.Printer[packages.Server] = (*ServerPrinter)(nil)

func DefaultServerHeader() output.WriteFunc[packages.Server] {
	return nil
}

func DefaultServerFooter() output.WriteFunc[packages.Server] {
	return func(w io.Writer, _ int) {
		_, _ = fmt.Fprintln(w, "")
		_, _ = fmt.Fprintln(w, "────────────────────────────────────────────")
		_, _ = fmt.Fprintln(w, "")
	}
}

type ServerPrinter struct {
	headerFunc output.WriteFunc[packages.Server]
	footerFunc output.WriteFunc[packages.Server]
}

func NewServerPrinter() *ServerPrinter {
	return &ServerPrinter{
		headerFunc: DefaultServerHeader(),
		footerFunc: DefaultServerFooter(),
	}
}

func (p *ServerPrinter) Header(w io.Writer, count int) {
	if p.headerFunc != nil {
		p.headerFunc(w, count)
	}
}

func (p *ServerPrinter) SetHeader(fn output.WriteFunc[packages.Server]) {
	p.headerFunc = fn
}

func (p *ServerPrinter) Footer(w io.Writer, count int) {
	if p.footerFunc != nil {
		p.footerFunc(w, count)
	}
}

func (p *ServerPrinter) SetFooter(fn output.WriteFunc[packages.Server]) {
	p.footerFunc = fn
}

// Item outputs a single server entry.
func (p *ServerPrinter) Item(w io.Writer, pkg packages.Server) error {
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

	if len(pkg.Installations) > 0 {
		p.printRuntimes(w, pkg)
	} else {
		_, _ = fmt.Fprintf(w, "  ⚠️ Warning: No supported runtimes found\n")
	}

	if len(pkg.Tools) > 0 {
		p.printTools(w, pkg)
	} else {
		_, _ = fmt.Fprintf(w, "  ⚠️ Warning: No tools found\n")
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

func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

func (p *ServerPrinter) printRuntimes(w io.Writer, pkg packages.Server) {
	if len(pkg.Installations) == 0 {
		return
	}

	keys := slices.Collect(maps.Keys(pkg.Installations))
	slices.SortFunc(keys, func(a, b runtime.Runtime) int {
		return cmp.Compare(string(a), string(b))
	})

	// Use the same width as other sections (24 characters)
	const alignWidth = 24

	format := func(inst packages.Installation) string {
		var b strings.Builder
		b.WriteString(string(inst.Runtime))
		if inst.Version != "" {
			b.WriteString(fmt.Sprintf(" (version: %s)", inst.Version))
		}

		// Pad to align with other sections
		currentStr := b.String()
		paddedStr := padRight(currentStr, alignWidth)

		if inst.Deprecated {
			return paddedStr + "\t⚠️ (Deprecated)"
		}
		return paddedStr
	}

	_, _ = fmt.Fprintln(w, "  Runtimes:")
	for _, rt := range keys {
		inst := pkg.Installations[rt]
		_, _ = fmt.Fprintf(w, "\t%s\n", format(inst))
	}
}

func (p *ServerPrinter) printTools(w io.Writer, pkg packages.Server) {
	slices.SortFunc(pkg.Tools, func(a, b packages.Tool) int {
		return cmp.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	})

	_, _ = fmt.Fprintln(w, "  🔨 Tools:")
	for _, t := range pkg.Tools {
		_, _ = fmt.Fprintf(w, "\t- %s\n", t.Name)
	}
}

func (p *ServerPrinter) printEnvVars(w io.Writer, pkg packages.Server) {
	envs := pkg.Arguments.FilterBy(packages.EnvVar).Ordered()
	if len(envs) == 0 {
		return
	}

	_, _ = fmt.Fprintln(w, "  🌍 Environment variables:")
	for _, env := range envs {
		_, _ = fmt.Fprintf(w, "\t%s\t%s\n", padRight(env.Name, 24), env.Description)
	}
}

func (p *ServerPrinter) printRequiredEnvs(w io.Writer, pkg packages.Server) {
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

func (p *ServerPrinter) printPositionalArgs(w io.Writer, pkg packages.Server) {
	args := pkg.Arguments.FilterBy(packages.PositionalArgument).Ordered()
	if len(args) == 0 {
		return
	}

	_, _ = fmt.Fprintln(w, "  📍 Positional arguments:")
	for _, arg := range args {
		_, _ = fmt.Fprintf(w, "\t(%d) %s\t%s\n", *arg.Position, padRight(arg.Name, 24), arg.Description)
	}
}

func (p *ServerPrinter) printValueFlags(w io.Writer, pkg packages.Server) {
	args := pkg.Arguments.FilterBy(packages.ValueArgument).Ordered()
	if len(args) == 0 {
		return
	}

	_, _ = fmt.Fprintln(w, "  🚩 Value flags:")
	for _, arg := range args {
		_, _ = fmt.Fprintf(w, "\t%s\t%s\n", padRight(arg.Name, 24), arg.Description)
	}
}

func (p *ServerPrinter) printBoolFlags(w io.Writer, pkg packages.Server) {
	args := pkg.Arguments.FilterBy(packages.BoolArgument).Ordered()
	if len(args) == 0 {
		return
	}

	_, _ = fmt.Fprintln(w, "  ✅ Boolean flags:")
	for _, arg := range args {
		_, _ = fmt.Fprintf(w, "\t%s\t%s\n", padRight(arg.Name, 24), arg.Description)
	}
}

func (p *ServerPrinter) printRequiredArgs(w io.Writer, pkg packages.Server) {
	args := pkg.Arguments.FilterBy(packages.Argument, packages.Required).Ordered()
	if len(args) == 0 {
		return
	}

	_, _ = fmt.Fprintln(w, "  ❗ Required args:")
	for _, arg := range args {
		_, _ = fmt.Fprintf(w, "\t%s\n", arg.Name)
	}
}
