package output

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

// fakePrinter simulates ListPrinter behavior with error injection by matching an item.
type fakePrinter[T comparable] struct {
	headerCalled bool
	headerCount  int
	items        []T
	footerCalled bool
	footerCount  int
	errOnItem    T
}

func (p *fakePrinter[T]) Header(w io.Writer, count int) {
	p.headerCalled = true
	p.headerCount = count
	_, _ = w.Write([]byte("HEADER\n"))
}

func (p *fakePrinter[T]) SetHeader(_ WriteFunc[T]) {}

func (p *fakePrinter[T]) SetFooter(_ WriteFunc[T]) {}

func (p *fakePrinter[T]) Item(w io.Writer, t T) error {
	p.items = append(p.items, t)

	if _, err := w.Write([]byte("ITEM:")); err != nil {
		return err
	}

	if _, err := fmt.Fprint(w, t); err != nil {
		return err
	}

	if _, err := w.Write([]byte("\n")); err != nil {
		return err
	}

	if t == p.errOnItem {
		return errors.New("item error")
	}

	return nil
}

func (p *fakePrinter[T]) Footer(w io.Writer, count int) {
	p.footerCalled = true
	p.footerCount = count
	if _, err := w.Write([]byte("FOOTER\n")); err != nil {
		panic(err)
	}
}

func TestNewTextHandler_Writer(t *testing.T) {
	t.Parallel()

	buf := &bytes.Buffer{}
	printer := &fakePrinter[string]{}
	h := NewTextHandler[string](buf, printer)
	require.Equal(t, buf, h.Writer())
}

func TestHandleResults_Empty(t *testing.T) {
	t.Parallel()

	buf := &bytes.Buffer{}
	printer := &fakePrinter[int]{}
	h := NewTextHandler[int](buf, printer)

	err := h.HandleResults([]int{}...)
	require.NoError(t, err)
	require.False(t, printer.headerCalled)
	require.False(t, printer.footerCalled)
	require.Equal(t, "No items found\n", buf.String())
}

func TestHandleResults_WithItems(t *testing.T) {
	t.Parallel()
	buf := &bytes.Buffer{}
	printer := &fakePrinter[string]{}
	h := NewTextHandler[string](buf, printer)

	items := []string{"a", "b", "c"}
	err := h.HandleResults(items...)
	require.NoError(t, err)

	require.True(t, printer.headerCalled)
	require.Equal(t, len(items), printer.headerCount)
	require.Equal(t, items, printer.items)
	require.True(t, printer.footerCalled)
	require.Equal(t, len(items), printer.footerCount)

	expected := "HEADER\n"
	for _, it := range items {
		expected += "ITEM:" + it + "\n"
	}
	expected += "FOOTER\n"
	require.Equal(t, expected, buf.String())
}

func TestHandleResults_ItemError(t *testing.T) {
	t.Parallel()

	buf := &bytes.Buffer{}
	printer := &fakePrinter[int]{errOnItem: 2}
	h := NewTextHandler[int](buf, printer)

	err := h.HandleResults([]int{1, 2, 3}...)
	require.Error(t, err)
	require.Contains(t, err.Error(), "item error")

	require.True(t, printer.headerCalled)
	require.Equal(t, []int{1, 2}, printer.items)
	require.False(t, printer.footerCalled)
}

func TestHandleError(t *testing.T) {
	t.Parallel()

	printer := &fakePrinter[string]{}
	h := NewTextHandler[string](nil, printer)

	testErr := errors.New("test failure")
	err := h.HandleError(testErr)
	require.EqualError(t, err, "test failure")
}
