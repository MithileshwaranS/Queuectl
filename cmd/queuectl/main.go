package main

import (
	"fmt"
	"os"

	"github.com/MithileshwaranS/queuectl/internal/config"
	"github.com/MithileshwaranS/queuectl/pkg/cli"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to load config: %v\n", err)
		fmt.Fprintf(os.Stderr, "Using default configuration\n")
		cfg = config.DefaultConfig()
	}

	// Execute CLI
	if err := cli.Execute(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
