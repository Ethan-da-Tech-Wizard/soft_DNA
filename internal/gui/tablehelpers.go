package gui

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// wideColumns names headers that get extra width because their cell content
// tends to be long free text rather than a short identifier.
var wideColumns = map[string]struct{}{
	"signature":   {},
	"reason":      {},
	"role":        {},
	"explanation": {},
}

// newLiveTable builds a widget.Table with a bold header row over rows, and
// returns a setter that replaces the visible rows and refreshes the table —
// used to back live text filtering without rebuilding the whole widget.
func newLiveTable(headers []string, rows [][]string) (*widget.Table, func([][]string)) {
	current := rows
	table := widget.NewTable(
		func() (int, int) { return len(current) + 1, len(headers) },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(id widget.TableCellID, o fyne.CanvasObject) {
			label := o.(*widget.Label)
			if id.Row == 0 {
				label.TextStyle = fyne.TextStyle{Bold: true}
				label.SetText(headers[id.Col])
				return
			}
			label.TextStyle = fyne.TextStyle{}
			label.SetText(current[id.Row-1][id.Col])
		},
	)
	for i, h := range headers {
		width := float32(140)
		if _, ok := wideColumns[strings.ToLower(h)]; ok {
			width = 360
		}
		table.SetColumnWidth(i, width)
	}
	setRows := func(newRows [][]string) {
		current = newRows
		table.Refresh()
	}
	return table, setRows
}

// newSimpleTable renders a static table, or a centered placeholder when
// there is nothing to show.
func newSimpleTable(headers []string, rows [][]string) fyne.CanvasObject {
	if len(rows) == 0 {
		return container.NewCenter(widget.NewLabel("No data identified."))
	}
	table, _ := newLiveTable(headers, rows)
	return table
}
