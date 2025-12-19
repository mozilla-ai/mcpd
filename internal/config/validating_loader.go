package config

import (
	"fmt"
	"reflect"
)

// ValidationPredicate evaluates a loaded Config and returns an error if invalid.
type ValidationPredicate func(*Config) error

// ValidatingLoader wraps a Loader to run additional validation predicates at load time.
// Uses decorator pattern to preserve custom loader implementations while adding validation.
type ValidatingLoader struct {
	Loader
	predicates []ValidationPredicate
}

// NewValidatingLoader creates a loader that runs validation predicates after Load().
func NewValidatingLoader(inner Loader, predicates ...ValidationPredicate) (*ValidatingLoader, error) {
	if inner == nil || reflect.ValueOf(inner).IsNil() {
		return nil, fmt.Errorf("loader cannot be nil")
	}

	return &ValidatingLoader{
		Loader:     inner,
		predicates: predicates,
	}, nil
}

// Load delegates to inner loader, then runs validation predicates.
func (l *ValidatingLoader) Load(path string) (Modifier, error) {
	mod, err := l.Loader.Load(path)
	if err != nil {
		return nil, err
	}

	cfg, ok := mod.(*Config)
	if !ok {
		return nil, fmt.Errorf("loader returned unexpected type (internal error)")
	}

	for _, predicate := range l.predicates {
		if err := predicate(cfg); err != nil {
			return nil, err
		}
	}

	return cfg, nil
}
