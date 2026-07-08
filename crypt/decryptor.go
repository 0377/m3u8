package crypt

type Decryptor interface {
	Name() string
	ProcessKey(ctx *Context, rawKey []byte, meta *KeyMeta) (key, iv []byte, err error)
	DecryptSegment(ctx *Context, ciphertext, key, iv []byte) ([]byte, error)
	DecryptFull(ctx *Context, ciphertext []byte) (plaintext []byte, ok bool, err error)
}
