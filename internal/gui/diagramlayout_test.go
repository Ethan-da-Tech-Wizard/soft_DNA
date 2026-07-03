package gui

import (
	"testing"

	"codelens-local/internal/scanner"
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

func TestComputeDiagramLayoutWrapping(t *testing.T) {
	// Create 12 folders. They should wrap into 3 rows of at most 5 columns:
	// Row 0: 5 items
	// Row 1: 5 items
	// Row 2: 2 items
	folders := make([]structure.Folder, 12)
	for i := 0; i < 12; i++ {
		folders[i] = structure.Folder{Path: "dir" + string(rune('a'+i)), Role: "Conventional"}
	}
	layout := ComputeDiagramLayout(stackdetect.Report{}, folders, nil)

	// Expect 13 nodes (root + 12 folders)
	if len(layout.Nodes) != 13 {
		t.Fatalf("len(Nodes) = %d, want 13", len(layout.Nodes))
	}

	// Verify coordinates for row/col placements
	// Node at index 1 is folders[0] (Row 0, Col 0)
	// Node at index 6 is folders[5] (Row 1, Col 0)
	// Node at index 11 is folders[10] (Row 2, Col 0)
	f0 := layout.Nodes[1]
	f5 := layout.Nodes[6]
	f10 := layout.Nodes[11]

	if f0.X != 0 || f0.Y != diagramVGap+diagramNodeH {
		t.Errorf("Node f0 position = (%v, %v), want (0, %v)", f0.X, f0.Y, diagramVGap+diagramNodeH)
	}
	if f5.X != 0 || f5.Y != diagramVGap+diagramNodeH+(diagramNodeH+diagramVGap) {
		t.Errorf("Node f5 position = (%v, %v), want (0, %v)", f5.X, f5.Y, diagramVGap+diagramNodeH+(diagramNodeH+diagramVGap))
	}
	if f10.X != 0 || f10.Y != diagramVGap+diagramNodeH+2*(diagramNodeH+diagramVGap) {
		t.Errorf("Node f10 position = (%v, %v), want (0, %v)", f10.X, f10.Y, diagramVGap+diagramNodeH+2*(diagramNodeH+diagramVGap))
	}

	// Verify root node is centered above the 5-column maximum width
	root := layout.Nodes[0]
	gridWidth := float32(5)*diagramNodeW + float32(4)*diagramHGap
	expectedRootX := (gridWidth - diagramNodeW) / 2
	if root.X != expectedRootX {
		t.Errorf("Root X = %v, want %v", root.X, expectedRootX)
	}
}

func TestComputeDiagramLayoutForScope(t *testing.T) {
	files := []scanner.FileInfo{
		{Path: "main.go"},
		{Path: "app/core.go"},
		{Path: "app/helper/helper.go"},
	}

	// 1. Root scope (empty string)
	layoutRoot := ComputeDiagramLayoutForScope(files, stackdetect.Report{}, nil, nil, "")
	if len(layoutRoot.Nodes) != 1 || layoutRoot.Nodes[0].ID != "root" {
		t.Fatalf("Root scope layout should fallback to top-level layout: %+v", layoutRoot.Nodes)
	}

	// 2. Drilled-down scope ("app")
	layoutApp := ComputeDiagramLayoutForScope(files, stackdetect.Report{}, nil, nil, "app")
	// Expected:
	// Parent node: folder_app
	// Children: folder_app/helper, file_app/core.go
	if len(layoutApp.Nodes) != 3 {
		t.Fatalf("len(Nodes) = %d, want 3: %+v", len(layoutApp.Nodes), layoutApp.Nodes)
	}

	hasHelper := false
	hasCore := false
	for _, n := range layoutApp.Nodes {
		if n.ID == "folder_app/helper" {
			hasHelper = true
		}
		if n.ID == "file_app/core.go" {
			hasCore = true
		}
	}
	if !hasHelper || !hasCore {
		t.Fatalf("missing expected children in drilled-down layout: %+v", layoutApp.Nodes)
	}

	// 3. Edges
	if len(layoutApp.Edges) != 2 {
		t.Fatalf("len(Edges) = %d, want 2: %+v", len(layoutApp.Edges), layoutApp.Edges)
	}
}
