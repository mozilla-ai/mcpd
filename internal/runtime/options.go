package runtime

import (
	"fmt"
)

type Options struct {
	SupportedRuntimes map[Runtime]struct{}
}

type Option func(*Options) error

func defaultOptions() Options {
	return Options{
		SupportedRuntimes: DefaultSupportedRuntimes(),
	}
}

func NewOptions(opt ...Option) (Options, error) {
	opts := defaultOptions()

	for _, o := range opt {
		if o == nil {
			continue
		}
		if err := o(&opts); err != nil {
			return Options{}, err
		}
	}
	return opts, nil
}

func DefaultSupportedRuntimes() map[Runtime]struct{} {
	return map[Runtime]struct{}{
		NPX: {},
		UVX: {},
	}
}

// IsSupported reports whether the runtime is among the allowed set.
func (o Options) IsSupported(rt Runtime) bool {
	_, ok := o.SupportedRuntimes[rt]
	return ok
}

func WithSupportedRuntimes(runtimes ...Runtime) Option {
	return func(o *Options) error {
		if len(runtimes) == 0 {
			return fmt.Errorf("must specify at least one supported runtime")
		}
		o.SupportedRuntimes = make(map[Runtime]struct{}, len(runtimes))
		for _, rt := range runtimes {
			o.SupportedRuntimes[rt] = struct{}{}
		}
		return nil
	}
}
