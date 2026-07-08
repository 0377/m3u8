package provider

import (
	"encoding/hex"
	"strings"

	"github.com/0377/m3u8/crypt"
)

func ivFromMeta(meta *crypt.KeyMeta) ([]byte, error) {
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
	// Fallback: match builtin/Starlark ivBytes (raw IV string bytes).
	return []byte(meta.IV), nil
}
