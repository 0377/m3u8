package crypt

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/0377/m3u8/tool"
)

func TestBuiltinDecryptor_ProcessKey_passthrough(t *testing.T) {
	d := &BuiltinDecryptor{}
	raw := []byte("1234567890123456")
	meta := &KeyMeta{IV: "hooked-iv"}
	key, iv, err := d.ProcessKey(&Context{}, raw, meta)
	if err != nil {
		t.Fatal(err)
	}
	if string(key) != string(raw) {
		t.Fatalf("key passthrough failed")
	}
	if string(iv) != meta.IV {
		t.Fatalf("iv passthrough failed")
	}
}

func TestBuiltinDecryptor_ProcessKey_hex_iv(t *testing.T) {
	d := &BuiltinDecryptor{}
	raw := []byte("1234567890123456")
	meta := &KeyMeta{IV: "0x0102030405060708090a0b0c0d0e0f10"}
	_, iv, err := d.ProcessKey(&Context{}, raw, meta)
	if err != nil {
		t.Fatal(err)
	}
	want, _ := hex.DecodeString("0102030405060708090a0b0c0d0e0f10")
	if !bytes.Equal(iv, want) {
		t.Fatalf("got %x want %x", iv, want)
	}
}

func TestBuiltinDecryptor_DecryptSegment_aes128(t *testing.T) {
	plain := []byte("helloworld")
	key := []byte("8dv4byf8b9e6bc1x")
	iv := []byte("xduio1f8a12348u4")
	enc, err := tool.AES128Encrypt(plain, key, iv)
	if err != nil {
		t.Fatal(err)
	}
	d := &BuiltinDecryptor{}
	out, err := d.DecryptSegment(&Context{Method: "AES-128"}, enc, key, iv)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != string(plain) {
		t.Fatalf("expected %q, got %q", plain, out)
	}
}

func TestBuiltinDecryptor_DecryptFull_not_implemented(t *testing.T) {
	d := &BuiltinDecryptor{}
	_, ok, err := d.DecryptFull(&Context{}, []byte("x"))
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("builtin should not implement full hook")
	}
}
