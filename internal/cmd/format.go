package cmd

import (
	"fmt"
	"slices"
	"strings"
)

type OutputFormat string

type OutputFormats []OutputFormat

const (
	FormatJSON OutputFormat = "json"
	FormatYAML OutputFormat = "yaml"
	FormatText OutputFormat = "text"
)

func AllowedOutputFormats() OutputFormats {
	formats := []OutputFormat{
		FormatJSON,
		FormatText,
		FormatYAML,
	}

	slices.Sort(formats)

	return formats
}

// String implements fmt.Stringer for a collection of export formats,
// converting them to a comma separated string.
func (f *OutputFormats) String() string {
	efs := *f
	out := make([]string, len(efs))
	for i := range efs {
		out[i] = efs[i].String()
	}
	return strings.Join(out, ", ")
}

// String implements fmt.Stringer for an export format.
// This is also required by Cobra as part of implementing flag.Value.
func (f *OutputFormat) String() string {
	return strings.ToLower(string(*f))
}

// Set is used by Cobra to set the export format value from a string.
// This is also required by Cobra as part of implementing flag.Value.
func (f *OutputFormat) Set(v string) error {
	v = strings.ToLower(strings.TrimSpace(v))
	allowed := AllowedOutputFormats()

	for _, a := range allowed {
		if string(a) == v {
			*f = OutputFormat(v)
			return nil
		}
	}

	return fmt.Errorf("invalid format '%s', must be one of %v", v, allowed.String())
}

// Type is used by Cobra to get the 'type' of an export format for display purposes.
// This is also required by Cobra as part of implementing flag.Value.
func (f *OutputFormat) Type() string {
	return "format"
}
