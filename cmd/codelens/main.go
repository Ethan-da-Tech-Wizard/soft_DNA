// Program: soft_DNA - main.go
// Programmer: Ethan-da-Tech-Wizard (written with Claude, an AI assistant)
// Last Modified: 2026-11-18
// Purpose: Entry point for the codelens command-line binary; it forwards arguments to the cli package and exits with its status code.

package main

import (
	"os"

	"codelens-local/internal/cli"
)

// version is the release string reported by --version; it defaults to "dev" and can be overridden at build time.
var version = "dev"

// main runs the CLI with the real arguments and streams, then exits with the returned status code.
func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr, version))
}
