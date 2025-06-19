package printer

type PackagePrinterOptions struct {
	showHeader         bool
	showSeparator      bool
	showMissingWarning bool
}

type PackagePrinterOption func(*PackagePrinterOptions) error

func defaultPackagePrinterOptions() PackagePrinterOptions {
	return PackagePrinterOptions{
		showHeader:         false,
		showSeparator:      false,
		showMissingWarning: true,
	}
}

func NewPackagePrinterOptions(opts ...PackagePrinterOption) (PackagePrinterOptions, error) {
	options := defaultPackagePrinterOptions()
	for _, opt := range opts {
		if err := opt(&options); err != nil {
			return PackagePrinterOptions{}, err
		}
	}
	return options, nil
}

func WithHeader(enabled bool) PackagePrinterOption {
	return func(o *PackagePrinterOptions) error {
		o.showHeader = enabled
		return nil
	}
}

func WithSeparator(enabled bool) PackagePrinterOption {
	return func(o *PackagePrinterOptions) error {
		o.showSeparator = enabled
		return nil
	}
}

func WithMissingWarnings(enabled bool) PackagePrinterOption {
	return func(o *PackagePrinterOptions) error {
		o.showMissingWarning = enabled
		return nil
	}
}
