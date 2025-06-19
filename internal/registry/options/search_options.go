package options

import (
	"github.com/mozilla-ai/mcpd/v2/internal/filter"
)

type SearchOption func(*SearchOptions) error

type SearchOptions struct {
	Source string
}

func defaultSearchOptions() SearchOptions {
	return SearchOptions{}
}

func NewSearchOptions(opt ...SearchOption) (SearchOptions, error) {
	opts := defaultSearchOptions()

	for _, opt := range opt {
		if err := opt(&opts); err != nil {
			return SearchOptions{}, err
		}
	}
	return opts, nil
}

func WithSearchSource(source string) SearchOption {
	return func(o *SearchOptions) error {
		o.Source = filter.NormalizeString(source)
		return nil
	}
}
