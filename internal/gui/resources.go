package gui

import (
	_ "embed"

	"fyne.io/fyne/v2"
)

//go:embed assets/icon.png
var iconPNG []byte

// appIcon is CodeLens Local's window/taskbar icon, embedded so the binary
// stays self-contained (see assets/gen_icon.go to regenerate it).
var appIcon = fyne.NewStaticResource("icon.png", iconPNG)
