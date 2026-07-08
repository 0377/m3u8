package crypt

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"sync"
	"testing"

	"go.starlark.net/starlark"

	"github.com/0377/m3u8/tool"
)

func evalStarlark(t *testing.T, src string) starlark.StringDict {
	t.Helper()
	thread := &starlark.Thread{Name: "test"}
	globals, err := starlark.ExecFile(thread, "test.star", src, starlarkPredeclared())
	if err != nil {
		t.Fatal(err)
	}
	return globals
}

func TestStarlark_sha256(t *testing.T) {
	globals := evalStarlark(t, `result = sha256(b"JduzsUuRvGVPRHvIYwLv")`)
	out, err := starlarkToBytes(globals["result"])
	if err != nil {
		t.Fatal(err)
	}
	want := sha256.Sum256([]byte("JduzsUuRvGVPRHvIYwLv"))
	if !bytes.Equal(out, want[:]) {
		t.Fatalf("got %x want %x", out, want[:])
	}
}

func TestStarlark_aes_cbc_decrypt_zero_iv_tencent_doc_vector(t *testing.T) {
	// 腾讯云官方文档示例：CipherContentKey -> ContentKey
	globals := evalStarlark(t, `
pkey = b"JduzsUuRvGVPRHvIYwLv"
sym_key = sha256(pkey)
cipher_key = hex_decode("68addf7984478a3e4797d3a13ecbb6fb")
result = aes_cbc_decrypt_zero_iv(cipher_key, sym_key)
`)
	out, err := starlarkToBytes(globals["result"])
	if err != nil {
		t.Fatal(err)
	}
	want, _ := hex.DecodeString("bed3747b8510b040826163c04956a4c1")
	if !bytes.Equal(out, want) {
		t.Fatalf("got %x want %x", out, want)
	}
}

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

func TestStarlarkDecryptor_concurrent_segment(t *testing.T) {
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

	const workers = 16
	var wg sync.WaitGroup
	errCh := make(chan error, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			out, err := d.DecryptSegment(&Context{Method: "AES-128", SegmentIdx: idx}, enc, key, iv)
			if err != nil {
				errCh <- err
				return
			}
			if string(out) != string(plain) {
				errCh <- err
			}
		}(i)
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatal(err)
		}
	}
}
