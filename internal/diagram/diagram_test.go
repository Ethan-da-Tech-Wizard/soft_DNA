// Program: soft_DNA - diagram_test.go
// Programmer: Ethan-da-Tech-Wizard (written with Claude, an AI assistant)
// Last Modified: 2026-11-18
// Purpose: Tests that WriteMermaid produces a flowchart containing the root, stack branch and entry-point node for the sample Node project.

package diagram

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"codelens-local/internal/scanner"
	"codelens-local/internal/stackdetect"
	"codelens-local/internal/structure"
)

// TestWriteMermaid scans the sample Node fixture, writes the diagram to a temp folder, and checks for the expected Mermaid lines.
func TestWriteMermaid(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "sample_node")
	files, err := scanner.Scan(root, scanner.Options{})
	if err != nil {
		t.Fatal(err)
	}
	stack := stackdetect.Detect(files, root)
	folders, entries := structure.Analyze(files)

	path, err := WriteMermaid(t.TempDir(), stack, folders, entries)
	if err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got := string(content)
	for _, want := range []string{"flowchart TD", `root["Project Root"]`, "root --> stack", `entry_src_main_tsx["src/main.tsx"]`} {
		if !strings.Contains(got, want) {
			t.Fatalf("diagram missing %q:\n%s", want, got)
		}
	}
}
