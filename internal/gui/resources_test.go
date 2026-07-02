package gui

import (
	"bytes"
	"image/png"
	"testing"
)

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
