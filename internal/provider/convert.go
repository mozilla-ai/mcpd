package provider

type Convertible[T any] interface {
	ToDomainType() (T, error)
}
