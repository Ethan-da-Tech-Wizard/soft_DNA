package gui

import (
	"fmt"
	"os"
	"strings"
)

// ExportDiagramSVG serialises layout into an SVG file at svgPath. It uses the
// same colours and box dimensions as the native Fyne canvas so the export
// matches what the user sees on screen. No external dependencies — just stdlib
// fmt / strings / os.
func ExportDiagramSVG(layout DiagramLayout, svgPath string) error {
	const pad = float32(20) // margin around the whole drawing
	w := layout.Width + 2*pad
	h := layout.Height + 2*pad
	if w <= 0 {
		w = 400
	}
	if h <= 0 {
		h = 200
	}

	var sb strings.Builder

	fmt.Fprintf(&sb, `<?xml version="1.0" encoding="UTF-8"?>`+"\n")
	fmt.Fprintf(&sb, `<svg xmlns="http://www.w3.org/2000/svg" width="%.0f" height="%.0f" viewBox="0 0 %.0f %.0f">`+"\n",
		w, h, w, h)

	// White background
	fmt.Fprintf(&sb, `  <rect width="%.0f" height="%.0f" fill="#1e1e2e"/>%s`, w, h, "\n")

	// Build a quick node-ID → center lookup for edge routing
	type center struct{ x, y float32 }
	nodeCenters := make(map[string]center, len(layout.Nodes))
	for _, n := range layout.Nodes {
		nx := pad + n.X + n.W/2
		ny := pad + n.Y + n.H/2
		nodeCenters[n.ID] = center{nx, ny}
	}

	// Draw edges first (underneath boxes)
	for _, e := range layout.Edges {
		f, ok1 := nodeCenters[e.From]
		t, ok2 := nodeCenters[e.To]
		if !ok1 || !ok2 {
			continue
		}
		fmt.Fprintf(&sb,
			`  <line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="#4b5563" stroke-width="1.5" marker-end="url(#arrowhead)"/>%s`,
			f.x, f.y, t.x, t.y, "\n",
		)
	}

	// Draw nodes
	for _, n := range layout.Nodes {
		x := pad + n.X
		y := pad + n.Y
		wn := n.W
		hn := n.H
		if wn == 0 {
			wn = diagramNodeW
		}
		if hn == 0 {
			hn = diagramNodeH
		}
		fillHex := nodeColorHex(n)
		fmt.Fprintf(&sb,
			`  <rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" rx="6" ry="6" fill="%s" stroke="#6b7280" stroke-width="1"/>%s`,
			x, y, wn, hn, fillHex, "\n",
		)
		// Label (truncate to ~24 chars so it fits)
		label := n.Label
		if len(label) > 26 {
			label = label[:24] + "…"
		}
		cy := y + hn/2 + 5
		fmt.Fprintf(&sb,
			`  <text x="%.1f" y="%.1f" font-family="monospace" font-size="11" fill="#f9fafb" text-anchor="middle">%s</text>%s`,
			x+wn/2, cy, svgEscape(label), "\n",
		)
	}

	// Arrowhead marker def
	arrow := `  <defs>
    <marker id="arrowhead" markerWidth="8" markerHeight="6" refX="8" refY="3" orient="auto">
      <polygon points="0 0, 8 3, 0 6" fill="#4b5563"/>
    </marker>
  </defs>` + "\n"
	// Prepend defs after opening svg tag — simplest: insert before first <rect
	content := sb.String()
	content = strings.Replace(content, "<rect", arrow+"  <rect", 1)
	content += "</svg>\n"

	return os.WriteFile(svgPath, []byte(content), 0o644)
}

// nodeColorHex returns the fill colour for a node based on its ID prefix,
// mirroring the role-colour logic used in the Fyne canvas renderer.
func nodeColorHex(n DiagramNode) string {
	switch {
	case n.ID == "root":
		return "#3b82f6" // blue  – colorRoot
	case strings.HasPrefix(n.ID, "folder_src") ||
		strings.HasPrefix(n.ID, "folder_cmd") ||
		strings.HasPrefix(n.ID, "folder_app"):
		return "#10b981" // green – colorSource
	case strings.HasPrefix(n.ID, "folder_model") ||
		strings.HasPrefix(n.ID, "folder_data"):
		return "#f59e0b" // amber – colorData
	case strings.HasPrefix(n.ID, "folder_config") ||
		strings.HasPrefix(n.ID, "folder_doc"):
		return "#8b5cf6" // purple – colorConfig
	case strings.HasPrefix(n.ID, "folder_test") ||
		strings.HasPrefix(n.ID, "folder_fixture"):
		return "#ef4444" // red – colorTest
	case n.ID == "stack" || strings.HasPrefix(n.ID, "tech_"):
		return "#64748b" // slate
	default:
		return "#6b7280" // grey – colorDefault
	}
}

// svgEscape escapes minimal XML special characters in text content.
func svgEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}
