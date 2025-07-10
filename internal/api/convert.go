package api

type Convertible[T any] interface {
	ToAPIType() (T, error)
}
