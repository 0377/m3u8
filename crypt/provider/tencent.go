package provider

import (
	"crypto/sha256"
	"errors"

	"github.com/0377/m3u8/crypt"
	"github.com/0377/m3u8/tool"
)

var errTencentKeyDecrypt = errors.New("key 解密失败，请检查 pkey 是否正确")

type tencentDecryptor struct{}

var _ crypt.Decryptor = (*tencentDecryptor)(nil)

func NewTencentDecryptor() crypt.Decryptor {
	return &tencentDecryptor{}
}

func DecryptTencentContentKey(cipherKey []byte, pkey string) ([]byte, error) {
	if pkey == "" {
		return nil, errTencentKeyDecrypt
	}
	sum := sha256.Sum256([]byte(pkey))
	symKey := sum[:]
	zeroIV := make([]byte, 16)
	out, err := tool.AES128CBCDecryptRaw(cipherKey, symKey, zeroIV)
	if err != nil {
		return nil, errTencentKeyDecrypt
	}
	return out, nil
}

func (d *tencentDecryptor) Name() string { return IDTencentSimpleAES }

func (d *tencentDecryptor) ProcessKey(ctx *crypt.Context, rawKey []byte, meta *crypt.KeyMeta) ([]byte, []byte, error) {
	pkey := ""
	if ctx != nil {
		pkey = ctx.Params.Pkey
	}
	key, err := DecryptTencentContentKey(rawKey, pkey)
	if err != nil {
		return nil, nil, err
	}
	iv, err := crypt.IVFromMeta(meta)
	if err != nil {
		return nil, nil, err
	}
	return key, iv, nil
}

func (d *tencentDecryptor) DecryptSegment(_ *crypt.Context, _, _, _ []byte) ([]byte, error) {
	return nil, crypt.ErrNotImplemented
}

func (d *tencentDecryptor) DecryptFull(_ *crypt.Context, _ []byte) ([]byte, bool, error) {
	return nil, false, nil
}
