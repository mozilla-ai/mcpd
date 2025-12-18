package config

import "fmt"

// ValidationPredicate evaluates a loaded Config and returns an error if invalid.
type ValidationPredicate func(*Config) error

// validatingLoader wraps a Loader to run additional validation predicates at load time.
// Uses decorator pattern to preserve custom loader implementations while adding validation.
type validatingLoader struct {
	Loader
	predicates []ValidationPredicate
}

// NewValidatingLoader creates a loader that runs validation predicates after Load().
func NewValidatingLoader(inner Loader, predicates ...ValidationPredicate) *validatingLoader {
	return &validatingLoader{
		Loader:     inner,
		predicates: predicates,
	}
}

// Load delegates to inner loader, then runs validation predicates.
func (l *validatingLoader) Load(path string) (Modifier, error) {
	mod, err := l.Loader.Load(path)
	if err != nil {
		return nil, err
	}

	cfg, ok := mod.(*Config)
	if !ok {
		return nil, fmt.Errorf("invalid config structure")
	}

	for _, predicate := range l.predicates {
		if err := predicate(cfg); err != nil {
			return nil, err
		}
	}

	return cfg, nil
}
