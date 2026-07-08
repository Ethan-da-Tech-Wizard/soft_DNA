package gui

import (
	"path/filepath"
	"strings"
	"testing"

	"codelens-local/internal/cli"
	"codelens-local/internal/scanner"
	"codelens-local/internal/stackdetect"
	"codelens-local/internal/structure"
	"codelens-local/internal/symbols"
	"codelens-local/internal/validate"
)

func sampleResult() cli.PipelineResult {
	return cli.PipelineResult{
		Root:   "/home/user/myproject",
		OutDir: "/home/user/myproject/codelens-out",
		Files: []scanner.FileInfo{
			{Path: "main.go"},
			{Path: "app/server.go"},
			{Path: "app/handlers.go"},
			{Path: "vendor/lib.go", Skipped: true},
		},
		Stack: stackdetect.Report{
			Summary: "Go module project.",
			Tech:    []stackdetect.Tech{{Name: "Go", Category: "language", EvidenceFile: "go.mod"}},
		},
		Folders: []structure.Folder{
			{Path: ".", Role: "Root files and project manifests", FileCount: 1},
			{Path: "app", Role: "Application source", FileCount: 2},
		},
		Entries: []structure.EntryPoint{{Path: "main.go", Reason: "Go main package"}},
		Symbols: []symbols.Symbol{
			{Name: "main", Kind: "function", File: "main.go"},
			{Name: "Server", Kind: "type", File: "app/server.go"},
			{Name: "handle", Kind: "function", File: "app/handlers.go"},
			{Name: "fmt", Kind: "import", File: "main.go"},
		},
		Index: cli.IndexStats{Files: 3, Symbols: 4},
	}
}

func nodeShape(t *testing.T, d ArchDiagram, id string) ArchNode {
	t.Helper()
	n, ok := d.NodeByID(id)
	if !ok {
		t.Fatalf("node %q missing; have %+v", id, d.Nodes)
	}
	return n
}

func TestBuildArchDiagramShapesAndEdges(t *testing.T) {
	d := BuildArchDiagram(sampleResult())

	entry := nodeShape(t, d, "arch_entry_main.go")
	if entry.Shape != ShapeOval {
		t.Errorf("entry shape = %v, want ShapeOval", entry.Shape)
	}
	if !strings.Contains(entry.Does, "Go main package") {
		t.Errorf("entry.Does = %q, want it grounded in the entry reason", entry.Does)
	}

	project := nodeShape(t, d, archProjectID)
	if project.Shape != ShapeSquare {
		t.Errorf("project shape = %v, want ShapeSquare", project.Shape)
	}
	// 3 active files (one skipped), 2 folders.
	if !strings.Contains(project.Does, "3 source files") || !strings.Contains(project.Does, "2 top-level parts") {
		t.Errorf("project.Does = %q, want grounded file/folder counts", project.Does)
	}

	folder := nodeShape(t, d, "arch_folder_app")
	if folder.Shape != ShapeRectangle {
		t.Errorf("folder shape = %v, want ShapeRectangle", folder.Shape)
	}
	if !strings.Contains(folder.Does, "Application source") || !strings.Contains(folder.Does, "2 files") {
		t.Errorf("folder.Does = %q, want role and file count", folder.Does)
	}
	if !strings.Contains(folder.Does, "1 functions and 1 classes/types") {
		t.Errorf("folder.Does = %q, want per-folder symbol counts", folder.Does)
	}

	tech := nodeShape(t, d, "arch_tech_0_Go")
	if tech.Shape != ShapeCloud {
		t.Errorf("tech shape = %v, want ShapeCloud", tech.Shape)
	}
	if !strings.Contains(tech.Does, "go.mod") {
		t.Errorf("tech.Does = %q, want the real evidence file", tech.Does)
	}

	logic := nodeShape(t, d, archSymbolsID)
	if logic.Shape != ShapeCircle {
		t.Errorf("symbols shape = %v, want ShapeCircle", logic.Shape)
	}
	if !strings.Contains(logic.Does, "2 functions") || !strings.Contains(logic.Does, "1 classes/types") || !strings.Contains(logic.Does, "1 imports") {
		t.Errorf("symbols.Does = %q, want grounded totals", logic.Does)
	}

	for _, id := range []string{archIndexID, archOutputsID} {
		if n := nodeShape(t, d, id); n.Shape != ShapeCylinder {
			t.Errorf("%s shape = %v, want ShapeCylinder", id, n.Shape)
		}
	}
	if idx := nodeShape(t, d, archIndexID); !strings.Contains(idx.Represents, "codelens.db") {
		t.Errorf("index.Represents = %q, want the real db path", idx.Represents)
	}

	// Every edge must reference existing nodes and carry a known kind + why.
	kinds := map[string]bool{ConnDependency: true, ConnDataFlow: true, ConnEntryPoint: true, ConnExternal: true, ConnOutput: true}
	for _, e := range d.Edges {
		if _, ok := d.NodeByID(e.From); !ok {
			t.Errorf("edge %+v: From is not a node", e)
		}
		if _, ok := d.NodeByID(e.To); !ok {
			t.Errorf("edge %+v: To is not a node", e)
		}
		if !kinds[e.Kind] {
			t.Errorf("edge %+v: unknown kind", e)
		}
		if e.Why == "" {
			t.Errorf("edge %+v: missing Why", e)
		}
	}

	wantEdge := func(from, to, kind string) {
		t.Helper()
		for _, e := range d.Edges {
			if e.From == from && e.To == to && e.Kind == kind {
				return
			}
		}
		t.Errorf("missing edge %s -> %s (%s)", from, to, kind)
	}
	wantEdge("arch_entry_main.go", archProjectID, ConnEntryPoint)
	wantEdge(archProjectID, "arch_folder_app", ConnDependency)
	wantEdge(archProjectID, "arch_tech_0_Go", ConnExternal)
	wantEdge(archProjectID, archSymbolsID, ConnDataFlow)
	wantEdge(archSymbolsID, archIndexID, ConnDataFlow)
	wantEdge(archIndexID, archOutputsID, ConnOutput)
}

