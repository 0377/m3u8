package crypt

import (
	"path/filepath"
	"testing"

	"github.com/0377/m3u8/tool"
)

func TestStarlarkDecryptor_key_hook(t *testing.T) {
	path := filepath.Join("testdata", "key_transform.star")
	d, err := newStarlarkDecryptor(path)
	if err != nil {
		t.Fatal(err)
	}
	raw := []byte("1234567890123456")
	key, iv, err := d.ProcessKey(&Context{Method: "AES-128"}, raw, &KeyMeta{IV: "iv"})
	if err != nil {
		t.Fatal(err)
	}
	if string(key) != "6543210987654321" {
		t.Fatalf("key transform failed: %q", key)
	}
	if string(iv) != "iv" {
		t.Fatalf("iv: %q", iv)
	}
}

func TestStarlarkDecryptor_segment_hook(t *testing.T) {
	path := filepath.Join("testdata", "key_transform.star")
	d, err := newStarlarkDecryptor(path)
	if err != nil {
		t.Fatal(err)
	}
	plain := []byte("helloworld")
	key := []byte("8dv4byf8b9e6bc1x")
	iv := []byte("xduio1f8a12348u4")
	enc, err := tool.AES128Encrypt(plain, key, iv)
	if err != nil {
		t.Fatal(err)
	}
	out, err := d.DecryptSegment(&Context{Method: "AES-128"}, enc, key, iv)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != string(plain) {
		t.Fatalf("segment decrypt failed")
	}
}
