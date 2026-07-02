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

	want := Settings{LLMEnabled: true, LLMBinary: "/bin/llama-cli", LLMModel: "/models/m.gguf", LLMThreads: 4, LastProject: "/home/user/myproject"}
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

func TestValidateLLMConfigDisabledIsAlwaysFine(t *testing.T) {
	if got := validateLLMConfig(Settings{LLMEnabled: false}); got != nil {
		t.Fatalf("validateLLMConfig(disabled) = %v, want nil", got)
	}
}

func TestValidateLLMConfigFlagsEmptyPaths(t *testing.T) {
	got := validateLLMConfig(Settings{LLMEnabled: true})
	if len(got) != 2 {
		t.Fatalf("validateLLMConfig(empty paths) = %v, want 2 problems", got)
	}
}

func TestValidateLLMConfigFlagsMissingFiles(t *testing.T) {
	got := validateLLMConfig(Settings{LLMEnabled: true, LLMBinary: "/no/such/binary", LLMModel: "/no/such/model.gguf"})
	if len(got) != 2 {
		t.Fatalf("validateLLMConfig(missing files) = %v, want 2 problems", got)
	}
}

func TestValidateLLMConfigPassesWithRealFiles(t *testing.T) {
	dir := t.TempDir()
	binPath := filepath.Join(dir, "llama-cli")
	modelPath := filepath.Join(dir, "model.gguf")
	if err := os.WriteFile(binPath, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(modelPath, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	got := validateLLMConfig(Settings{LLMEnabled: true, LLMBinary: binPath, LLMModel: modelPath})
	if len(got) != 0 {
		t.Fatalf("validateLLMConfig(real files) = %v, want none", got)
	}
}
