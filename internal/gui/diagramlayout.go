package gui

import (
	"strconv"

	"codelens-local/internal/stackdetect"
	"codelens-local/internal/structure"
)

// DiagramNode is one box in the architecture diagram: an ID for edge
// lookups, a display label, and an absolute position/size in layout units.
type DiagramNode struct {
	ID, Label  string
	X, Y, W, H float32
}

// DiagramEdge connects two node IDs with a line.
type DiagramEdge struct {
	From, To string
}

// DiagramLayout is the complete, renderer-agnostic result of laying out the
// same root/folders/entry-points/stack tree that diagram.WriteMermaid
// renders as Mermaid text — computed here as absolute box positions so a
// native canvas can draw it directly, with no Mermaid parser involved.
type DiagramLayout struct {
	Nodes         []DiagramNode
	Edges         []DiagramEdge
	Width, Height float32
}

const (
	diagramNodeW = float32(180)
	diagramNodeH = float32(44)
	diagramHGap  = float32(24)
	diagramVGap  = float32(70)
)

// ComputeDiagramLayout lays out one root node fanning out to a grid of folder
// and entry-point boxes plus a "Detected Stack" box, which itself fans out
// to a row of technology boxes — mirroring diagram.WriteMermaid's structure.
//
// Layer 1 is wrapped into rows of at most 5 columns so it remains readable
// on medium displays, and the root is centered horizontally above them.
func ComputeDiagramLayout(stack stackdetect.Report, folders []structure.Folder, entries []structure.EntryPoint) DiagramLayout {
	const rootID = "root"
	const stackID = "stack"

	layout := DiagramLayout{
		Nodes: []DiagramNode{{ID: rootID, Label: "Project Root", W: diagramNodeW, H: diagramNodeH}},
	}

	var layer1 []DiagramNode
	for _, f := range folders {
		id := "folder_" + f.Path
		layer1 = append(layer1, DiagramNode{ID: id, Label: f.Path + ": " + f.Role})
		layout.Edges = append(layout.Edges, DiagramEdge{From: rootID, To: id})
	}
	for _, e := range entries {
		id := "entry_" + e.Path
		layer1 = append(layer1, DiagramNode{ID: id, Label: e.Path})
		layout.Edges = append(layout.Edges, DiagramEdge{From: rootID, To: id})
	}
	hasStack := len(stack.Tech) > 0
	if hasStack {
		layer1 = append(layer1, DiagramNode{ID: stackID, Label: "Detected Stack"})
		layout.Edges = append(layout.Edges, DiagramEdge{From: rootID, To: stackID})
	}

	const maxCols = 5
	rowY := diagramVGap + diagramNodeH
	var lastRowIndex int
	for i := range layer1 {
		col := i % maxCols
		row := i / maxCols
		layer1[i].X = float32(col) * (diagramNodeW + diagramHGap)
		layer1[i].Y = rowY + float32(row)*(diagramNodeH+diagramVGap)
		layer1[i].W = diagramNodeW
		layer1[i].H = diagramNodeH
		lastRowIndex = row
	}
	layout.Nodes = append(layout.Nodes, layer1...)

	var layer2Y float32
	if len(layer1) > 0 {
		layer2Y = rowY + float32(lastRowIndex)*(diagramNodeH+diagramVGap) + diagramNodeH + diagramVGap
	} else {
		layer2Y = rowY + diagramNodeH + diagramVGap
	}

	var layer2 []DiagramNode
	for i, tech := range stack.Tech {
		id := "tech_" + strconv.Itoa(i) + "_" + tech.Name
		layer2 = append(layer2, DiagramNode{ID: id, Label: tech.Name})
		layout.Edges = append(layout.Edges, DiagramEdge{From: stackID, To: id})
	}
	for i := range layer2 {
		layer2[i].X = float32(i) * (diagramNodeW + diagramHGap)
		layer2[i].Y = layer2Y
		layer2[i].W = diagramNodeW
		layer2[i].H = diagramNodeH
	}
	layout.Nodes = append(layout.Nodes, layer2...)

	maxColsInLayer1 := maxCols
	if len(layer1) < maxCols {
		maxColsInLayer1 = len(layer1)
	}
	if maxColsInLayer1 == 0 {
		maxColsInLayer1 = 1
	}
	gridWidth := float32(maxColsInLayer1)*diagramNodeW + float32(maxColsInLayer1-1)*diagramHGap
	layout.Nodes[0].X = (gridWidth - diagramNodeW) / 2
	layout.Nodes[0].Y = 0

	for _, n := range layout.Nodes {
		if right := n.X + n.W; right > layout.Width {
			layout.Width = right
		}
		if bottom := n.Y + n.H; bottom > layout.Height {
			layout.Height = bottom
		}
	}
	return layout
}
