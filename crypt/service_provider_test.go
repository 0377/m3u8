package crypt_test

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/0377/m3u8/crypt"
	_ "github.com/0377/m3u8/crypt/provider"
	"github.com/0377/m3u8/crypt/provider"
)

func TestService_processKey_tencent_provider(t *testing.T) {
	pkey := "JduzsUuRvGVPRHvIYwLv"
	cipherKey, err := hex.DecodeString("68addf7984478a3e4797d3a13ecbb6fb")
	if err != nil {
		t.Fatal(err)
	}
	want, err := hex.DecodeString("bed3747b8510b040826163c04956a4c1")
	if err != nil {
		t.Fatal(err)
	}

	reg, err := crypt.NewRegistry(crypt.RegistryOptions{ScriptsDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	svc := crypt.NewService(reg, crypt.ServiceProviderOptions{
		ActiveID: provider.IDTencentSimpleAES,
		Params:   crypt.ProviderParams{Pkey: pkey},
	})

	ctx := &crypt.Context{
		M3U8URL: "https://example.vod2.myqcloud.com/a.m3u8",
		Method:  "AES-128",
	}
	meta := &crypt.KeyMeta{IV: "0x0102030405060708090a0b0c0d0e0f10"}
	material, err := svc.ProcessKey(ctx, cipherKey, meta)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(material.Key, want) {
		t.Fatalf("key: got %x want %x", material.Key, want)
	}
	wantIV, err := hex.DecodeString("0102030405060708090a0b0c0d0e0f10")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(material.IV, wantIV) {
		t.Fatalf("iv: got %x want %x", material.IV, wantIV)
	}
	if ctx.Provider != provider.IDTencentSimpleAES {
		t.Fatalf("ctx.Provider: got %q want %q", ctx.Provider, provider.IDTencentSimpleAES)
	}
	if ctx.Params.Pkey != pkey {
		t.Fatalf("ctx.Params.Pkey: got %q want %q", ctx.Params.Pkey, pkey)
	}
}
