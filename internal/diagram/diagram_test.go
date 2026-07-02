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
