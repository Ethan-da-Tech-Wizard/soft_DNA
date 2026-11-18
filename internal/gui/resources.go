// Program: soft_DNA - resources.go
// Programmer: Ethan-da-Tech-Wizard (written with Claude, an AI assistant)
// Last Modified: 2026-11-18
// Purpose: Embeds the generated icon.png into the binary and exposes it as the Fyne application icon resource.

package gui

import (
	_ "embed"

	"fyne.io/fyne/v2"
)

// iconPNG holds the raw bytes of assets/icon.png, embedded at build time.
//go:embed assets/icon.png
var iconPNG []byte

// appIcon is CodeLens Local's window/taskbar icon, embedded so the binary
// stays self-contained (see assets/gen_icon.go to regenerate it).
var appIcon = fyne.NewStaticResource("icon.png", iconPNG)
