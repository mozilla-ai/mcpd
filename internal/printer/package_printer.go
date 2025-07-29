package printer

import (
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
	if _, err := fmt.Fprintf(w, "  🆔 %s\n", pkg.ID); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "  🔒 Official: %s\n", map[bool]string{true: "✅", false: "❌"}[pkg.IsOfficial]); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "  📁 Registry: %s\n", pkg.Source); err != nil {
		return err
	}

	if strings.TrimSpace(pkg.Description) != "" {
		if _, err := fmt.Fprintf(w, "  ℹ️ Description: %s\n", pkg.Description); err != nil {
			return err
		}
	}

	if strings.TrimSpace(pkg.License) != "" {
		if _, err := fmt.Fprintf(w, "  📄 License: %s\n", pkg.License); err != nil {
			return err
		}
	}

	if len(pkg.Runtimes) > 0 {
		runtimes := make([]string, len(pkg.Runtimes))
		for i, r := range pkg.Runtimes {
			runtimes[i] = string(r)
		}
		if _, err := fmt.Fprintf(w, "  🏗️ Runtimes: %s\n", strings.Join(runtimes, ", ")); err != nil {
			return err
		}
	} else {
		if _, err := fmt.Fprintf(w, "  ⚠️ Warning: No supported runtimes found in package description\n"); err != nil {
			return err
		}
	}

	if len(pkg.Tools) > 0 {
		if _, err := fmt.Fprintf(w, "  🔨 Tools: %s\n", strings.Join(pkg.Tools.Names(), ", ")); err != nil {
			return err
		}
	} else {
		if _, err := fmt.Fprintf(w, "  ⚠️ Warning: No tools found in package description\n"); err != nil {
			return err
		}
	}

	if len(pkg.Tags) > 0 {
		if _, err := fmt.Fprintf(w, "  🏷️ Tags: %s\n", strings.Join(pkg.Tags, ", ")); err != nil {
			return err
		}
	}

	if len(pkg.Categories) > 0 {
		if _, err := fmt.Fprintf(w, "  📂 Categories: %s\n", strings.Join(pkg.Categories, ", ")); err != nil {
			return err
		}
	}

	if len(pkg.Arguments) > 0 {
		if _, err := fmt.Fprintln(w, "  ⚙️ Found startup args..."); err != nil {
			return err
		}
		requiredArgs := getArgs(pkg.Arguments, true)
		if len(requiredArgs) > 0 {
			if _, err := fmt.Fprintf(w, "  ❗ Required: %s\n", strings.Join(requiredArgs, ", ")); err != nil {
				return err
			}
		}
		optionalArgs := getArgs(pkg.Arguments, false)
		if len(optionalArgs) > 0 {
			if _, err := fmt.Fprintf(w, "  🔹️ Optional: %s\n", strings.Join(optionalArgs, ", ")); err != nil {
				return err
			}
		}

		envs := pkg.Arguments.FilterBy(packages.EnvVar).Names()
		if len(envs) > 0 {
			if _, err := fmt.Fprintln(w, "  📋 Args configurable via environment variables..."); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(w, "  🌍 %s\n", strings.Join(envs, ", ")); err != nil {
				return err
			}
		}

		args := pkg.Arguments.FilterBy(packages.Argument).Names()
		if len(args) > 0 {
			if _, err := fmt.Fprintln(w, "  📋 Args configurable via command line..."); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(w, "  🖥️ %s\n", strings.Join(args, ", ")); err != nil {
				return err
			}
		}
	}

	return nil
}

func getArgs(args map[string]packages.ArgumentMetadata, required bool) []string {
	res := make([]string, 0, len(args))
	for name, meta := range args {
		if meta.Required == required {
			res = append(res, name)
		}
	}
	slices.Sort(res)
	return res
}
