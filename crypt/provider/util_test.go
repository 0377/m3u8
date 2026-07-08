package provider

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/0377/m3u8/crypt"
)

func TestIvFromMeta_hex_with_prefix(t *testing.T) {
	meta := &crypt.KeyMeta{IV: "0x0102030405060708090a0b0c0d0e0f10"}
	got, err := ivFromMeta(meta)
	if err != nil {
		t.Fatal(err)
	}
	want, _ := hex.DecodeString("0102030405060708090a0b0c0d0e0f10")
	if !bytes.Equal(got, want) {
		t.Fatalf("got %x want %x", got, want)
	}
}

func TestIvFromMeta_raw_string_fallback(t *testing.T) {
	meta := &crypt.KeyMeta{IV: "hooked-iv"}
	got, err := ivFromMeta(meta)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "hooked-iv" {
		t.Fatalf("got %q want %q", got, "hooked-iv")
	}
}

func TestIvFromMeta_empty(t *testing.T) {
	got, err := ivFromMeta(nil)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}
