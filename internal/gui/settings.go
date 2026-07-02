// Package gui implements CodeLens Local's native desktop window (Fyne).
// It is a thin presentation layer over internal/cli's exported pipeline API
// (RunPipeline, WriteMarkdownReport, Answer) — no scanning, indexing, or LLM
// orchestration logic is duplicated here.
package gui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
)

// Settings holds user-configurable options persisted between runs, stored as
// plain JSON under the OS user config directory (e.g. ~/.config/codelens on
// Linux). No network call is ever made to load, save, or use these values.
type Settings struct {
	LLMEnabled bool   `json:"llm_enabled"`
	LLMBinary  string `json:"llm_binary"`
	LLMModel   string `json:"llm_model"`
	LLMThreads int    `json:"llm_threads"`
}

// configDir resolves the OS user config directory; a package variable so
// tests can redirect it without touching the real home directory.
var configDir = os.UserConfigDir

func settingsPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "codelens", "settings.json"), nil
}

// LoadSettings reads persisted settings from the local config directory. A
// missing or unreadable file is not an error; it yields sane defaults
// (LLM disabled, thread count matching the machine).
func LoadSettings() Settings {
	defaults := Settings{LLMThreads: runtime.NumCPU()}

	path, err := settingsPath()
	if err != nil {
		return defaults
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return defaults
	}
	var s Settings
	if err := json.Unmarshal(data, &s); err != nil {
		return defaults
	}
	if s.LLMThreads <= 0 {
		s.LLMThreads = defaults.LLMThreads
	}
	return s
}

// Save persists s as JSON in the local config directory, creating it if
// needed. This is the only disk write Settings performs; nothing is ever
// uploaded anywhere.
func (s Settings) Save() error {
	path, err := settingsPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
