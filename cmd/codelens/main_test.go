// Program: soft_DNA - main_test.go
// Programmer: Ethan-da-Tech-Wizard (written with Claude, an AI assistant)
// Last Modified: 2026-11-18
// Purpose: End-to-end test that builds and runs the CLI with --version and checks the printed version string.

package main

import (
	"os/exec"
	"strings"
	"testing"
)

// TestVersionFlagEndToEnd runs the real command through `go run` and checks the --version output.
func TestVersionFlagEndToEnd(t *testing.T) {
	cmd := exec.Command("go", "run", ".", "--version")

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go run --version failed: %v\n%s", err, out)
	}

	// Trim the trailing newline so the comparison is exact.
	got := strings.TrimSpace(string(out))
	if got != "codelens dev" {
		t.Fatalf("version output = %q, want %q", got, "codelens dev")
	}
}
