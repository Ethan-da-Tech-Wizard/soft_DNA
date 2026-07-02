package gui

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSettingsSaveAndLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	orig := configDir
	configDir = func() (string, error) { return dir, nil }
	defer func() { configDir = orig }()

	want := Settings{LLMEnabled: true, LLMBinary: "/bin/llama-cli", LLMModel: "/models/m.gguf", LLMThreads: 4}
	if err := want.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	if _, err := filepath.Abs(filepath.Join(dir, "codelens", "settings.json")); err != nil {
		t.Fatal(err)
	}

	got := LoadSettings()
	if got != want {
		t.Fatalf("LoadSettings() = %+v, want %+v", got, want)
	}
}

func TestLoadSettingsDefaultsWhenMissing(t *testing.T) {
	orig := configDir
	configDir = func() (string, error) { return t.TempDir(), nil }
	defer func() { configDir = orig }()

	got := LoadSettings()
	if got.LLMEnabled {
		t.Fatal("LLMEnabled should default to false")
	}
	if got.LLMThreads <= 0 {
		t.Fatalf("LLMThreads default = %d, want > 0", got.LLMThreads)
	}
}

func TestLoadSettingsDefaultsOnCorruptFile(t *testing.T) {
	dir := t.TempDir()
	orig := configDir
	configDir = func() (string, error) { return dir, nil }
	defer func() { configDir = orig }()

	s := Settings{}
	_ = s.Save()
	path, _ := settingsPath()
	if err := os.WriteFile(path, []byte("not json"), 0o644); err != nil {
		t.Fatal(err)
	}

	got := LoadSettings()
	if got.LLMThreads <= 0 {
		t.Fatalf("LLMThreads default = %d, want > 0 even after corrupt file", got.LLMThreads)
	}
}
