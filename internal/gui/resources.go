// ==============================================================================
// Program: resources.go
// Programmer: Ethan-da-Tech-Wizard
// Last Modified: November 18, 2026
// Purpose: Monitors system memory headroom, CPU load, and active model memory footprints to ensure local inference runs within host hardware constraints.
// ==============================================================================

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
