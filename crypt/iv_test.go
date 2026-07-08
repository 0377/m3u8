package crypt

import (
	"bytes"
	"encoding/hex"
	"testing"
)

func TestIVFromMeta_hex_with_prefix(t *testing.T) {
	meta := &KeyMeta{IV: "0x0102030405060708090a0b0c0d0e0f10"}
	got, err := IVFromMeta(meta)
	if err != nil {
		t.Fatal(err)
	}
	want, _ := hex.DecodeString("0102030405060708090a0b0c0d0e0f10")
	if !bytes.Equal(got, want) {
		t.Fatalf("got %x want %x", got, want)
	}
}

func TestIVFromMeta_raw_string_fallback(t *testing.T) {
	meta := &KeyMeta{IV: "hooked-iv"}
	got, err := IVFromMeta(meta)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "hooked-iv" {
		t.Fatalf("got %q want %q", got, "hooked-iv")
	}
}

func TestIVFromMeta_empty(t *testing.T) {
	got, err := IVFromMeta(nil)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}
