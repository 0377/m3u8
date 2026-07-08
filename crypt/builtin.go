package crypt

import "github.com/0377/m3u8/tool"

type BuiltinDecryptor struct{}

var _ Decryptor = (*BuiltinDecryptor)(nil)

func (d *BuiltinDecryptor) Name() string { return "builtin" }

func (d *BuiltinDecryptor) ProcessKey(_ *Context, rawKey []byte, meta *KeyMeta) ([]byte, []byte, error) {
	iv, err := IVFromMeta(meta)
	if err != nil {
		return nil, nil, err
	}
	return rawKey, iv, nil
}

func (d *BuiltinDecryptor) DecryptSegment(_ *Context, ciphertext, key, iv []byte) ([]byte, error) {
	return tool.AES128Decrypt(ciphertext, key, iv)
}

func (d *BuiltinDecryptor) DecryptFull(_ *Context, _ []byte) ([]byte, bool, error) {
	return nil, false, nil
}
