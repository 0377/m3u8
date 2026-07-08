package provider

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"testing"
)

func TestDecodeAliyunKeyResponse_binary16(t *testing.T) {
	raw, err := hex.DecodeString("bed3747b8510b040826163c04956a4c1")
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) != 16 {
		t.Fatalf("test vector must be 16 bytes, got %d", len(raw))
	}

	got, err := DecodeAliyunKeyResponse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, raw) {
		t.Fatalf("got %x want %x", got, raw)
	}
}

func TestDecodeAliyunKeyResponse_base64(t *testing.T) {
	want, err := hex.DecodeString("bed3747b8510b040826163c04956a4c1")
	if err != nil {
		t.Fatal(err)
	}
	raw := []byte(base64.StdEncoding.EncodeToString(want))

	got, err := DecodeAliyunKeyResponse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("got %x want %x", got, want)
	}
}
