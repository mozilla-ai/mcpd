package output

import "io"

type Handler[T any] interface {
	// Writer returns the io.Writer this Handler will write to.
	Writer() io.Writer

	// HandleResult takes *any* data and renders it accordingly.
	HandleResult(item T) error

	// HandleResults takes *any* collection of data and renders it accordingly.
	HandleResults(items ...T) error

	// HandleError renders the error.
	HandleError(err error) error
}

// WriteFunc is a generic function type used for writing output related to
// a collection of items of type T. It is typically used for writing headers
// or footers in formatted output.
//
// The function receives an io.Writer to write to, and the total count of
// items being printed. It does not receive or operate on individual items.
type WriteFunc[T any] func(w io.Writer, count int)

type Printer[T any] interface {
	// Header should be called once before the Item.
	Header(w io.Writer, count int)

	// SetHeader can be used to configure the Header function.
	SetHeader(fn WriteFunc[T])

	// Item prints one element.
	Item(w io.Writer, elem T) error

	// Footer should be called once after the Item.
	Footer(w io.Writer, count int)

	// SetFooter can be used to configure the Footer function.
	SetFooter(fn WriteFunc[T])
}

// ResultsPayload is a generic wrapper for multiple result values.
// It is intended for use in API responses that return a list of items.
// The payload is serialized with the key "results".
type ResultsPayload[T any] struct {
	Results []T `json:"results" yaml:"results"`
}

// ResultPayload is a generic wrapper for a single result value.
// It is intended for use in API responses that return one item.
// The payload is serialized with the key "result".
type ResultPayload[T any] struct {
	Result T `json:"result" yaml:"result"`
}

// ErrorPayload represents an error message returned by an API.
// The payload is serialized with the key "error".
type ErrorPayload struct {
	Error string `json:"error" yaml:"error"`
}
