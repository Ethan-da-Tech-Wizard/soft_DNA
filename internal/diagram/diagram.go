// Package diagram writes a simple Mermaid architecture diagram.
package diagram

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"codelens-local/internal/stackdetect"
	"codelens-local/internal/structure"
)

const fileName = "architecture.mmd"

var nodeRe = regexp.MustCompile(`[^A-Za-z0-9_]`)

// WriteMermaid writes architecture.mmd and returns its path.
func WriteMermaid(outDir string, stack stackdetect.Report, folders []structure.Folder, entries []structure.EntryPoint) (string, error) {
	var b strings.Builder
	b.WriteString("flowchart TD\n")
	b.WriteString(`    root["Project Root"]` + "\n")

	for _, folder := range folders {
		id := "folder_" + nodeID(folder.Path)
		fmt.Fprintf(&b, `    %s["%s: %s"]`+"\n", id, escape(folder.Path), escape(folder.Role))
		fmt.Fprintf(&b, "    root --> %s\n", id)
	}

	for _, entry := range entries {
		id := "entry_" + nodeID(entry.Path)
		fmt.Fprintf(&b, `    %s["%s"]`+"\n", id, escape(entry.Path))
		b.WriteString("    root --> " + id + "\n")
	}

	if len(stack.Tech) > 0 {
		b.WriteString(`    stack["Detected Stack"]` + "\n")
		b.WriteString("    root --> stack\n")
		for _, tech := range stack.Tech {
			id := "tech_" + nodeID(tech.Category+"_"+tech.Name+"_"+tech.EvidenceFile)
			fmt.Fprintf(&b, `    %s["%s"]`+"\n", id, escape(tech.Name))
			fmt.Fprintf(&b, "    stack --> %s\n", id)
		}
	}

	path := filepath.Join(outDir, fileName)
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

func nodeID(value string) string {
	value = nodeRe.ReplaceAllString(value, "_")
	value = strings.Trim(value, "_")
	if value == "" || value == "." {
		return "root_files"
	}
	return value
}

func escape(value string) string {
	value = strings.ReplaceAll(value, `"`, `\"`)
	return value
}
