package crypt

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadConfig_parses_rules(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "decrypt.yaml")
	content := `scripts_dir: myscripts
external_timeout: 10s
rules:
  - match:
      host: "*.example.com"
      method: SAMPLE-AES
    script: myscripts/sample.py
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ScriptsDir != "myscripts" {
		t.Fatalf("scripts_dir: got %q", cfg.ScriptsDir)
	}
	if cfg.ExternalTimeout != 10*time.Second {
		t.Fatalf("timeout: got %v", cfg.ExternalTimeout)
	}
	if len(cfg.Rules) != 1 {
		t.Fatalf("rules count: %d", len(cfg.Rules))
	}
	if cfg.Rules[0].Match.Method != "SAMPLE-AES" {
		t.Fatalf("method: %q", cfg.Rules[0].Match.Method)
	}
}

func TestLoadConfig_missing_file_returns_nil(t *testing.T) {
	cfg, err := LoadConfig("/nonexistent/decrypt.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if cfg != nil {
		t.Fatal("expected nil config for missing file")
	}
}
