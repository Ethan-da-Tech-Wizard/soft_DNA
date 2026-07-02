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

// ComputeDiagramLayout lays out one root node fanning out to a row of folder
// and entry-point boxes plus a "Detected Stack" box, which itself fans out
// to a row of technology boxes — mirroring diagram.WriteMermaid's structure.
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
	placeRow(layer1, diagramVGap+diagramNodeH)
	layout.Nodes = append(layout.Nodes, layer1...)

	var layer2 []DiagramNode
	for i, tech := range stack.Tech {
		id := "tech_" + strconv.Itoa(i) + "_" + tech.Name
		layer2 = append(layer2, DiagramNode{ID: id, Label: tech.Name})
		layout.Edges = append(layout.Edges, DiagramEdge{From: stackID, To: id})
	}
	placeRow(layer2, 2*(diagramVGap+diagramNodeH))
	layout.Nodes = append(layout.Nodes, layer2...)

	if len(layer1) > 0 {
		centerX := (layer1[0].X + layer1[len(layer1)-1].X + diagramNodeW) / 2
		layout.Nodes[0].X = centerX - diagramNodeW/2
	}

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

func placeRow(nodes []DiagramNode, y float32) {
	for i := range nodes {
		nodes[i].X = float32(i) * (diagramNodeW + diagramHGap)
		nodes[i].Y = y
		nodes[i].W = diagramNodeW
		nodes[i].H = diagramNodeH
	}
}
