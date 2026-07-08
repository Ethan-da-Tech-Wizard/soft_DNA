package gui

// Architecture Explainer model: a shaped, explained view of a scanned
// project built entirely from cli.PipelineResult — no new analysis pass.
// This file is pure logic (no Fyne imports) so it can be tested headlessly;
// the native rendering lives in architectureview.go, mirroring the
// diagramlayout.go / diagramview.go split.

import (
	"fmt"
	"path/filepath"
	"strings"

	"codelens-local/internal/cli"
)

// ShapeKind classifies what a diagram shape means, per the legend in
// ARCHITECTURE_TODO.md.
type ShapeKind int

const (
	ShapeRectangle ShapeKind = iota // file or module
	ShapeSquare                     // important component
	ShapeCircle                     // function, service, or focused unit of logic
	ShapeOval                       // user action, entry point, or workflow
	ShapeCylinder                   // database, index, local storage, persisted data
	ShapeCloud                      // external service, API, or outside dependency
)

// Meaning returns the legend text for the shape, shown in the explanation
// panel and the on-screen legend.
func (k ShapeKind) Meaning() string {
	switch k {
	case ShapeRectangle:
		return "Rectangle — file or module"
	case ShapeSquare:
		return "Square — important component"
	case ShapeCircle:
		return "Circle — focused unit of logic"
	case ShapeOval:
		return "Oval — entry point or user workflow"
	case ShapeCylinder:
		return "Cylinder — persisted data (database, index, files)"
	case ShapeCloud:
		return "Cloud — outside dependency or external technology"
	}
	return "Unknown shape"
}

// Connection kinds, per ARCHITECTURE_TODO.md §4.
const (
	ConnDependency = "dependency"
	ConnDataFlow   = "data flow"
	ConnEntryPoint = "entry point"
	ConnExternal   = "external service"
	ConnOutput     = "output"
)

// ArchNode is one selectable shape: identity and geometry for the canvas,
// plus the plain-English explanation shown when the shape is selected.
type ArchNode struct {
	ID         string
	Label      string
	Shape      ShapeKind
	Represents string // the real file, folder, or system part
	Does       string // what that part does, in plain English
	X, Y, W, H float32
}

// ArchEdge is one arrow. Kind is a Conn* constant and Why says, in plain
// English, why the connection matters.
type ArchEdge struct {
	From, To string
	Kind     string
	Why      string
}

// ArchDiagram is the complete renderer-agnostic Architecture Explainer
// content for one scan.
type ArchDiagram struct {
	Nodes         []ArchNode
	Edges         []ArchEdge
	Width, Height float32
}

// ArchConnection is one entry in a selected shape's "connects to" list,
// derived from the edges touching that shape.
type ArchConnection struct {
	OtherLabel string
	Outgoing   bool // true: arrow points from the selected shape to Other
	Kind       string
	Why        string
}

// ConnectionsFor lists the connections touching node id, for the
// explanation panel.
func (d *ArchDiagram) ConnectionsFor(id string) []ArchConnection {
	labels := make(map[string]string, len(d.Nodes))
	for _, n := range d.Nodes {
		labels[n.ID] = n.Label
	}
	var out []ArchConnection
	for _, e := range d.Edges {
		switch id {
		case e.From:
			out = append(out, ArchConnection{OtherLabel: labels[e.To], Outgoing: true, Kind: e.Kind, Why: e.Why})
		case e.To:
			out = append(out, ArchConnection{OtherLabel: labels[e.From], Outgoing: false, Kind: e.Kind, Why: e.Why})
		}
	}
	return out
}

// NodeByID returns the node with the given ID, if present.
func (d *ArchDiagram) NodeByID(id string) (ArchNode, bool) {
	for _, n := range d.Nodes {
		if n.ID == id {
			return n, true
		}
	}
	return ArchNode{}, false
}

// archTopFolder returns the top-level folder a relative file path lives in,
// or "." for root files — matching how structure.Analyze buckets folders.
func archTopFolder(path string) string {
	dir := filepath.ToSlash(filepath.Dir(path))
	if dir == "." {
		return "."
	}
	return strings.SplitN(dir, "/", 2)[0]
}

const (
	archProjectID = "arch_project"
	archSymbolsID = "arch_symbols"
	archIndexID   = "arch_index"
	archOutputsID = "arch_outputs"
)