func TestBuildArchDiagramLayeredLayout(t *testing.T) {
	d := BuildArchDiagram(sampleResult())

	for _, n := range d.Nodes {
		if n.W <= 0 || n.H <= 0 {
			t.Errorf("node %s has degenerate size %vx%v", n.ID, n.W, n.H)
		}
		if n.X+n.W > d.Width || n.Y+n.H > d.Height {
			t.Errorf("node %s (%v,%v,%v,%v) outside diagram %vx%v", n.ID, n.X, n.Y, n.W, n.H, d.Width, d.Height)
		}
	}

	entry := nodeShape(t, d, "arch_entry_main.go")
	project := nodeShape(t, d, archProjectID)
	folder := nodeShape(t, d, "arch_folder_app")
	index := nodeShape(t, d, archIndexID)

	if entry.Y+entry.H > project.Y {
		t.Errorf("entry row (bottom %v) should sit above project (top %v)", entry.Y+entry.H, project.Y)
	}
	if project.Y+project.H > folder.Y {
		t.Errorf("project (bottom %v) should sit above modules (top %v)", project.Y+project.H, folder.Y)
	}
	if folder.Y+folder.H > index.Y {
		t.Errorf("modules (bottom %v) should sit above data row (top %v)", folder.Y+folder.H, index.Y)
	}

	if square := project; square.W != square.H {
		t.Errorf("square node W=%v H=%v, want equal sides", square.W, square.H)
	}
	if circle := nodeShape(t, d, archSymbolsID); circle.W != circle.H {
		t.Errorf("circle node W=%v H=%v, want equal sides", circle.W, circle.H)
	}
}

func TestBuildArchDiagramEmptyProject(t *testing.T) {
	d := BuildArchDiagram(cli.PipelineResult{Root: "/tmp/empty", OutDir: "/tmp/empty/codelens-out"})

	if _, ok := d.NodeByID(archProjectID); !ok {
		t.Fatal("project node missing for empty scan")
	}
	if _, ok := d.NodeByID(archSymbolsID); ok {
		t.Error("symbols circle should be absent when no symbols were extracted")
	}
	for _, e := range d.Edges {
		if _, ok := d.NodeByID(e.From); !ok {
			t.Errorf("dangling edge %+v", e)
		}
		if _, ok := d.NodeByID(e.To); !ok {
			t.Errorf("dangling edge %+v", e)
		}
	}
}

