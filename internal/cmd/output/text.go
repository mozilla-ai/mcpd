package output

import (
	"io"
)

type TextHandler[T any] struct {
	out     io.Writer
	printer ListPrinter[T]
}

func NewTextHandler[T any](w io.Writer, p ListPrinter[T]) *TextHandler[T] {
	return &TextHandler[T]{
		out:     w,
		printer: p,
	}
}

// Writer returns the underlying io.Writer where text will be written.
func (h *TextHandler[T]) Writer() io.Writer {
	return h.out
}

func (h *TextHandler[T]) HandleResults(items []T) error {
	if len(items) == 0 {
		_, _ = io.WriteString(h.out, "No items found\n")
		return nil
	}

	h.printer.Header(h.out, len(items))

	for _, it := range items {
		if err := h.printer.Item(h.out, it); err != nil {
			return err
		}
	}

	h.printer.Footer(h.out, len(items))

	return nil
}

func (h *TextHandler[T]) HandleError(err error) error {
	return err
}
