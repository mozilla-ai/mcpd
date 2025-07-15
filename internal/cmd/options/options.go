package options

import (
	"github.com/mozilla-ai/mcpd/v2/internal/cmd"
	"github.com/mozilla-ai/mcpd/v2/internal/config"
	"github.com/mozilla-ai/mcpd/v2/internal/context"
	"github.com/mozilla-ai/mcpd/v2/internal/printer"
	"github.com/mozilla-ai/mcpd/v2/internal/registry"
)

type CmdOption func(*CmdOptions) error

type CmdOptions struct {
	ConfigLoader      config.Loader
	ConfigInitializer config.Initializer
	ContextLoader     context.Loader
	Printer           printer.Printer
	RegistryBuilder   registry.Builder
}

func defaultOptions() CmdOptions {
	configLoader := &config.DefaultLoader{}
	return CmdOptions{
		ConfigLoader:      configLoader,
		ConfigInitializer: configLoader,
		ContextLoader:     &context.DefaultLoader{},
		Printer:           &printer.DefaultPrinter{},
		RegistryBuilder:   &cmd.BaseCmd{},
	}
}

func NewOptions(opt ...CmdOption) (CmdOptions, error) {
	opts := defaultOptions()

	for _, o := range opt {
		if o == nil {
			continue
		}
		if err := o(&opts); err != nil {
			return CmdOptions{}, err
		}
	}
	return opts, nil
}

func WithConfigLoader(l config.Loader) CmdOption {
	return func(o *CmdOptions) error {
		o.ConfigLoader = l
		return nil
	}
}

func WithContextLoader(l context.Loader) CmdOption {
	return func(o *CmdOptions) error {
		o.ContextLoader = l
		return nil
	}
}

func WithPrinter(p printer.Printer) CmdOption {
	return func(o *CmdOptions) error {
		o.Printer = p
		return nil
	}
}

func WithRegistryBuilder(b registry.Builder) CmdOption {
	return func(o *CmdOptions) error {
		o.RegistryBuilder = b
		return nil
	}
}
