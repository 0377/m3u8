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

func TestAES128Decrypt_invalidCiphertext(t *testing.T) {
	key := []byte("8dv4byf8b9e6bc1x")
	iv := []byte("xduio1f8a12348u4")

	_, err := AES128Decrypt([]byte("short"), key, iv)
	if err == nil {
		t.Fatal("expected error for ciphertext not multiple of block size")
	}

	_, err = AES128Decrypt(nil, key, iv)
	if err == nil {
		t.Fatal("expected error for empty ciphertext")
	}
}

func Test_pkcs5UnPadding_invalid(t *testing.T) {
	const blockSize = 16

	_, err := pkcs5UnPadding(nil, blockSize)
	if err == nil {
		t.Fatal("expected error for empty input")
	}

	_, err = pkcs5UnPadding([]byte{0}, blockSize)
	if err == nil {
		t.Fatal("expected error for padding byte 0")
	}

	data := make([]byte, blockSize)
	data[blockSize-1] = byte(blockSize + 1)
	_, err = pkcs5UnPadding(data, blockSize)
	if err == nil {
		t.Fatal("expected error for padding byte > blockSize")
	}

	_, err = pkcs5UnPadding([]byte{5, 3}, blockSize)
	if err == nil {
		t.Fatal("expected error for padding byte > length")
	}

	_, err = pkcs5UnPadding([]byte{5, 5, 5, 5, 4}, blockSize)
	if err == nil {
		t.Fatal("expected error for inconsistent padding block")
	}
}

func TestAES128Decrypt_invalidPKCS7Padding(t *testing.T) {
	key := []byte("8dv4byf8b9e6bc1x")
	iv := []byte("xduio1f8a12348u4")
	encrypt, err := AES128Encrypt([]byte("helloworld"), key, iv)
	if err != nil {
		t.Fatal(err)
	}
	encrypt[len(encrypt)-1] ^= 0xff
	_, err = AES128Decrypt(encrypt, key, iv)
	if err == nil {
		t.Fatal("expected error for invalid PKCS7 padding")
	}
}

func TestAES128Decrypt_shortIV(t *testing.T) {
	key := []byte("1234567890123456")
	ciphertext := make([]byte, 16)
	_, err := AES128Decrypt(ciphertext, key, []byte("short"))
	if err == nil {
		t.Fatal("expected error for short IV")
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
