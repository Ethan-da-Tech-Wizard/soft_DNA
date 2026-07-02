package gui

import (
	"codelens-local/internal/stackdetect"
	"codelens-local/internal/structure"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const (
	diagramMinScale = float32(0.5)
	diagramMaxScale = float32(2.5)
	diagramZoomStep = float32(0.25)
)

// newDiagramTab wraps the diagram canvas with Zoom In / Zoom Out / Reset
// controls. Zooming rebuilds the canvas at a new scale rather than trying to
// transform existing objects — simplest correct option given Fyne has no
// built-in scale transform for arbitrary containers.
func newDiagramTab(stack stackdetect.Report, folders []structure.Folder, entries []structure.EntryPoint) fyne.CanvasObject {
	scale := float32(1.0)
	view := container.NewStack(newDiagramView(stack, folders, entries, scale))

	rebuild := func() {
		view.Objects = []fyne.CanvasObject{newDiagramView(stack, folders, entries, scale)}
		view.Refresh()
	}

	zoomOut := widget.NewButton("−", func() {
		if scale > diagramMinScale {
			scale -= diagramZoomStep
			rebuild()
		}
	})
	zoomIn := widget.NewButton("+", func() {
		if scale < diagramMaxScale {
			scale += diagramZoomStep
			rebuild()
		}
	})
	reset := widget.NewButton("Reset", func() {
		scale = 1.0
		rebuild()
	})
	controls := container.NewHBox(widget.NewLabel("Zoom:"), zoomOut, reset, zoomIn)

	return container.NewBorder(controls, nil, nil, nil, view)
}

// newDiagramView renders the same root/folders/entry-points/stack tree as
// diagram.WriteMermaid produces textually, but as native boxes and lines on
// a Fyne canvas — no Mermaid parser, no browser, no image export step.
func newDiagramView(stack stackdetect.Report, folders []structure.Folder, entries []structure.EntryPoint, scale float32) fyne.CanvasObject {
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
		line.Position1 = fyne.NewPos((from.X+from.W/2)*scale, (from.Y+from.H)*scale)
		line.Position2 = fyne.NewPos((to.X+to.W/2)*scale, to.Y*scale)
		line.StrokeWidth = 1.5
		objects = append(objects, line)
	}

	for _, n := range layout.Nodes {
		x, y, w, h := n.X*scale, n.Y*scale, n.W*scale, n.H*scale

		box := canvas.NewRectangle(theme.Color(theme.ColorNameInputBackground))
		box.StrokeColor = theme.Color(theme.ColorNamePrimary)
		box.StrokeWidth = 1.5
		box.Resize(fyne.NewSize(w, h))
		box.Move(fyne.NewPos(x, y))

		label := canvas.NewText(n.Label, theme.Color(theme.ColorNameForeground))
		label.Alignment = fyne.TextAlignCenter
		label.TextSize = 12 * scale
		label.Resize(fyne.NewSize(w-8, h))
		label.Move(fyne.NewPos(x+4, y+h/2-8*scale))

		objects = append(objects, box, label)
	}

	canvasArea := container.NewWithoutLayout(objects...)
	canvasArea.Resize(fyne.NewSize((layout.Width+diagramHGap)*scale, (layout.Height+diagramVGap)*scale))
	return container.NewScroll(canvasArea)
}
