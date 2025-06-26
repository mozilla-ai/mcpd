package printer

import (
	"fmt"
	"io"
	"os"
	"slices"
	"strings"

	"github.com/mozilla-ai/mcpd/v2/internal/packages"
)

type Printer interface {
	PrintPackage(pkg packages.Package) error
	SetOptions(opt ...PackagePrinterOption) error
}

type DefaultPrinter struct{}

type PackagePrinter struct {
	out  io.Writer
	opts PackagePrinterOptions
}

func (p *DefaultPrinter) PrintPackage(pkg packages.Package) error {
	if _, err := fmt.Fprintf(os.Stderr, "Default:\n%#v\n", pkg); err != nil {
		return err
	}
	return nil
}

func (p *DefaultPrinter) SetOptions(opt ...PackagePrinterOption) error {
	return nil
}

// NewPrinter creates a new Printer with the provided output options.
func NewPrinter(out io.Writer, options ...PackagePrinterOption) (Printer, error) {
	opts, err := NewPackagePrinterOptions(options...)
	if err != nil {
		return nil, err
	}

	return &PackagePrinter{
		out:  out,
		opts: opts,
	}, nil
}

func (p *PackagePrinter) SetOptions(opt ...PackagePrinterOption) error {
	// Get current options.
	opts := []PackagePrinterOption{
		WithHeader(p.opts.showHeader),
		WithSeparator(p.opts.showSeparator),
		WithMissingWarnings(p.opts.showMissingWarning),
	}

	// Add updated options.
	for _, o := range opt {
		opts = append(opts, o)
	}

	// 'last write wins' for options, so updated options will be applied.
	newOpts, err := NewPackagePrinterOptions(opts...)
	if err != nil {
		return err
	}
	p.opts = newOpts

	return nil
}

// PrintPackage outputs a single package entry with options.
func (p *PackagePrinter) PrintPackage(pkg packages.Package) error {
	err := p.printDetails(pkg)
	if err != nil {
		return err
	}

	if p.opts.showSeparator {
		_, err := fmt.Fprintln(p.out)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(p.out, "────────────────────────────────────────────")
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(p.out)
		if err != nil {
			return err
		}
	}

	return nil
}

// printDetails contains the actual printing logic of package details.
func (p *PackagePrinter) printDetails(pkg packages.Package) error {
	if _, err := fmt.Fprintf(p.out, "  🆔 %s\n", pkg.ID); err != nil {
		return err
	}

	if strings.TrimSpace(pkg.DisplayName) != "" {
		if _, err := fmt.Fprintf(p.out, "  🏷️ Name: %s\n", pkg.Name); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintf(p.out, "  📁 Registry: %s\n", pkg.Source); err != nil {
		return err
	}

	if strings.TrimSpace(pkg.Description) != "" {
		if _, err := fmt.Fprintf(p.out, "  ℹ️ Description: %s\n", pkg.Description); err != nil {
			return err
		}
	}

	if strings.TrimSpace(pkg.License) != "" {
		if _, err := fmt.Fprintf(p.out, "  📄 License: %s\n", pkg.License); err != nil {
			return err
		}
	}

	if len(pkg.Runtimes) > 0 {
		runtimes := make([]string, len(pkg.Runtimes))
		for i, r := range pkg.Runtimes {
			runtimes[i] = string(r)
		}
		if _, err := fmt.Fprintf(p.out, "  🏗️ Runtimes: %s\n", strings.Join(runtimes, ", ")); err != nil {
			return err
		}
	} else if p.opts.showMissingWarning {
		if _, err := fmt.Fprintf(p.out, "  ⚠️ Warning: No supported runtimes found in package description\n"); err != nil {
			return err
		}
	}

	if len(pkg.Tools) > 0 {
		if _, err := fmt.Fprintf(p.out, "  🔨 Tools: %s\n", strings.Join(pkg.Tools, ", ")); err != nil {
			return err
		}
	} else if p.opts.showMissingWarning {
		if _, err := fmt.Fprintf(p.out, "  ⚠️ Warning: No tools found in package description\n"); err != nil {
			return err
		}
	}

	if len(pkg.Arguments) > 0 {
		if _, err := fmt.Fprintln(p.out, "  ⚙️ Found startup args..."); err != nil {
			return err
		}
		requiredArgs := getArgs(pkg.Arguments, true)
		if len(requiredArgs) > 0 {
			if _, err := fmt.Fprintf(p.out, "  ❗ Required: %s\n", strings.Join(requiredArgs, ", ")); err != nil {
				return err
			}
		}
		optionalArgs := getArgs(pkg.Arguments, false)
		if len(optionalArgs) > 0 {
			if _, err := fmt.Fprintf(p.out, "  🔹️ Optional: %s\n", strings.Join(optionalArgs, ", ")); err != nil {
				return err
			}
		}

		envVars := pkg.Arguments.EnvVars()
		envVarNames := envVars.EnvVarNames()
		if len(envVarNames) > 0 {
			if _, err := fmt.Fprintln(p.out, "  📋 Args configurable via environment variables..."); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(p.out, "  🌍 %s\n", strings.Join(envVarNames, ", ")); err != nil {
				return err
			}
		}

		args := pkg.Arguments.Args()
		argNames := args.ArgNames()
		if len(argNames) > 0 {
			if _, err := fmt.Fprintln(p.out, "  📋 Args configurable via command line..."); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(p.out, "  🖥️ %s\n", strings.Join(argNames, ", ")); err != nil {
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