// BuildArchDiagram turns one scan result into the Architecture Explainer
// diagram. Every explanation is grounded in data the pipeline already
// produced; nothing is invented. When the optional LLM pass ran, its
// verified file summaries enrich the matching entry-point shapes.
func BuildArchDiagram(res cli.PipelineResult) ArchDiagram {
	var d ArchDiagram

	// Symbol counts per top-level folder ground the module explanations.
	type counts struct{ funcs, types int }
	perFolder := make(map[string]counts)
	var totalFuncs, totalTypes, totalImports int
	for _, s := range res.Symbols {
		top := archTopFolder(s.File)
		c := perFolder[top]
		switch s.Kind {
		case "import":
			totalImports++
		case "class", "type", "table", "view":
			c.types++
			totalTypes++
		default: // function, method, procedure
			c.funcs++
			totalFuncs++
		}
		perFolder[top] = c
	}

	// Ovals: entry points — where a user or the runtime starts.
	var entryIDs []string
	for _, e := range res.Entries {
		id := "arch_entry_" + e.Path
		does := "A likely starting point of this project (" + e.Reason + "). Reading it first shows how the program boots and hands off to the rest of the code."
		if res.LLM.Enabled {
			if fe, ok := res.LLM.FileSummaries[e.Path]; ok && fe.FileSummary != "" {
				does = fe.FileSummary
			}
		}
		d.Nodes = append(d.Nodes, ArchNode{
			ID: id, Label: e.Path, Shape: ShapeOval,
			Represents: e.Path,
			Does:       does,
		})
		entryIDs = append(entryIDs, id)
		d.Edges = append(d.Edges, ArchEdge{
			From: id, To: archProjectID, Kind: ConnEntryPoint,
			Why: "Execution starts here and flows into the rest of the project, so this is where a reader should start too.",
		})
	}

	// Square: the scanned project itself — the one important component
	// everything else hangs off.
	activeFiles := 0
	for _, f := range res.Files {
		if !f.Skipped {
			activeFiles++
		}
	}
	projectLabel := filepath.Base(strings.TrimRight(filepath.ToSlash(res.Root), "/"))
	if projectLabel == "" || projectLabel == "." || projectLabel == ".." {
		projectLabel = "Scanned project"
	}
	d.Nodes = append(d.Nodes, ArchNode{
		ID: archProjectID, Label: projectLabel, Shape: ShapeSquare,
		Represents: res.Root,
		Does: fmt.Sprintf("The scanned codebase as a whole: %d source files across %d top-level parts. %s",
			activeFiles, len(res.Folders), strings.TrimSpace(res.Stack.Summary)),
	})

	// Rectangles: top-level folders (modules).
	for _, f := range res.Folders {
		id := "arch_folder_" + f.Path
		label := f.Path
		represents := f.Path + "/"
		if f.Path == "." {
			label = "(root files)"
			represents = "files at the project root"
		}
		does := fmt.Sprintf("%s. Holds %d files", f.Role, f.FileCount)
		if c := perFolder[f.Path]; c.funcs > 0 || c.types > 0 {
			does += fmt.Sprintf("; %d functions and %d classes/types were found in it", c.funcs, c.types)
		}
		does += "."
		d.Nodes = append(d.Nodes, ArchNode{
			ID: id, Label: label, Shape: ShapeRectangle,
			Represents: represents,
			Does:       does,
		})
		d.Edges = append(d.Edges, ArchEdge{
			From: archProjectID, To: id, Kind: ConnDependency,
			Why: "The project is organised into this part; understanding it explains one slice of the codebase.",
		})
	}

	// Clouds: detected technologies — outside dependencies, not this
	// project's own source.
	for i, tech := range res.Stack.Tech {
		id := fmt.Sprintf("arch_tech_%d_%s", i, tech.Name)
		d.Nodes = append(d.Nodes, ArchNode{
			ID: id, Label: tech.Name, Shape: ShapeCloud,
			Represents: tech.Category + " (detected from " + tech.EvidenceFile + ")",
			Does:       "An outside technology this project is built on. It was detected from " + tech.EvidenceFile + ", so the project depends on it but does not contain its source.",
		})
		d.Edges = append(d.Edges, ArchEdge{
			From: archProjectID, To: id, Kind: ConnExternal,
			Why: "Knowing the stack tells you which language or tooling skills the project assumes.",
		})
	}

	// Circle: the extracted symbols — the project's units of logic.
	if len(res.Symbols) > 0 {
		d.Nodes = append(d.Nodes, ArchNode{
			ID: archSymbolsID, Label: "Functions & classes", Shape: ShapeCircle,
			Represents: fmt.Sprintf("%d symbols extracted from the source", len(res.Symbols)),
			Does: fmt.Sprintf("The project's units of logic: %d functions, %d classes/types, and %d imports, found by reading the source (never running it).",
				totalFuncs, totalTypes, totalImports),
		})
		d.Edges = append(d.Edges, ArchEdge{
			From: archProjectID, To: archSymbolsID, Kind: ConnDataFlow,
			Why: "Scanning the project's source produces this inventory of functions and classes.",
		})
		d.Edges = append(d.Edges, ArchEdge{
			From: archSymbolsID, To: archIndexID, Kind: ConnDataFlow,
			Why: "Every symbol is stored in the local search index so questions about the code can be answered with citations.",
		})
	}

	// Cylinders: persisted data — the search index and the written reports.
	d.Nodes = append(d.Nodes, ArchNode{
		ID: archIndexID, Label: "Search index", Shape: ShapeCylinder,
		Represents: filepath.Join(res.OutDir, "codelens.db"),
		Does: fmt.Sprintf("A local SQLite full-text index of %d files and %d symbols. It stays on this machine and powers the Ask tab's cited answers.",
			res.Index.Files, res.Index.Symbols),
	})
	d.Nodes = append(d.Nodes, ArchNode{
		ID: archOutputsID, Label: "Reports", Shape: ShapeCylinder,
		Represents: res.OutDir + "/",
		Does:       "Everything a scan writes for the user: CODEBASE_REPORT.md, architecture.mmd, ANSWER.md, and cached scan data — plain files, kept locally.",
	})
	d.Edges = append(d.Edges, ArchEdge{
		From: archIndexID, To: archOutputsID, Kind: ConnOutput,
		Why: "Answers and reports generated from the index are written here as files the user keeps.",
	})

	layoutArch(&d, entryIDs)
	return d
}

