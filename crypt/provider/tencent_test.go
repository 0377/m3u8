package provider

import (
	"bytes"
	"encoding/hex"
	"testing"
)

func TestDecryptTencentContentKey_doc_vector(t *testing.T) {
	pkey := "JduzsUuRvGVPRHvIYwLv"
	cipherKey, err := hex.DecodeString("68addf7984478a3e4797d3a13ecbb6fb")
	if err != nil {
		t.Fatal(err)
	}
	want, err := hex.DecodeString("bed3747b8510b040826163c04956a4c1")
	if err != nil {
		t.Fatal(err)
	}

	got, err := DecryptTencentContentKey(cipherKey, pkey)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("got %x want %x", got, want)
	}
}
