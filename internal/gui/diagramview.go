package gui

import (
	"image/color"
	"path/filepath"
	"strings"

	"codelens-local/internal/scanner"
	"codelens-local/internal/stackdetect"
	"codelens-local/internal/structure"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const (
	diagramMinScale = float32(0.3)
	diagramMaxScale = float32(2.5)
	diagramZoomStep = float32(0.15)
)

var (
	colorRoot    = color.RGBA{R: 59, G: 130, B: 246, A: 255}  // Blue
	colorSource  = color.RGBA{R: 16, G: 185, B: 129, A: 255}  // Green
	colorData    = color.RGBA{R: 245, G: 158, B: 11, A: 255}  // Orange/Amber
	colorConfig  = color.RGBA{R: 139, G: 92, B: 246, A: 255}  // Purple
	colorTest    = color.RGBA{R: 239, G: 68, B: 68, A: 255}   // Red
	colorDefault = color.RGBA{R: 107, G: 114, B: 128, A: 255} // Grey
)

// newDiagramTab wraps the diagram canvas with Zoom In / Zoom Out / Reset / Fit
// controls, color role legends, and folder drill-down navigation headers.
func newDiagramTab(files []scanner.FileInfo, stack stackdetect.Report, folders []structure.Folder, entries []structure.EntryPoint) fyne.CanvasObject {
	scale := float32(1.0)
	currentScope := ""

	view := container.NewStack()
	navBar := container.NewHBox()

	var rebuild func()
	rebuild = func() {
		view.Objects = []fyne.CanvasObject{newDiagramView(files, stack, folders, entries, scale, currentScope, func(newScope string) {
			currentScope = newScope
			rebuild()
		})}
		view.Refresh()

		// Update Breadcrumbs
		breadcrumbText := "Location: Project Root"
		if currentScope != "" {
			breadcrumbText = "Location: Project Root › " + strings.ReplaceAll(currentScope, "/", " › ")
		}
		breadcrumb := widget.NewLabel(breadcrumbText)

		backBtn := widget.NewButton("Back", func() {
			if currentScope != "" {
				parent := filepath.ToSlash(filepath.Dir(currentScope))
				if parent == "." || parent == "/" {
					parent = ""
				}
				currentScope = parent
				rebuild()
			}
		})
		if currentScope == "" {
			backBtn.Disable()
		}

		navBar.Objects = []fyne.CanvasObject{backBtn, breadcrumb}
		navBar.Refresh()
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
	fit := widget.NewButton("Fit", func() {
		viewportSize := view.Size()
		layout := ComputeDiagramLayoutForScope(files, stack, folders, entries, currentScope)
		if viewportSize.Width > 0 && viewportSize.Height > 0 && layout.Width > 0 && layout.Height > 0 {
			scaleW := viewportSize.Width / (layout.Width + diagramHGap)
			scaleH := viewportSize.Height / (layout.Height + diagramVGap)
			scale = scaleW
			if scaleH < scale {
				scale = scaleH
			}
			if scale > 1.0 {
				scale = 1.0
			}
			if scale < diagramMinScale {
				scale = diagramMinScale
			}
			rebuild()
		}
	})

	controls := container.NewHBox(widget.NewLabel("Zoom:"), zoomOut, reset, zoomIn, fit)

	legend := container.NewHBox(
		widget.NewLabel("Legend:"),
		legendItem(colorRoot, "Core / Entry"),
		legendItem(colorSource, "Source / UI"),
		legendItem(colorData, "Data / Services"),
		legendItem(colorConfig, "Config / Docs"),
		legendItem(colorTest, "Tests / Fixtures"),
	)

	rebuild()

	top := container.NewVBox(controls, legend, navBar)
	return container.NewBorder(top, nil, nil, nil, view)
}

func legendItem(col color.Color, text string) fyne.CanvasObject {
	rect := canvas.NewRectangle(col)
	rect.SetMinSize(fyne.NewSize(12, 12))
	return container.NewHBox(rect, widget.NewLabel(text))
}

// newDiagramView renders diagram nodes using custom tappable widgets to support
// folder drilling, and colors them by architectural role.
func newDiagramView(files []scanner.FileInfo, stack stackdetect.Report, folders []structure.Folder, entries []structure.EntryPoint, scale float32, scope string, onScopeChanged func(string)) fyne.CanvasObject {
	layout := ComputeDiagramLayoutForScope(files, stack, folders, entries, scope)

	byID := make(map[string]DiagramNode, len(layout.Nodes))
	for _, n := range layout.Nodes {
		byID[n.ID] = n
	}

	objects := make([]fyne.CanvasObject, 0, len(layout.Nodes)+len(layout.Edges))

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
		bgColor := roleColor(n.ID)

		var onTap func()
		if strings.HasPrefix(n.ID, "folder_") {
			targetScope := strings.TrimPrefix(n.ID, "folder_")
			onTap = func() {
				onScopeChanged(targetScope)
			}
		}

		box := NewDiagramNodeWidget(n, bgColor, scale, onTap)
		box.Resize(fyne.NewSize(w, h))
		box.Move(fyne.NewPos(x, y))

		objects = append(objects, box)
	}

	canvasArea := container.NewWithoutLayout(objects...)
	canvasArea.Resize(fyne.NewSize((layout.Width+diagramHGap)*scale, (layout.Height+diagramVGap)*scale))
	return container.NewScroll(canvasArea)
}

func roleColor(nodeID string) color.Color {
	if nodeID == "root" || nodeID == "stack" {
		return colorRoot
	}
	if strings.HasPrefix(nodeID, "tech_") {
		return colorSource
	}
	if strings.HasPrefix(nodeID, "file_") {
		return colorDefault
	}
	if nodeID == "empty_placeholder" {
		return colorDefault
	}

	if strings.HasPrefix(nodeID, "folder_") {
		folderPath := strings.TrimPrefix(nodeID, "folder_")
		folderName := filepath.Base(folderPath)
		role := structure.RoleFor(folderName)

		switch role {
		case "Root files and project manifests", "Command-line entry points", "CI/CD & Workflows":
			return colorRoot
		case "Application source", "User Interface & Frontend", "API & Network Handlers":
			return colorSource
		case "Database & Storage Models", "Backend Services & Utilities":
			return colorData
		case "Configuration Settings", "Project documentation", "Notebooks & Scripting", "Static assets":
			return colorConfig
		case "Tests and fixtures":
			return colorTest
		default:
			return colorDefault
		}
	}
	return colorDefault
}

// DiagramNodeWidget is a custom Fyne widget that represents a diagram node.
// It implements fyne.Tappable to enable clicking/drilling down.
type DiagramNodeWidget struct {
	widget.BaseWidget
	node    DiagramNode
	bgColor color.Color
	scale   float32
	onTap   func()
}

func NewDiagramNodeWidget(n DiagramNode, bgColor color.Color, scale float32, onTap func()) *DiagramNodeWidget {
	w := &DiagramNodeWidget{node: n, bgColor: bgColor, scale: scale, onTap: onTap}
	w.ExtendBaseWidget(w)
	return w
}

func (w *DiagramNodeWidget) CreateRenderer() fyne.WidgetRenderer {
	box := canvas.NewRectangle(w.bgColor)
	box.StrokeColor = theme.Color(theme.ColorNamePrimary)
	box.StrokeWidth = 1.5

	label := canvas.NewText(w.node.Label, theme.Color(theme.ColorNameForeground))
	label.Alignment = fyne.TextAlignCenter
	label.TextSize = 12 * w.scale

	return &diagramNodeRenderer{
		widget: w,
		box:    box,
		label:  label,
	}
}

func (w *DiagramNodeWidget) Tapped(ev *fyne.PointEvent) {
	if w.onTap != nil {
		w.onTap()
	}
}

type diagramNodeRenderer struct {
	widget *DiagramNodeWidget
	box    *canvas.Rectangle
	label  *canvas.Text
}

func (r *diagramNodeRenderer) Destroy() {}

func (r *diagramNodeRenderer) Layout(size fyne.Size) {
	r.box.Resize(size)
	r.label.Resize(fyne.NewSize(size.Width-8, size.Height))
	r.label.Move(fyne.NewPos(4, size.Height/2-8*r.widget.scale))
}

func (r *diagramNodeRenderer) MinSize() fyne.Size {
	return fyne.NewSize(r.widget.node.W*r.widget.scale, r.widget.node.H*r.widget.scale)
}

func (r *diagramNodeRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.box, r.label}
}

func (r *diagramNodeRenderer) Refresh() {
	r.box.FillColor = r.widget.bgColor
	r.label.Text = r.widget.node.Label
	r.label.TextSize = 12 * r.widget.scale
	r.label.Refresh()
	r.box.Refresh()
}
