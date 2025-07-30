package output

import (
	"io"

	"gopkg.in/yaml.v3"
)

// YAMLHandler writes YAML for both data and errors, honoring struct tags.
// It wraps a slice of T in a top-level `results` field, or an `error` field on failure.
// Indentation is configurable (default is 2 spaces if using NewYAMLHandler without args).
type YAMLHandler[T any] struct {
	out    io.Writer
	indent int
}

// NewYAMLHandler constructs a new YAMLHandler for items of type T.
// indentSpaces controls the number of spaces to indent nested nodes.
func NewYAMLHandler[T any](w io.Writer, indentSpaces int) *YAMLHandler[T] {
	return &YAMLHandler[T]{
		out:    w,
		indent: indentSpaces,
	}
}

// Writer returns the underlying io.Writer where YAML will be written.
func (h *YAMLHandler[T]) Writer() io.Writer {
	return h.out
}

// HandleResult marshals the given item under a "result" key to YAML.
func (h *YAMLHandler[T]) HandleResult(item T) error {
	payload := ResultPayload[T]{Result: item}
	enc := yaml.NewEncoder(h.out)
	defer func(enc *yaml.Encoder) {
		_ = enc.Close()
	}(enc)
	enc.SetIndent(h.indent)
	return enc.Encode(payload)
}

// HandleResults marshals the given slice of items under a "results" key to YAML.
func (h *YAMLHandler[T]) HandleResults(items ...T) error {
	payload := ResultsPayload[T]{Results: items}
	enc := yaml.NewEncoder(h.out)
	defer func(enc *yaml.Encoder) {
		_ = enc.Close()
	}(enc)
	enc.SetIndent(h.indent)
	return enc.Encode(payload)
}

// HandleError marshals the given error string under an "error" key to YAML.
func (h *YAMLHandler[T]) HandleError(err error) error {
	payload := ErrorPayload{Error: err.Error()}

	enc := yaml.NewEncoder(h.out)
	defer func(enc *yaml.Encoder) {
		// Ensure encoder is closed to flush any buffered data.
		_ = enc.Close()
	}(enc)

	enc.SetIndent(h.indent)
	return enc.Encode(payload)
}