// Shape sizes in layout units. Rectangles/ovals reuse the Diagram tab's
// box size; squares and circles must keep equal sides to read as their
// shape.
const (
	archSquareSide = float32(96)
	archCircleSide = float32(110)
	archCylinderH  = float32(56)
)

// layoutArch assigns positions in simple layered rows, top to bottom:
// entry ovals → project square → modules/tech/logic (wrapped grid) →
// data cylinders. Every node in a row fans out from (or into) the row
// above, keeping arrows readable without any graph routing.
func layoutArch(d *ArchDiagram, entryIDs []string) {
	byID := make(map[string]*ArchNode, len(d.Nodes))
	for i := range d.Nodes {
		byID[d.Nodes[i].ID] = &d.Nodes[i]
	}

	entrySet := make(map[string]bool, len(entryIDs))
	for _, id := range entryIDs {
		entrySet[id] = true
	}

	var rowEntries, rowMiddle, rowData []*ArchNode
	for i := range d.Nodes {
		n := &d.Nodes[i]
		switch {
		case entrySet[n.ID]:
			rowEntries = append(rowEntries, n)
		case n.ID == archProjectID:
			// placed separately, centered
		case n.Shape == ShapeCylinder:
			rowData = append(rowData, n)
		default:
			rowMiddle = append(rowMiddle, n)
		}
	}

	// Size every node by its shape.
	size := func(n *ArchNode) {
		switch n.Shape {
		case ShapeSquare:
			n.W, n.H = archSquareSide, archSquareSide
		case ShapeCircle:
			n.W, n.H = archCircleSide, archCircleSide
		case ShapeCylinder:
			n.W, n.H = diagramNodeW, archCylinderH
		default:
			n.W, n.H = diagramNodeW, diagramNodeH
		}
	}
	for i := range d.Nodes {
		size(&d.Nodes[i])
	}

	const maxCols = 5
	placeRow := func(nodes []*ArchNode, y float32) float32 {
		rowH := float32(0)
		for i, n := range nodes {
			col := i % maxCols
			row := i / maxCols
			n.X = float32(col) * (diagramNodeW + diagramHGap)
			n.Y = y + float32(row)*(archCircleSide+diagramVGap)
			if bottom := n.Y + n.H - y; bottom > rowH {
				rowH = bottom
			}
		}
		if len(nodes) == 0 {
			return 0
		}
		return rowH
	}

	gridCols := len(rowMiddle)
	if gridCols > maxCols {
		gridCols = maxCols
	}
	if gridCols == 0 {
		gridCols = 1
	}
	gridWidth := float32(gridCols)*diagramNodeW + float32(gridCols-1)*diagramHGap

	y := float32(0)
	if h := placeRow(rowEntries, y); h > 0 {
		y += h + diagramVGap
	}
	if project, ok := byID[archProjectID]; ok {
		project.X = (gridWidth - project.W) / 2
		project.Y = y
		y += project.H + diagramVGap
	}
	if h := placeRow(rowMiddle, y); h > 0 {
		y += h + diagramVGap
	}
	placeRow(rowData, y)

	for _, n := range d.Nodes {
		if right := n.X + n.W; right > d.Width {
			d.Width = right
		}
		if bottom := n.Y + n.H; bottom > d.Height {
			d.Height = bottom
		}
	}
}
