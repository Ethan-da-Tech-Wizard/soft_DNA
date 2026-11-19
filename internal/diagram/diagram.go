// ==============================================================================
// Program: diagram.go
// Programmer: Ethan-da-Tech-Wizard
// Last Modified: November 18, 2026
// Purpose: Writes a Mermaid flowchart (architecture.mmd) that summarises a scanned project: its
// top-level folders, likely entry points and detected technology stack.
//
// How it fits the app: the CLI (internal/cli) runs scanner -> stackdetect -> structure.Analyze, then
// calls WriteMermaid with those results. The .mmd file is plain text that any Mermaid renderer can
// display; this package does no rendering itself (the GUI draws its own diagram, see
// internal/gui/diagramlayout.go, mirroring this structure). Only Mermaid is produced here; there is no Graphviz or ASCII output.
// Inputs: output directory, stackdetect.Report, folders and entry points. Output: the file path.
// Known limit: the diagram is a flat star (root -> everything); it shows no import/dependency edges.
// ==============================================================================

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

// fileName is the fixed output name, so repeated runs overwrite the previous diagram in outDir.
const fileName = "architecture.mmd"

// nodeRe matches every character that is not legal in a Mermaid node id; such characters are
// replaced by "_" in nodeID.
var nodeRe = regexp.MustCompile(`[^A-Za-z0-9_]`)

// WriteMermaid writes architecture.mmd into outDir and returns its path.
//
// stack, folders and entries come from stackdetect.Detect and structure.Analyze. The output is a
// top-down flowchart ("flowchart TD") with a single root node that fans out to folder nodes,
// entry-point nodes and, when any technology was detected, a "Detected Stack" node with one child
// per technology. outDir must already exist; the file is written with mode 0644 (it contains only
// project structure, not secrets). The error is the os.WriteFile error, if any.
func WriteMermaid(outDir string, stack stackdetect.Report, folders []structure.Folder, entries []structure.EntryPoint) (string, error) {
	// Build the whole diagram in memory and write once, so a failure never leaves a half-written file.
	var b strings.Builder
	b.WriteString("flowchart TD\n")
	b.WriteString(`    root["Project Root"]` + "\n")

	// One node per top-level folder, labelled "path: role". The "folder_" prefix keeps ids from
	// colliding with entry_/tech_ ids even when the underlying names are equal.
	for _, folder := range folders {
		id := "folder_" + nodeID(folder.Path)
		fmt.Fprintf(&b, `    %s["%s: %s"]`+"\n", id, escape(folder.Path), escape(folder.Role))
		fmt.Fprintf(&b, "    root --> %s\n", id)
	}

	// Entry points hang directly off the root so they are easy to spot, regardless of which folder holds them.
	for _, entry := range entries {
		id := "entry_" + nodeID(entry.Path)
		fmt.Fprintf(&b, `    %s["%s"]`+"\n", id, escape(entry.Path))
		b.WriteString("    root --> " + id + "\n")
	}

	// Skip the stack branch entirely when nothing was detected, to avoid an empty, confusing node.
	if len(stack.Tech) > 0 {
		b.WriteString(`    stack["Detected Stack"]` + "\n")
		b.WriteString("    root --> stack\n")
		for _, tech := range stack.Tech {
			// The id includes the evidence file so the same technology found in two places gets two nodes.
			id := "tech_" + nodeID(tech.Category+"_"+tech.Name+"_"+tech.EvidenceFile)
			fmt.Fprintf(&b, `    %s["%s"]`+"\n", id, escape(tech.Name))
			fmt.Fprintf(&b, "    stack --> %s\n", id)
		}
	}

	// 0o644: readable by others on purpose so the diagram can be shared or committed.
	path := filepath.Join(outDir, fileName)
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// nodeID turns an arbitrary path or name into a valid Mermaid node id.
// Illegal characters become "_", leading/trailing "_" are trimmed, and an empty or "." input (the
// repository root folder) maps to the fixed id "root_files".
func nodeID(value string) string {
	value = nodeRe.ReplaceAllString(value, "_")
	value = strings.Trim(value, "_")
	if value == "" || value == "." {
		return "root_files"
	}
	return value
}

// escape makes a label safe inside a double-quoted Mermaid string by escaping embedded quotes.
// Other special characters are left as they are, which Mermaid accepts inside quotes.
func escape(value string) string {
	value = strings.ReplaceAll(value, `"`, `\"`)
	return value
}
