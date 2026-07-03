package gui

import (
	"os"
	"strings"
	"testing"
)

func TestExportDiagramSVG(t *testing.T) {
	layout := DiagramLayout{
		Nodes: []DiagramNode{
			{ID: "root", Label: "Project Root", X: 100, Y: 0, W: 180, H: 44},
			{ID: "folder_src", Label: "src: source", X: 0, Y: 100, W: 180, H: 44},
			{ID: "stack", Label: "Detected Stack", X: 200, Y: 100, W: 180, H: 44},
		},
		Edges: []DiagramEdge{
			{From: "root", To: "folder_src"},
			{From: "root", To: "stack"},
		},
		Width:  400,
		Height: 200,
	}

	tmp := t.TempDir()
	outPath := tmp + "/architecture.svg"

	if err := ExportDiagramSVG(layout, outPath); err != nil {
		t.Fatalf("ExportDiagramSVG returned error: %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("SVG file not written: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "<svg") {
		t.Error("SVG output missing <svg> element")
	}
	if !strings.Contains(content, "Project Root") {
		t.Error("SVG output missing root node label")
	}
	if !strings.Contains(content, "</svg>") {
		t.Error("SVG output not properly closed")
	}
	if !strings.Contains(content, "<line") {
		t.Error("SVG output missing edge <line> elements")
	}
}

func TestExportDiagramSVGEmpty(t *testing.T) {
	tmp := t.TempDir()
	outPath := tmp + "/empty.svg"

	if err := ExportDiagramSVG(DiagramLayout{}, outPath); err != nil {
		t.Fatalf("ExportDiagramSVG(empty) returned error: %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("SVG file not written: %v", err)
	}
	if !strings.Contains(string(data), "<svg") {
		t.Error("empty layout SVG missing <svg> element")
	}
}
