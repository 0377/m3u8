package crypt

import (
	"fmt"
	"time"
)

func newExternalDecryptor(path string, timeout time.Duration) (Decryptor, error) {
	return nil, fmt.Errorf("external runner not implemented")
}
