package registry

import (
	"fmt"

	"github.com/mozilla-ai/mcpd-cli/v2/internal/registry/types"
)

type Option func(*options) error

type options struct {
	supportedRuntimes map[types.Runtime]struct{}
}

func getDefaultOptions() options {
	return options{
		supportedRuntimes: DefaultSupportedRuntimes(),
	}
}

func getOpts(opts ...Option) (options, error) {
	opt := getDefaultOptions()
	for _, o := range opts {
		if o == nil {
			continue
		}
		if err := o(&opt); err != nil {
			return options{}, err
		}
	}
	return opt, nil
}

func DefaultSupportedRuntimes() map[types.Runtime]struct{} {
	return map[types.Runtime]struct{}{
		types.RuntimeNpx: {},
		types.RuntimeUvx: {},
	}
}

func WithSupportedRuntimes(runtimes ...types.Runtime) Option {
	return func(o *options) error {
		if len(runtimes) == 0 {
			return fmt.Errorf("must specify at least one supported runtime")
		}
		o.supportedRuntimes = make(map[types.Runtime]struct{}, len(runtimes))
		for _, rt := range runtimes {
			o.supportedRuntimes[rt] = struct{}{}
		}
		return nil
	}
}
