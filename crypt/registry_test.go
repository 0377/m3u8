package crypt

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRegistry_resolve_priority(t *testing.T) {
	dir := t.TempDir()
	scriptsDir := filepath.Join(dir, "scripts")
	if err := os.MkdirAll(scriptsDir, 0755); err != nil {
		t.Fatal(err)
	}
	autoScript := filepath.Join(scriptsDir, "AES-128.star")
	if err := os.WriteFile(autoScript, []byte(`def decrypt_key(raw_key, method, uri, iv, m3u8_url):
    return {"key": raw_key, "iv": iv}
`), 0644); err != nil {
		t.Fatal(err)
	}
	cliScript := filepath.Join(dir, "cli.star")
	if err := os.WriteFile(cliScript, []byte(`def decrypt_key(raw_key, method, uri, iv, m3u8_url):
    return {"key": b"cli-key", "iv": iv}
`), 0644); err != nil {
		t.Fatal(err)
	}

	reg, err := NewRegistry(RegistryOptions{
		ScriptsDir:    scriptsDir,
		CLIScript:     cliScript,
		ScriptsDirAbs: scriptsDir,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx := &Context{M3U8URL: "https://example.com/a.m3u8", Method: "AES-128"}
	d, err := reg.Resolve(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if d.Name() != filepath.Base(cliScript) {
		t.Fatalf("CLI should win, got %q", d.Name())
	}
}

func TestRegistry_auto_discover_by_method(t *testing.T) {
	dir := t.TempDir()
	scriptsDir := filepath.Join(dir, "scripts")
	if err := os.MkdirAll(scriptsDir, 0755); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(scriptsDir, "CUSTOM-METHOD.star")
	if err := os.WriteFile(script, []byte(`def decrypt_full(ciphertext, index, uri, method, key, iv):
    return ciphertext
`), 0644); err != nil {
		t.Fatal(err)
	}
	reg, err := NewRegistry(RegistryOptions{ScriptsDir: scriptsDir, ScriptsDirAbs: scriptsDir})
	if err != nil {
		t.Fatal(err)
	}
	ctx := &Context{M3U8URL: "https://foo.com/a.m3u8", Method: "CUSTOM-METHOD"}
	d, err := reg.Resolve(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if d.Name() != "CUSTOM-METHOD.star" {
		t.Fatalf("got %q", d.Name())
	}
}

func TestRegistry_builtin_fallback_aes128(t *testing.T) {
	reg, err := NewRegistry(RegistryOptions{ScriptsDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	ctx := &Context{M3U8URL: "https://foo.com/a.m3u8", Method: "AES-128"}
	d, err := reg.Resolve(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if d.Name() != "builtin" {
		t.Fatalf("expected builtin, got %q", d.Name())
	}
}

func TestRegistry_unknown_method_errors(t *testing.T) {
	reg, err := NewRegistry(RegistryOptions{ScriptsDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	ctx := &Context{M3U8URL: "https://foo.com/a.m3u8", Method: "UNKNOWN-X"}
	_, err = reg.Resolve(ctx)
	if err == nil {
		t.Fatal("expected error for unknown method")
	}
}

func TestRegistry_cache_same_instance(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "AES-128.star")
	if err := os.WriteFile(script, []byte(`def decrypt_key(raw_key, method, uri, iv, m3u8_url):
    return {"key": raw_key, "iv": iv}
`), 0644); err != nil {
		t.Fatal(err)
	}
	reg, err := NewRegistry(RegistryOptions{
		ScriptsDir:    dir,
		ScriptsDirAbs: dir,
		CLIScript:     script,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer reg.Close()

	ctx := &Context{M3U8URL: "https://example.com/a.m3u8", Method: "AES-128"}
	d1, err := reg.Resolve(ctx)
	if err != nil {
		t.Fatal(err)
	}
	d2, err := reg.Resolve(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if d1 != d2 {
		t.Fatal("expected same decryptor instance from cache on second Resolve")
	}
}

func TestRegistry_config_rule_resolves_relative_script(t *testing.T) {
	dir := t.TempDir()
	scriptsDir := filepath.Join(dir, "myscripts")
	if err := os.MkdirAll(scriptsDir, 0755); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(scriptsDir, "sample.py")
	if err := os.WriteFile(script, []byte("#!/usr/bin/env python3\nimport sys\nfor l in sys.stdin: pass\n"), 0755); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(dir, "decrypt.yaml")
	config := `scripts_dir: myscripts
rules:
  - match:
      method: SAMPLE-AES
    script: sample.py
`
	if err := os.WriteFile(configPath, []byte(config), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatal(err)
	}
	scriptsAbs, _ := filepath.Abs(scriptsDir)
	configAbs, _ := filepath.Abs(configPath)
	reg, err := NewRegistry(RegistryOptions{
		ScriptsDir:    "myscripts",
		ScriptsDirAbs: scriptsAbs,
		ConfigPath:    configAbs,
		Config:        cfg,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer reg.Close()

	ctx := &Context{M3U8URL: "https://example.com/a.m3u8", Method: "SAMPLE-AES"}
	d, err := reg.Resolve(ctx)
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	if d.Name() != "sample.py" {
		t.Fatalf("got %q", d.Name())
	}
}
