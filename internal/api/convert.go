package api

type Convertible[T any] interface {
	// ToAPIType can be used to convert a wrapped domain type to an API-safe type.
	// It should be responsible for any normalization required to ensure consistency
	// across the API boundary.
	ToAPIType() (T, error)
}
