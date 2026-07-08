package crypt

import (
	"path/filepath"
	"testing"
	"time"
)

func TestExternalDecryptor_key_and_segment(t *testing.T) {
	path := filepath.Join("testdata", "echo_decrypt.py")
	d, err := newExternalDecryptor(path, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer d.(*externalDecryptor).Close()

	raw := []byte("1234567890123456")
	key, iv, err := d.ProcessKey(&Context{Method: "CUSTOM", M3U8URL: "https://x.com/a.m3u8"}, raw, &KeyMeta{IV: "iv"})
	if err != nil {
		t.Fatal(err)
	}
	if string(key) != string(raw) {
		t.Fatalf("key mismatch")
	}

	out, err := d.DecryptSegment(&Context{SegmentIdx: 0, SegmentURI: "s.ts"}, []byte("cipher"), key, iv)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "cipher" {
		t.Fatalf("segment echo failed: %q", out)
	}
}
