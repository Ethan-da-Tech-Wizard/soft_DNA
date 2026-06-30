package main

import (
	"os/exec"
	"strings"
	"testing"
)

func TestVersionFlagEndToEnd(t *testing.T) {
	cmd := exec.Command("go", "run", ".", "--version")

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go run --version failed: %v\n%s", err, out)
	}

	got := strings.TrimSpace(string(out))
	if got != "codelens dev" {
		t.Fatalf("version output = %q, want %q", got, "codelens dev")
	}
}
