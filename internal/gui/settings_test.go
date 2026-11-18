// Program: soft_DNA - settings_test.go
// Programmer: Ethan-da-Tech-Wizard (written with Claude, an AI assistant)
// Last Modified: 2026-11-18
// Purpose: Tests settings persistence (save/load round trip, defaults, corrupt file), LLM configuration validation, and the automatic inference timeout rules.

package gui

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestSettingsSaveAndLoadRoundTrip saves settings into a temporary config directory and checks that loading returns the same values.
func TestSettingsSaveAndLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	orig := configDir
	configDir = func() (string, error) { return dir, nil }
	defer func() { configDir = orig }()

	want := Settings{LLMEnabled: true, LLMBinary: "/bin/llama-cli", LLMModel: "/models/m.gguf", LLMThreads: 4, LLMMaxFiles: 25, LastProject: "/home/user/myproject"}
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

// TestLoadSettingsDefaultsWhenMissing checks that with no settings file the LLM is off and the thread count is positive.
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

// TestLoadSettingsDefaultsOnCorruptFile checks that an unreadable settings file falls back to defaults instead of failing.
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

// TestValidateLLMConfigDisabledIsAlwaysFine checks that validation reports nothing while the LLM is disabled.
func TestValidateLLMConfigDisabledIsAlwaysFine(t *testing.T) {
	if got := validateLLMConfig(Settings{LLMEnabled: false}); got != nil {
		t.Fatalf("validateLLMConfig(disabled) = %v, want nil", got)
	}
}

// TestValidateLLMConfigFlagsEmptyPaths checks that empty binary and model paths each produce a problem.
func TestValidateLLMConfigFlagsEmptyPaths(t *testing.T) {
	got := validateLLMConfig(Settings{LLMEnabled: true})
	if len(got) != 2 {
		t.Fatalf("validateLLMConfig(empty paths) = %v, want 2 problems", got)
	}
}

// TestValidateLLMConfigFlagsMissingFiles checks that paths pointing at non-existent files each produce a problem.
func TestValidateLLMConfigFlagsMissingFiles(t *testing.T) {
	got := validateLLMConfig(Settings{LLMEnabled: true, LLMBinary: "/no/such/binary", LLMModel: "/no/such/model.gguf"})
	if len(got) != 2 {
		t.Fatalf("validateLLMConfig(missing files) = %v, want 2 problems", got)
	}
}

// TestValidateLLMConfigPassesWithRealFiles checks that existing binary and model files produce no problems.
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

// TestLoadSettingsDefaultsMaxFilesTenWhenMissing checks that the max-files default is 10 when no settings file exists.
func TestLoadSettingsDefaultsMaxFilesTenWhenMissing(t *testing.T) {
	orig := configDir
	configDir = func() (string, error) { return t.TempDir(), nil }
	defer func() { configDir = orig }()

	if got := LoadSettings().LLMMaxFiles; got != 10 {
		t.Fatalf("LLMMaxFiles default = %d, want 10", got)
	}
}

// TestLoadSettingsPreservesExplicitZeroMaxFiles checks that a saved max-files value of 0 (no limit) is not replaced by the default.
func TestLoadSettingsPreservesExplicitZeroMaxFiles(t *testing.T) {
	dir := t.TempDir()
	orig := configDir
	configDir = func() (string, error) { return dir, nil }
	defer func() { configDir = orig }()

	if err := (Settings{LLMMaxFiles: 0}).Save(); err != nil {
		t.Fatal(err)
	}
	if got := LoadSettings().LLMMaxFiles; got != 0 {
		t.Fatalf("LLMMaxFiles = %d, want 0 (uncapped) to survive an explicit save", got)
	}
}

// TestEffectiveLLMTimeoutExplicitSettingWins checks that a user-set timeout is used as is.
func TestEffectiveLLMTimeoutExplicitSettingWins(t *testing.T) {
	s := Settings{LLMTimeoutSec: 55}
	if got := s.EffectiveLLMTimeout(); got != 55*time.Second {
		t.Fatalf("EffectiveLLMTimeout = %v, want 55s", got)
	}
}

// TestEffectiveLLMTimeoutSmallOrMissingModel checks that a missing or small model file gets the 120 second default.
func TestEffectiveLLMTimeoutSmallOrMissingModel(t *testing.T) {
	// Missing model file: fall back to the small-model default.
	s := Settings{LLMModel: "/no/such/model.gguf"}
	if got := s.EffectiveLLMTimeout(); got != 120*time.Second {
		t.Fatalf("EffectiveLLMTimeout(missing) = %v, want 120s", got)
	}

	// Real but small model file: also the small-model default.
	small := filepath.Join(t.TempDir(), "small.gguf")
	if err := os.WriteFile(small, []byte("tiny"), 0o644); err != nil {
		t.Fatal(err)
	}
	s = Settings{LLMModel: small}
	if got := s.EffectiveLLMTimeout(); got != 120*time.Second {
		t.Fatalf("EffectiveLLMTimeout(small) = %v, want 120s", got)
	}
}

// TestEffectiveLLMTimeoutLargeModelGetsMinutes checks that a model file of 3 GiB or more gets the 420 second default.
func TestEffectiveLLMTimeoutLargeModelGetsMinutes(t *testing.T) {
	// A sparse file of 3 GiB: no real disk usage, but Stat reports the size.
	big := filepath.Join(t.TempDir(), "qwen7b.gguf")
	f, err := os.Create(big)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Truncate(largeModelBytes); err != nil {
		f.Close()
		t.Skipf("filesystem does not support sparse truncate: %v", err)
	}
	f.Close()

	s := Settings{LLMModel: big}
	if got := s.EffectiveLLMTimeout(); got != 420*time.Second {
		t.Fatalf("EffectiveLLMTimeout(7B-sized) = %v, want 420s", got)
	}
}
