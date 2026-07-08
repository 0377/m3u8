package crypt

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/0377/m3u8/tool"
)

func TestService_decrypt_segment_builtin_fallback(t *testing.T) {
	reg, err := NewRegistry(RegistryOptions{ScriptsDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	svc := NewService(reg)
	plain := []byte("helloworld")
	key := []byte("8dv4byf8b9e6bc1x")
	iv := []byte("xduio1f8a12348u4")
	enc, err := tool.AES128Encrypt(plain, key, iv)
	if err != nil {
		t.Fatal(err)
	}
	ctx := &Context{Method: "AES-128", M3U8URL: "https://x.com/a.m3u8"}
	out, err := svc.DecryptSegment(ctx, enc, key, iv)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != string(plain) {
		t.Fatalf("fallback decrypt failed")
	}
}

func TestService_decrypt_segment_full_hook(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "FULL.star")
	starScript := `def decrypt_full(ciphertext, index, uri, method, key, iv):
    parts = []
    for b in b"ok:".elems():
        parts.append(b)
    for b in ciphertext.elems():
        parts.append(b)
    return bytes(parts)
`
	if err := os.WriteFile(script, []byte(starScript), 0644); err != nil {
		t.Fatal(err)
	}
	reg, err := NewRegistry(RegistryOptions{ScriptsDir: dir, CLIScript: script, ScriptsDirAbs: dir})
	if err != nil {
		t.Fatal(err)
	}
	svc := NewService(reg)
	ctx := &Context{Method: "CUSTOM", M3U8URL: "https://x.com/a.m3u8"}
	out, err := svc.DecryptSegment(ctx, []byte("data"), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "ok:data" {
		t.Fatalf("full hook failed: %q", out)
	}
}
