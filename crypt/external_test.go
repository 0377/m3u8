package crypt

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"path/filepath"
	"strings"
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

func TestExternalDecryptor_timeout(t *testing.T) {
	path := filepath.Join("testdata", "hang_decrypt.py")
	d, err := newExternalDecryptor(path, 100*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	defer d.(*externalDecryptor).Close()

	_, err = d.DecryptSegment(&Context{SegmentIdx: 0}, []byte("x"), []byte("k"), []byte("iv"))
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(err.Error(), "timeout") {
		t.Fatalf("expected timeout, got: %v", err)
	}
}

func TestExternalDecryptor_restarts_after_crash(t *testing.T) {
	path := filepath.Join("testdata", "crash_once.py")
	d, err := newExternalDecryptor(path, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer d.(*externalDecryptor).Close()

	_, err = d.DecryptSegment(&Context{SegmentIdx: 0}, []byte("first"), nil, nil)
	if err == nil {
		t.Fatal("expected error on first call after crash")
	}

	out, err := d.DecryptSegment(&Context{SegmentIdx: 1}, []byte("second"), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "second" {
		t.Fatalf("got %q", out)
	}
}

func TestReadExternalLine_large(t *testing.T) {
	payload := strings.Repeat("A", 128*1024)
	line := []byte(payload + "\n")
	r := bufio.NewReader(bytes.NewReader(line))
	got, err := readExternalLine(r, maxExternalLineBytes)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(payload) {
		t.Fatalf("got len %d want %d", len(got), len(payload))
	}
}

func TestDecodeExternalIV(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte{1, 2, 3})
	got := decodeExternalIV("fallback", encoded)
	if string(got) != string([]byte{1, 2, 3}) {
		t.Fatalf("base64 decode failed: %v", got)
	}
	got = decodeExternalIV("fallback", "plain")
	if string(got) != "plain" {
		t.Fatalf("plain fallback failed: %q", got)
	}
	got = decodeExternalIV("fallback", "")
	if string(got) != "fallback" {
		t.Fatalf("empty uses fallback: %q", got)
	}
}
