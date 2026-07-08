package tool

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"fmt"
)

func AES128Encrypt(origData, key, iv []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	blockSize := block.BlockSize()
	if len(iv) == 0 {
		iv = key
	}
	origData = pkcs5Padding(origData, blockSize)
	blockMode := cipher.NewCBCEncrypter(block, iv[:blockSize])
	crypted := make([]byte, len(origData))
	blockMode.CryptBlocks(crypted, origData)
	return crypted, nil
}

func AES128Decrypt(crypted, key, iv []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	blockSize := block.BlockSize()
	if len(iv) == 0 {
		iv = key
	}
	blockMode := cipher.NewCBCDecrypter(block, iv[:blockSize])
	origData := make([]byte, len(crypted))
	blockMode.CryptBlocks(origData, crypted)
	origData = pkcs5UnPadding(origData)
	return origData, nil
}

func pkcs5Padding(cipherText []byte, blockSize int) []byte {
	padding := blockSize - len(cipherText)%blockSize
	padText := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(cipherText, padText...)
}

func pkcs5UnPadding(origData []byte) []byte {
	length := len(origData)
	unPadding := int(origData[length-1])
	return origData[:(length - unPadding)]
}

// AES128CBCDecryptRaw decrypts one or more AES-CBC blocks without PKCS7 unpadding.
//
// The name follows this package's AES128* convention, but key may be 16, 24, or 32 bytes
// (AES-128, AES-192, or AES-256). Tencent Cloud VOD SimpleAES uses a 32-byte SHA256 digest
// as the key.
//
// When iv is nil or empty, a zero IV is used. This differs from AES128Decrypt, which falls
// back to key-as-IV when iv is omitted.
func AES128CBCDecryptRaw(ciphertext, key, iv []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	if len(ciphertext)%block.BlockSize() != 0 {
		return nil, fmt.Errorf("ciphertext length %d is not a multiple of block size", len(ciphertext))
	}
	if len(iv) == 0 {
		iv = make([]byte, block.BlockSize())
	}
	out := make([]byte, len(ciphertext))
	mode := cipher.NewCBCDecrypter(block, iv[:block.BlockSize()])
	mode.CryptBlocks(out, ciphertext)
	return out, nil
}
