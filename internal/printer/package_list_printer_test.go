package printer

import (
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mozilla-ai/mcpd/v2/internal/packages"
)

// testPrinterInner records PrintPackage calls and optionally errors
type testPrinterInner struct {
	calledPackages []packages.Package
	errOnPackage   string
}

func (f *testPrinterInner) PrintPackage(pkg packages.Package) error {
	f.calledPackages = append(f.calledPackages, pkg)
	if pkg.Name == f.errOnPackage {
		return errors.New("print error")
	}
	return nil
}

func (f *testPrinterInner) SetOptions(_ ...PackagePrinterOption) error {
	return nil // no-op for testing
}

// dummy package for testing
func newPkg(name string) packages.Package {
	return packages.Package{Name: name}
}

func TestPackageListPrinter_Header(t *testing.T) {
	t.Parallel()

	buf := &bytes.Buffer{}
	printer := NewPackageListPrinter(&testPrinterInner{})
	printer.Header(buf, 5)

	out := buf.String()
	// should contain title and separator
	require.Contains(t, out, "🔎 Registry search results...")
	require.Contains(t, out, "────────────────────────────────────────────")
}

func TestPackageListPrinter_Item(t *testing.T) {
	t.Parallel()

	inner := &testPrinterInner{}
	printer := NewPackageListPrinter(inner)

	pkg := newPkg("testpkg")
	err := printer.Item(nil, pkg)
	require.NoError(t, err)
	require.Equal(t, []packages.Package{pkg}, inner.calledPackages)

	// error case
	inner = &testPrinterInner{errOnPackage: "badpkg"}
	printer = NewPackageListPrinter(inner)
	bad := newPkg("badpkg")
	err = printer.Item(nil, bad)
	require.EqualError(t, err, "print error")
}

func TestPackageListPrinter_Footer(t *testing.T) {
	t.Parallel()

	buf := &bytes.Buffer{}
	printer := NewPackageListPrinter(&testPrinterInner{})

	// singular
	printer.Footer(buf, 1)
	require.Contains(t, buf.String(), "📦 Found 1 package")

	buf.Reset()
	// plural
	printer.Footer(buf, 3)
	require.Contains(t, buf.String(), "📦 Found 3 packages")
}
