package options

import (
	"github.com/mozilla-ai/mcpd/v2/internal/filter"
	"github.com/mozilla-ai/mcpd/v2/internal/runtime"
)

type ResolveOption func(*ResolveOptions) error

type ResolveOptions struct {
	Runtime runtime.Runtime
	Source  string
	Version string
}

func defaultResolveOptions() ResolveOptions {
	return ResolveOptions{}
}

func NewResolveOptions(opt ...ResolveOption) (ResolveOptions, error) {
	opts := defaultResolveOptions()

	for _, opt := range opt {
		if err := opt(&opts); err != nil {
			return ResolveOptions{}, err
		}
	}
	return opts, nil
}

func ResolveFilters(opts ResolveOptions) map[string]string {
	f := map[string]string{}

	if opts.Runtime != "" {
		f[FilterKeyRuntime] = string(opts.Runtime)
	}

	if opts.Version != "" {
		f[FilterKeyVersion] = opts.Version
	}

	return f
}

func WithResolveRuntime(runtime runtime.Runtime) ResolveOption {
	return func(o *ResolveOptions) error {
		o.Runtime = runtime
		return nil
	}
}

func WithResolveVersion(version string) ResolveOption {
	return func(o *ResolveOptions) error {
		o.Version = filter.NormalizeString(version)
		return nil
	}
}

func WithResolveSource(source string) ResolveOption {
	return func(o *ResolveOptions) error {
		o.Source = filter.NormalizeString(source)
		return nil
	}
}
