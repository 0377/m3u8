package crypt

import (
	"encoding/hex"
	"strings"
)

// IVFromMeta decodes IV from KeyMeta. Hex strings with optional 0x prefix are
// decoded; otherwise the IV string is used as raw bytes (builtin/Starlark fallback).
func IVFromMeta(meta *KeyMeta) ([]byte, error) {
	if meta == nil || meta.IV == "" {
		return nil, nil
	}
	s := meta.IV
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		s = s[2:]
	}
	if b, err := hex.DecodeString(s); err == nil {
		return b, nil
	}
	return []byte(meta.IV), nil
}
