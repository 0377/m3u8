package provider

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/0377/m3u8/crypt"
)

var errAliyunKeyLength = errors.New("aliyun key must be 16 bytes")

type aliyunDecryptor struct{}

var _ crypt.Decryptor = (*aliyunDecryptor)(nil)

func NewAliyunDecryptor() crypt.Decryptor {
	return &aliyunDecryptor{}
}

func DecodeAliyunKeyResponse(raw []byte) ([]byte, error) {
	if len(raw) == 16 {
		return raw, nil
	}
	decoded, err := base64.StdEncoding.DecodeString(string(bytes.TrimSpace(raw)))
	if err != nil {
		return nil, err
	}
	if len(decoded) != 16 {
		return nil, fmt.Errorf("%w: got %d", errAliyunKeyLength, len(decoded))
	}
	return decoded, nil
}

func (d *aliyunDecryptor) Name() string { return IDAliyunHLSStandard }

func (d *aliyunDecryptor) ProcessKey(_ *crypt.Context, rawKey []byte, meta *crypt.KeyMeta) ([]byte, []byte, error) {
	key, err := DecodeAliyunKeyResponse(rawKey)
	if err != nil {
		return nil, nil, err
	}
	iv, err := crypt.IVFromMeta(meta)
	if err != nil {
		return nil, nil, err
	}
	return key, iv, nil
}

func (d *aliyunDecryptor) DecryptSegment(_ *crypt.Context, _, _, _ []byte) ([]byte, error) {
	return nil, crypt.ErrNotImplemented
}

func (d *aliyunDecryptor) DecryptFull(_ *crypt.Context, _ []byte) ([]byte, bool, error) {
	return nil, false, nil
}
