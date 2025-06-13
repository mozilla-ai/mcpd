package types

import "strings"

type GetterOption func(*GetterOptions) error

type GetterOptions struct {
	Runtime Runtime
	Tools   []string
	Version string
	// TODO: add more later...
}

func defaultGetterOptions() GetterOptions {
	return GetterOptions{
		Version: "latest",
	}
}

func GetGetterOpts(opts ...GetterOption) (GetterOptions, error) {
	options := defaultGetterOptions()
	for _, opt := range opts {
		if err := opt(&options); err != nil {
			return GetterOptions{}, err
		}
	}
	return options, nil
}

func WithRuntime(runtime Runtime) GetterOption {
	return func(o *GetterOptions) error {
		o.Runtime = runtime
		return nil
	}
}

func WithTools(tools ...string) GetterOption {
	return func(o *GetterOptions) error {
		for _, tool := range tools {
			o.Tools = append(o.Tools, strings.TrimSpace(tool))
		}
		return nil
	}
}

func WithVersion(version string) GetterOption {
	return func(o *GetterOptions) error {
		o.Version = strings.TrimSpace(version)
		return nil
	}
}
