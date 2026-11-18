// ==============================================================================
// Program: main.go
// Programmer: Ethan-da-Tech-Wizard
// Last Modified: November 18, 2026
// Purpose: Command-line interface entry point for CodeLens Local. Reads process arguments, binds stdout/stderr streams, and routes execution to the internal CLI command runner.
// ==============================================================================

package main

import (
	"os"

	"codelens-local/internal/cli"
)

var version = "dev"

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr, version))
}
