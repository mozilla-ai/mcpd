package output

import (
	"io"
)

type TextHandler[T any] struct {
	out            io.Writer
	resultsPrinter Printer[T]
}

func NewTextHandler[T any](w io.Writer, p Printer[T]) *TextHandler[T] {
	return &TextHandler[T]{
		out:            w,
		resultsPrinter: p,
	}
}

// Writer returns the underlying io.Writer where text will be written.
func (h *TextHandler[T]) Writer() io.Writer {
	return h.out
}

func (h *TextHandler[T]) HandleResult(item T) error {
	return h.resultsPrinter.Item(h.out, item)
}

func (h *TextHandler[T]) HandleResults(items ...T) error {
	if len(items) == 0 {
		_, _ = io.WriteString(h.out, "No items found\n")
		return nil
	}

	h.resultsPrinter.Header(h.out, len(items))

	for _, it := range items {
		if err := h.resultsPrinter.Item(h.out, it); err != nil {
			return err
		}
	}

	h.resultsPrinter.Footer(h.out, len(items))

	return nil
}

func (h *TextHandler[T]) HandleError(err error) error {
	return err
}
