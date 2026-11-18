// Program: soft_DNA - resources_test.go
// Programmer: Ethan-da-Tech-Wizard (written with Claude, an AI assistant)
// Last Modified: 2026-11-18
// Purpose: Tests that the embedded application icon is a decodable, square, reasonably sized PNG with the expected resource name.

package gui

import (
	"bytes"
	"image/png"
	"testing"
)

// TestAppIconIsValidPNG decodes the embedded icon and checks it is square, at least 32 pixels wide, and named icon.png.
func TestAppIconIsValidPNG(t *testing.T) {
	img, err := png.Decode(bytes.NewReader(iconPNG))
	if err != nil {
		t.Fatalf("embedded icon.png failed to decode: %v", err)
	}
	b := img.Bounds()
	if b.Dx() != b.Dy() {
		t.Fatalf("icon should be square, got %dx%d", b.Dx(), b.Dy())
	}
	if b.Dx() < 32 {
		t.Fatalf("icon too small for a usable app icon: %dx%d", b.Dx(), b.Dy())
	}
	if appIcon.Name() != "icon.png" {
		t.Fatalf("appIcon.Name() = %q, want %q", appIcon.Name(), "icon.png")
	}
}
