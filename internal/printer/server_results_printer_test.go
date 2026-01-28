package printer

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mozilla-ai/mcpd/internal/cmd/output"
	"github.com/mozilla-ai/mcpd/internal/packages"
)

// testPrinterInner records PrintPackage calls and optionally errors
type testPrinterInner struct {
	calledPackages []packages.Server
	errOnPackage   string
}

func (f *testPrinterInner) Header(_ io.Writer, _ int) {}

func (f *testPrinterInner) SetHeader(_ output.WriteFunc[packages.Server]) {}

func (f *testPrinterInner) Item(_ io.Writer, pkg packages.Server) error {
	f.calledPackages = append(f.calledPackages, pkg)
	if pkg.Name == f.errOnPackage {
		return errors.New("print error")
	}
	return nil
}

func (f *testPrinterInner) Footer(_ io.Writer, _ int) {}

func (f *testPrinterInner) SetFooter(_ output.WriteFunc[packages.Server]) {}

// dummy package for testing
func newPkg(name string) packages.Server {
	return packages.Server{Name: name}
}

func TestPackageListPrinter_Header(t *testing.T) {
	t.Parallel()

	buf := &bytes.Buffer{}
	printer := NewServerResultsPrinter(&testPrinterInner{})
	printer.Header(buf, 5)

	out := buf.String()
	// should contain title and separator
	require.Contains(t, out, "🔎 Registry search results...")
	require.Contains(t, out, "────────────────────────────────────────────")
}

func TestPackageListPrinter_Item(t *testing.T) {
	t.Parallel()

	inner := &testPrinterInner{}
	printer := NewServerResultsPrinter(inner)

	pkg := newPkg("testpkg")
	err := printer.Item(nil, pkg)
	require.NoError(t, err)
	require.Equal(t, []packages.Server{pkg}, inner.calledPackages)

	// error case
	inner = &testPrinterInner{errOnPackage: "badpkg"}
	printer = NewServerResultsPrinter(inner)
	bad := newPkg("badpkg")
	err = printer.Item(nil, bad)
	require.EqualError(t, err, "print error")
}

func TestPackageListPrinter_Footer(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		count    int
		expected string
	}{
		{name: "zero", count: 0, expected: "📦 Found 0 servers\n"},
		{name: "singular", count: 1, expected: "📦 Found 1 server\n"},
		{name: "plural", count: 2, expected: "📦 Found 2 servers\n"},
		{name: "many", count: 10, expected: "📦 Found 10 servers\n"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			buf := &bytes.Buffer{}
			printer := NewServerResultsPrinter(&testPrinterInner{})
			printer.Footer(buf, tc.count)
			require.Contains(t, buf.String(), tc.expected)
		})
	}
}