func TestBuildArchDiagramUsesLLMSummaryWhenPresent(t *testing.T) {
	res := sampleResult()
	res.LLM = cli.LLMResult{
		Enabled: true,
		FileSummaries: map[string]validate.FileExplanation{
			"main.go": {FileSummary: "Boots the HTTP server and wires routes."},
		},
	}
	d := BuildArchDiagram(res)
	if entry := nodeShape(t, d, "arch_entry_main.go"); entry.Does != "Boots the HTTP server and wires routes." {
		t.Errorf("entry.Does = %q, want the verified LLM summary", entry.Does)
	}
}

func TestConnectionsFor(t *testing.T) {
	d := BuildArchDiagram(sampleResult())

	conns := d.ConnectionsFor(archProjectID)
	var sawEntryIn, sawFolderOut bool
	for _, c := range conns {
		if !c.Outgoing && c.Kind == ConnEntryPoint && c.OtherLabel == "main.go" {
			sawEntryIn = true
		}
		if c.Outgoing && c.Kind == ConnDependency && c.OtherLabel == "app" {
			sawFolderOut = true
		}
		if c.Why == "" {
			t.Errorf("connection %+v missing Why", c)
		}
	}
	if !sawEntryIn {
		t.Errorf("project connections missing incoming entry point; got %+v", conns)
	}
	if !sawFolderOut {
		t.Errorf("project connections missing outgoing folder dependency; got %+v", conns)
	}

	if got := d.ConnectionsFor("nonexistent"); len(got) != 0 {
		t.Errorf("ConnectionsFor(nonexistent) = %+v, want empty", got)
	}
}

func TestNewArchitectureTabBuildsHeadlessly(t *testing.T) {
	if got := (&App{}).newArchitectureTab(sampleResult()); got == nil {
		t.Fatal("newArchitectureTab returned nil")
	}
}

func TestArchShapeWidgetTapReportsID(t *testing.T) {
	tapped := ""
	n := ArchNode{ID: "arch_folder_app", Label: "app", Shape: ShapeRectangle, W: 180, H: 44}
	w := newArchShapeWidget(n, 1.0, false, func() { tapped = n.ID })
	w.Tapped(nil)
	if tapped != "arch_folder_app" {
		t.Errorf("tap reported %q, want arch_folder_app", tapped)
	}
}

// TestBuildArchDiagramFromRealScan runs the actual pipeline over the Go
// fixture and checks the explainer stays grounded in what the scan found.
func TestBuildArchDiagramFromRealScan(t *testing.T) {
	res, err := cli.RunPipeline(cli.PipelineOptions{
		Root:   filepath.Join("..", "..", "testdata", "sample_go"),
		OutDir: t.TempDir(),
	}, nil)
	if err != nil {
		t.Fatalf("RunPipeline: %v", err)
	}

	d := BuildArchDiagram(res)

	if _, ok := d.NodeByID(archProjectID); !ok {
		t.Fatal("project node missing")
	}
	var oval, cloud bool
	for _, n := range d.Nodes {
		if n.Shape == ShapeOval {
			oval = true
		}
		if n.Shape == ShapeCloud {
			cloud = true
		}
		if n.Does == "" || n.Represents == "" {
			t.Errorf("node %s missing explanation text: %+v", n.ID, n)
		}
	}
	if !oval {
		t.Error("no entry-point oval found for sample_go (it has cmd/app/main.go)")
	}
	if !cloud {
		t.Error("no tech cloud found for sample_go (it has go.mod)")
	}
	for _, e := range d.Edges {
		if _, ok := d.NodeByID(e.From); !ok {
			t.Errorf("dangling edge %+v", e)
		}
		if _, ok := d.NodeByID(e.To); !ok {
			t.Errorf("dangling edge %+v", e)
		}
	}
}

func TestArchTopFolder(t *testing.T) {
	cases := map[string]string{
		"main.go":            ".",
		"app/server.go":      "app",
		"app/core/deep.go":   "app",
		"cmd/tool/main.go":   "cmd",
		"internal/gui/x.go":  "internal",
		"pkg/util/helper.py": "pkg",
	}
	for in, want := range cases {
		if got := archTopFolder(in); got != want {
			t.Errorf("archTopFolder(%q) = %q, want %q", in, got, want)
		}
	}
}
