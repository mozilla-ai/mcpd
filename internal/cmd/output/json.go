package output

import (
	"encoding/json"
	"io"
	"strings"
)

// JSONHandler writes JSON for both data and errors, honoring struct tags.
type JSONHandler[T any] struct {
	out    io.Writer
	indent string
}

func NewJSONHandler[T any](w io.Writer, indentSpaces int) *JSONHandler[T] {
	return &JSONHandler[T]{
		w,
		strings.Repeat(" ", indentSpaces),
	}
}

// Writer returns the underlying io.Writer where JSON will be written.
func (h *JSONHandler[T]) Writer() io.Writer {
	return h.out
}

// HandleResults marshals the given slice of items under a "results" key to JSON.
func (h *JSONHandler[T]) HandleResults(items []T) error {
	payload := struct {
		Results []T `json:"results"`
	}{
		items,
	}

	enc := json.NewEncoder(h.out)
	enc.SetIndent("", h.indent)
	return enc.Encode(payload)
}

// HandleError marshals the given error string under an "error" key to JSON.
func (h *JSONHandler[T]) HandleError(err error) error {
	payload := struct {
		Error string `json:"error"`
	}{
		err.Error(),
	}

	enc := json.NewEncoder(h.out)
	enc.SetIndent("", h.indent)
	return enc.Encode(payload)
}
