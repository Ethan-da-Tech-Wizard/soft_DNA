package gui

// Native Fyne rendering for the Architecture Explainer tab. All content
// comes from the ArchDiagram model in architecture.go; this file only
// draws shapes/arrows on a canvas and fills the explanation side panel
// when a shape is selected. No HTML, no webview, no browser, no localhost.

import (
	"fmt"
	"image/color"
	"math"

	"codelens-local/internal/cli"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

var archColorCloud = color.RGBA{R: 148, G: 163, B: 184, A: 255} // Slate

// archShapeColor gives each shape kind a stable fill so the legend, canvas,
// and explanation panel all agree.
func archShapeColor(k ShapeKind) color.Color {
	switch k {
	case ShapeOval:
		return colorSource // green: entry points / workflows
	case ShapeSquare:
		return colorRoot // blue: the important component
	case ShapeCircle:
		return colorConfig // purple: units of logic
	case ShapeCylinder:
		return colorData // amber: persisted data
	case ShapeCloud:
		return archColorCloud // slate: outside dependencies
	default:
		return colorDefault // grey: files and modules
	}
}

// newArchitectureTab builds the Architecture Explainer: a shaped, clickable
// diagram of the scanned project with an always-visible explanation panel.
func (a *App) newArchitectureTab(res cli.PipelineResult) fyne.CanvasObject {
	d := BuildArchDiagram(res)

	scale := float32(1.0)
	selectedID := ""

	view := container.NewStack()
	panel := container.NewStack()

	var rebuild func()
	rebuild = func() {
		view.Objects = []fyne.CanvasObject{newArchCanvas(&d, scale, selectedID, func(id string) {
			if selectedID == id {
				selectedID = "" // tapping the selected shape (or background) deselects
			} else {
				selectedID = id
			}
			rebuild()
		})}
		view.Refresh()

		panel.Objects = []fyne.CanvasObject{archPanelContent(&d, selectedID)}
		panel.Refresh()
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

	legend := container.NewHBox(
		widget.NewLabel("Legend:"),
		legendItem(archShapeColor(ShapeOval), "Oval: entry point"),
		legendItem(archShapeColor(ShapeSquare), "Square: the project"),
		legendItem(archShapeColor(ShapeRectangle), "Rectangle: module"),
		legendItem(archShapeColor(ShapeCircle), "Circle: logic"),
		legendItem(archShapeColor(ShapeCylinder), "Cylinder: stored data"),
		legendItem(archShapeColor(ShapeCloud), "Cloud: outside dependency"),
	)

	rebuild()

	panelScroll := container.NewVScroll(panel)
	panelScroll.SetMinSize(fyne.NewSize(320, 0))

	top := container.NewVBox(controls, legend)
	return container.NewBorder(top, nil, nil, panelScroll, view)
}

// newArchCanvas draws every edge (as an arrow) and node (as its shape) at
// the given zoom, marking the selected node. Tapping a shape reports its
// ID; tapping empty canvas reports the current selection so it toggles off.
func newArchCanvas(d *ArchDiagram, scale float32, selectedID string, onTap func(string)) fyne.CanvasObject {
	byID := make(map[string]ArchNode, len(d.Nodes))
	for _, n := range d.Nodes {
		byID[n.ID] = n
	}

	var objects []fyne.CanvasObject

	// Background: tapping empty space deselects.
	bg := newArchTapCatcher(func() { onTap(selectedID) })
	bg.Resize(fyne.NewSize((d.Width+diagramHGap)*scale, (d.Height+diagramVGap)*scale))
	objects = append(objects, bg)

	for _, e := range d.Edges {
		from, ok1 := byID[e.From]
		to, ok2 := byID[e.To]
		if !ok1 || !ok2 {
			continue
		}
		objects = append(objects, archArrow(from, to, scale)...)
	}

	for _, n := range d.Nodes {
		node := n
		shape := newArchShapeWidget(node, scale, node.ID == selectedID, func() { onTap(node.ID) })
		shape.Resize(fyne.NewSize(node.W*scale, node.H*scale))
		shape.Move(fyne.NewPos(node.X*scale, node.Y*scale))
		objects = append(objects, shape)
	}

	area := container.NewWithoutLayout(objects...)
	area.Resize(fyne.NewSize((d.Width+diagramHGap)*scale, (d.Height+diagramVGap)*scale))
	return container.NewScroll(area)
}

// archArrow returns the canvas objects for one edge: a line between the
// nearest sides of the two shapes plus a two-stroke arrowhead at the target.
func archArrow(from, to ArchNode, scale float32) []fyne.CanvasObject {
	var x1, y1, x2, y2 float32
	switch {
	case to.Y >= from.Y+from.H: // target below: bottom-center → top-center
		x1, y1 = from.X+from.W/2, from.Y+from.H
		x2, y2 = to.X+to.W/2, to.Y
	case from.Y >= to.Y+to.H: // target above: top-center → bottom-center
		x1, y1 = from.X+from.W/2, from.Y
		x2, y2 = to.X+to.W/2, to.Y+to.H
	case to.X >= from.X: // same row, target right: right-center → left-center
		x1, y1 = from.X+from.W, from.Y+from.H/2
		x2, y2 = to.X, to.Y+to.H/2
	default: // same row, target left
		x1, y1 = from.X, from.Y+from.H/2
		x2, y2 = to.X+to.W, to.Y+to.H/2
	}
	x1, y1, x2, y2 = x1*scale, y1*scale, x2*scale, y2*scale

	col := theme.Color(theme.ColorNameSeparator)
	line := canvas.NewLine(col)
	line.Position1 = fyne.NewPos(x1, y1)
	line.Position2 = fyne.NewPos(x2, y2)
	line.StrokeWidth = 1.5

	// Arrowhead: two short strokes angled back from the tip.
	angle := math.Atan2(float64(y2-y1), float64(x2-x1))
	headLen := float64(9 * scale)
	head := func(offset float64) *canvas.Line {
		l := canvas.NewLine(col)
		l.StrokeWidth = 1.5
		l.Position1 = fyne.NewPos(x2, y2)
		l.Position2 = fyne.NewPos(
			x2-float32(headLen*math.Cos(angle+offset)),
			y2-float32(headLen*math.Sin(angle+offset)),
		)
		return l
	}
	return []fyne.CanvasObject{line, head(math.Pi / 7), head(-math.Pi / 7)}
}

// archPanelContent renders the explanation panel for the selected shape,
// or a short hint when nothing is selected.
func archPanelContent(d *ArchDiagram, selectedID string) fyne.CanvasObject {
	hint := func(text string) fyne.CanvasObject {
		l := widget.NewLabel(text)
		l.Wrapping = fyne.TextWrapWord
		return container.NewVBox(widget.NewLabelWithStyle("Explanation", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}), l)
	}
	if selectedID == "" {
		return hint("Click any shape to see what it represents, what it does, and how it connects to the rest of the project.")
	}
	n, ok := d.NodeByID(selectedID)
	if !ok {
		return hint("Click any shape to see what it represents, what it does, and how it connects to the rest of the project.")
	}

	wrapped := func(text string) *widget.Label {
		l := widget.NewLabel(text)
		l.Wrapping = fyne.TextWrapWord
		return l
	}
	bold := func(text string) *widget.Label {
		return widget.NewLabelWithStyle(text, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	}

	items := []fyne.CanvasObject{
		bold(n.Label),
		wrapped(n.Shape.Meaning()),
		widget.NewSeparator(),
		bold("Represents"),
		wrapped(n.Represents),
		bold("What it does"),
		wrapped(n.Does),
	}

	if conns := d.ConnectionsFor(n.ID); len(conns) > 0 {
		items = append(items, widget.NewSeparator(), bold("Connections"))
		for _, c := range conns {
			dir := "→"
			if !c.Outgoing {
				dir = "←"
			}
			items = append(items,
				wrapped(fmt.Sprintf("%s %s (%s)", dir, c.OtherLabel, c.Kind)),
				wrapped("    "+c.Why),
			)
		}
	}
	return container.NewVBox(items...)
}

// archShapeWidget is one tappable shape on the Architecture Explainer
// canvas. The selected shape is drawn with a thick primary-color outline.
type archShapeWidget struct {
	widget.BaseWidget
	node     ArchNode
	scale    float32
	selected bool
	onTap    func()
}

func newArchShapeWidget(n ArchNode, scale float32, selected bool, onTap func()) *archShapeWidget {
	w := &archShapeWidget{node: n, scale: scale, selected: selected, onTap: onTap}
	w.ExtendBaseWidget(w)
	return w
}

func (w *archShapeWidget) Tapped(*fyne.PointEvent) {
	if w.onTap != nil {
		w.onTap()
	}
}

func (w *archShapeWidget) CreateRenderer() fyne.WidgetRenderer {
	fill := archShapeColor(w.node.Shape)
	strokeColor := theme.Color(theme.ColorNameSeparator)
	strokeWidth := float32(1.5)
	if w.selected {
		strokeColor = theme.Color(theme.ColorNamePrimary)
		strokeWidth = 3
	}

	var parts []fyne.CanvasObject
	switch w.node.Shape {
	case ShapeCircle, ShapeOval:
		c := canvas.NewCircle(fill) // canvas.Circle draws an ellipse fitting its bounds
		c.StrokeColor = strokeColor
		c.StrokeWidth = strokeWidth
		parts = append(parts, c)
	case ShapeCylinder:
		body := canvas.NewRectangle(fill)
		body.StrokeColor = strokeColor
		body.StrokeWidth = strokeWidth
		capEllipse := canvas.NewCircle(fill)
		capEllipse.StrokeColor = strokeColor
		capEllipse.StrokeWidth = strokeWidth
		parts = append(parts, body, capEllipse)
	case ShapeCloud:
		for i := 0; i < 3; i++ {
			puff := canvas.NewCircle(fill)
			if w.selected {
				puff.StrokeColor = strokeColor
				puff.StrokeWidth = strokeWidth
			}
			parts = append(parts, puff)
		}
	default: // ShapeRectangle, ShapeSquare
		r := canvas.NewRectangle(fill)
		r.StrokeColor = strokeColor
		r.StrokeWidth = strokeWidth
		parts = append(parts, r)
	}

	label := canvas.NewText(w.node.Label, theme.Color(theme.ColorNameForeground))
	label.Alignment = fyne.TextAlignCenter
	label.TextSize = 11 * w.scale

	return &archShapeRenderer{widget: w, parts: parts, label: label}
}

type archShapeRenderer struct {
	widget *archShapeWidget
	parts  []fyne.CanvasObject
	label  *canvas.Text
}

func (r *archShapeRenderer) Destroy() {}

func (r *archShapeRenderer) Layout(size fyne.Size) {
	switch r.widget.node.Shape {
	case ShapeCylinder:
		// Body below a shallow ellipse cap, so it reads as a cylinder.
		capH := size.Height * 0.35
		r.parts[0].Resize(fyne.NewSize(size.Width, size.Height-capH/2))
		r.parts[0].Move(fyne.NewPos(0, capH/2))
		r.parts[1].Resize(fyne.NewSize(size.Width, capH))
		r.parts[1].Move(fyne.NewPos(0, 0))
	case ShapeCloud:
		// Three overlapping puffs.
		r.parts[0].Resize(fyne.NewSize(size.Width*0.55, size.Height*0.7))
		r.parts[0].Move(fyne.NewPos(0, size.Height*0.3))
		r.parts[1].Resize(fyne.NewSize(size.Width*0.55, size.Height*0.7))
		r.parts[1].Move(fyne.NewPos(size.Width*0.45, size.Height*0.3))
		r.parts[2].Resize(fyne.NewSize(size.Width*0.5, size.Height*0.7))
		r.parts[2].Move(fyne.NewPos(size.Width*0.25, 0))
	default:
		r.parts[0].Resize(size)
		r.parts[0].Move(fyne.NewPos(0, 0))
	}

	r.label.Resize(fyne.NewSize(size.Width-8, size.Height))
	labelY := size.Height/2 - 8*r.widget.scale
	if r.widget.node.Shape == ShapeCloud || r.widget.node.Shape == ShapeCylinder {
		labelY = size.Height*0.55 - 8*r.widget.scale
	}
	r.label.Move(fyne.NewPos(4, labelY))
}

func (r *archShapeRenderer) MinSize() fyne.Size {
	return fyne.NewSize(r.widget.node.W*r.widget.scale, r.widget.node.H*r.widget.scale)
}

func (r *archShapeRenderer) Objects() []fyne.CanvasObject {
	return append(append([]fyne.CanvasObject{}, r.parts...), r.label)
}

func (r *archShapeRenderer) Refresh() {
	for _, p := range r.parts {
		p.Refresh()
	}
	r.label.Refresh()
}

// archTapCatcher is an invisible full-canvas widget behind the shapes;
// tapping it (i.e. empty space) clears the selection.
type archTapCatcher struct {
	widget.BaseWidget
	onTap func()
}

func newArchTapCatcher(onTap func()) *archTapCatcher {
	w := &archTapCatcher{onTap: onTap}
	w.ExtendBaseWidget(w)
	return w
}

func (w *archTapCatcher) Tapped(*fyne.PointEvent) {
	if w.onTap != nil {
		w.onTap()
	}
}

func (w *archTapCatcher) CreateRenderer() fyne.WidgetRenderer {
	bg := canvas.NewRectangle(color.Transparent)
	return widget.NewSimpleRenderer(bg)
}
