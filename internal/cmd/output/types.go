package output

import "io"

type Handler[T any] interface {
	// Writer returns the io.Writer this Handler will write to.
	Writer() io.Writer

	// HandleResults takes *any* data and renders it accordingly.
	HandleResults(items []T) error

	// HandleError renders the error.
	HandleError(err error) error
}

// ListPrinter knows how to render a sequence of items of type T.
type ListPrinter[T any] interface {
	// Header is printed once if len(items)>0.
	Header(w io.Writer, count int)

	// Item prints one element.
	Item(w io.Writer, elem T) error

	// Footer is printed once after all items.
	Footer(w io.Writer, count int)
}
