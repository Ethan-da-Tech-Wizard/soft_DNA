package gui

import (
	"testing"

	"codelens-local/internal/stackdetect"
	"codelens-local/internal/structure"
)

func TestComputeDiagramLayoutRootAlone(t *testing.T) {
	layout := ComputeDiagramLayout(stackdetect.Report{}, nil, nil)
	if len(layout.Nodes) != 1 || layout.Nodes[0].ID != "root" {
		t.Fatalf("Nodes = %+v, want just the root node", layout.Nodes)
	}
	if len(layout.Edges) != 0 {
		t.Fatalf("Edges = %+v, want none", layout.Edges)
	}
}

func TestComputeDiagramLayoutFoldersEntriesAndStack(t *testing.T) {
	folders := []structure.Folder{{Path: ".", Role: "Root files"}, {Path: "cmd", Role: "Entry points"}}
	entries := []structure.EntryPoint{{Path: "main.go", Reason: "Go main"}}
	stack := stackdetect.Report{Tech: []stackdetect.Tech{{Name: "Go", Category: "language"}}}

	layout := ComputeDiagramLayout(stack, folders, entries)

	// root + 2 folders + 1 entry + 1 stack node + 1 tech node = 6
	if len(layout.Nodes) != 6 {
		t.Fatalf("Nodes = %d, want 6: %+v", len(layout.Nodes), layout.Nodes)
	}
	// root->folder x2, root->entry, root->stack, stack->tech = 5
	if len(layout.Edges) != 5 {
		t.Fatalf("Edges = %d, want 5: %+v", len(layout.Edges), layout.Edges)
	}

	var root, tech *DiagramNode
	for i := range layout.Nodes {
		switch layout.Nodes[i].ID {
		case "root":
			root = &layout.Nodes[i]
		case "tech_0_Go":
			tech = &layout.Nodes[i]
		}
	}
	if root == nil || tech == nil {
		t.Fatalf("missing expected node IDs: %+v", layout.Nodes)
	}
	if tech.Y <= root.Y {
		t.Fatalf("tech node Y=%v should be below root Y=%v", tech.Y, root.Y)
	}
	if layout.Width <= 0 || layout.Height <= 0 {
		t.Fatalf("Width/Height not computed: %+v", layout)
	}
}

func TestComputeDiagramLayoutNoDuplicateEdgesWithoutStack(t *testing.T) {
	layout := ComputeDiagramLayout(stackdetect.Report{}, []structure.Folder{{Path: "src", Role: "Source"}}, nil)
	for _, e := range layout.Edges {
		if e.To == "stack" || e.From == "stack" {
			t.Fatalf("edge referencing stack node when there is no stack: %+v", e)
		}
	}
}
