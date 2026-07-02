package gui

import (
	"codelens-local/internal/stackdetect"
	"codelens-local/internal/structure"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
)

// newDiagramView renders the same root/folders/entry-points/stack tree as
// diagram.WriteMermaid produces textually, but as native boxes and lines on
// a Fyne canvas — no Mermaid parser, no browser, no image export step.
func newDiagramView(stack stackdetect.Report, folders []structure.Folder, entries []structure.EntryPoint) fyne.CanvasObject {
	layout := ComputeDiagramLayout(stack, folders, entries)

	byID := make(map[string]DiagramNode, len(layout.Nodes))
	for _, n := range layout.Nodes {
		byID[n.ID] = n
	}

	objects := make([]fyne.CanvasObject, 0, len(layout.Nodes)*2+len(layout.Edges))

	for _, e := range layout.Edges {
		from, ok1 := byID[e.From]
		to, ok2 := byID[e.To]
		if !ok1 || !ok2 {
			continue
		}
		line := canvas.NewLine(theme.Color(theme.ColorNameSeparator))
		line.Position1 = fyne.NewPos(from.X+from.W/2, from.Y+from.H)
		line.Position2 = fyne.NewPos(to.X+to.W/2, to.Y)
		line.StrokeWidth = 1.5
		objects = append(objects, line)
	}

	for _, n := range layout.Nodes {
		box := canvas.NewRectangle(theme.Color(theme.ColorNameInputBackground))
		box.StrokeColor = theme.Color(theme.ColorNamePrimary)
		box.StrokeWidth = 1.5
		box.Resize(fyne.NewSize(n.W, n.H))
		box.Move(fyne.NewPos(n.X, n.Y))

		label := canvas.NewText(n.Label, theme.Color(theme.ColorNameForeground))
		label.Alignment = fyne.TextAlignCenter
		label.TextSize = 12
		label.Resize(fyne.NewSize(n.W-8, n.H))
		label.Move(fyne.NewPos(n.X+4, n.Y+n.H/2-8))

		objects = append(objects, box, label)
	}

	canvasArea := container.NewWithoutLayout(objects...)
	canvasArea.Resize(fyne.NewSize(layout.Width+diagramHGap, layout.Height+diagramVGap))
	return container.NewScroll(canvasArea)
}
