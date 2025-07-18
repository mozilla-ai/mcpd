package main

import (
	"fmt"
	"os"

	"github.com/mozilla-ai/mcpd/v2/cmd"
)

func main() {
	// Execute the root command.
	if err := cmd.Execute(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	}
}
