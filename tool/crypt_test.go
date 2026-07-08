package tool

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func Test_AES128Encrypt_AND_AES128Decrypt(t *testing.T) {
	expected := "helloworld"
	key := "8dv4byf8b9e6bc1x"
	iv := "xduio1f8a12348u4"
	encrypt, err := AES128Encrypt([]byte(expected), []byte(key), []byte(iv))
	if err != nil {
		t.Fatal(err)
	}
	decrypt, err := AES128Decrypt(encrypt, []byte(key), []byte(iv))
	if err != nil {
		t.Fatal(err)
	}
	de := string(decrypt)
	if de != expected {
		t.Fatalf("expected: %s, result: %s", expected, de)
	}
}

func TestAES128CBCDecryptRaw_tencent_doc_vector(t *testing.T) {
	// 腾讯云官方文档示例：CipherContentKey -> ContentKey
	pkey := "JduzsUuRvGVPRHvIYwLv"
	sum := sha256.Sum256([]byte(pkey))
	symKey := sum[:]
	cipherKey, _ := hex.DecodeString("68addf7984478a3e4797d3a13ecbb6fb")
	iv := make([]byte, 16)
	out, err := AES128CBCDecryptRaw(cipherKey, symKey, iv)
	if err != nil {
		t.Fatal(err)
	}
	want, _ := hex.DecodeString("bed3747b8510b040826163c04956a4c1")
	if !bytes.Equal(out, want) {
		t.Fatalf("got %x want %x", out, want)
	}
}
