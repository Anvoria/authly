package main

import (
	"fmt"
	"os"

	"github.com/Anvoria/authly/internal/cli"
	"github.com/Anvoria/authly/internal/cli/admin"
	"github.com/Anvoria/authly/internal/cli/keys"
)

func main() {
	registry := cli.NewRegistry()

	// Register commands
	registry.Register(&keys.Command{})
	registry.Register(&admin.Command{})

	// Run
	if err := registry.Run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
