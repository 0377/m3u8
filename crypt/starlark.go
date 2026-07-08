package crypt

import (
	"fmt"
	"os"
	"path/filepath"

	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"

	"github.com/0377/m3u8/tool"
)

type starlarkDecryptor struct {
	name    string
	thread  *starlark.Thread
	globals starlark.StringDict
	hasKey  bool
	hasSeg  bool
	hasFull bool
}

func newStarlarkDecryptor(path string) (Decryptor, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	thread := &starlark.Thread{Name: filepath.Base(path)}
	predeclared := starlark.StringDict{
		"aes128_cbc_decrypt": starlark.NewBuiltin("aes128_cbc_decrypt", starlarkAES128Decrypt),
	}
	globals, err := starlark.ExecFile(thread, path, src, predeclared)
	if err != nil {
		return nil, fmt.Errorf("load starlark script %s: %w", path, err)
	}
	d := &starlarkDecryptor{
		name:    filepath.Base(path),
		thread:  thread,
		globals: globals,
	}
	if fn, ok := globals["decrypt_key"]; ok {
		_, d.hasKey = fn.(starlark.Callable)
	}
	if fn, ok := globals["decrypt_segment"]; ok {
		_, d.hasSeg = fn.(starlark.Callable)
	}
	if fn, ok := globals["decrypt_full"]; ok {
		_, d.hasFull = fn.(starlark.Callable)
	}
	return d, nil
}

func (d *starlarkDecryptor) Name() string { return d.name }

func (d *starlarkDecryptor) ProcessKey(ctx *Context, rawKey []byte, meta *KeyMeta) ([]byte, []byte, error) {
	if !d.hasKey {
		return rawKey, ivBytes(meta), nil
	}
	fn := d.globals["decrypt_key"].(starlark.Callable)
	iv := ""
	if meta != nil {
		iv = meta.IV
	}
	val, err := starlark.Call(d.thread, fn, starlark.Tuple{
		starlarkBytes(rawKey),
		starlark.String(ctx.Method),
		starlark.String(metaURI(meta)),
		starlark.String(iv),
		starlark.String(ctx.M3U8URL),
	}, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("script error [%s/decrypt_key]: %w", d.name, err)
	}
	return parseKeyResult(val, rawKey, meta)
}

func (d *starlarkDecryptor) DecryptSegment(ctx *Context, ciphertext, key, iv []byte) ([]byte, error) {
	if !d.hasSeg {
		return nil, ErrNotImplemented
	}
	fn := d.globals["decrypt_segment"].(starlark.Callable)
	val, err := starlark.Call(d.thread, fn, starlark.Tuple{
		starlarkBytes(ciphertext),
		starlarkBytes(key),
		starlarkBytes(iv),
		starlark.MakeInt(ctx.SegmentIdx),
		starlark.String(ctx.SegmentURI),
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("script error [%s/decrypt_segment]: %w", d.name, err)
	}
	return starlarkToBytes(val)
}

func (d *starlarkDecryptor) DecryptFull(ctx *Context, ciphertext []byte) ([]byte, bool, error) {
	if !d.hasFull {
		return nil, false, nil
	}
	key := ctx.Key
	iv := ctx.IV
	if len(iv) == 0 {
		iv = ivBytes(&ctx.KeyMeta)
	}
	fn := d.globals["decrypt_full"].(starlark.Callable)
	val, err := starlark.Call(d.thread, fn, starlark.Tuple{
		starlarkBytes(ciphertext),
		starlark.MakeInt(ctx.SegmentIdx),
		starlark.String(ctx.SegmentURI),
		starlark.String(ctx.Method),
		starlarkBytes(key),
		starlarkBytes(iv),
	}, nil)
	if err != nil {
		return nil, true, fmt.Errorf("script error [%s/decrypt_full]: %w", d.name, err)
	}
	out, err := starlarkToBytes(val)
	return out, true, err
}

func starlarkAES128Decrypt(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var ct, key, iv starlark.Bytes
	if err := starlark.UnpackArgs(b.Name(), args, kwargs, "ciphertext", &ct, "key", &key, "iv", &iv); err != nil {
		return nil, err
	}
	out, err := tool.AES128Decrypt([]byte(ct), []byte(key), []byte(iv))
	if err != nil {
		return nil, err
	}
	return starlarkBytes(out), nil
}

func starlarkBytes(b []byte) starlark.Bytes { return starlark.Bytes(b) }

func starlarkToBytes(v starlark.Value) ([]byte, error) {
	if b, ok := v.(starlark.Bytes); ok {
		return []byte(b), nil
	}
	return nil, fmt.Errorf("expected bytes, got %s", v.Type())
}

func parseKeyResult(v starlark.Value, fallbackKey []byte, meta *KeyMeta) ([]byte, []byte, error) {
	keyVal, ivVal, ok := keyResultFields(v)
	if !ok {
		if b, err := starlarkToBytes(v); err == nil {
			return b, ivBytes(meta), nil
		}
		return nil, nil, fmt.Errorf("decrypt_key must return dict or bytes")
	}
	key, err := starlarkToBytes(keyVal)
	if err != nil {
		key = fallbackKey
	}
	iv, err := starlarkToBytes(ivVal)
	if err != nil {
		iv = ivBytes(meta)
	}
	return key, iv, nil
}

func keyResultFields(v starlark.Value) (keyVal, ivVal starlark.Value, ok bool) {
	if dict, ok := v.(*starlarkstruct.Struct); ok {
		keyVal, _ = dict.Attr("key")
		ivVal, _ = dict.Attr("iv")
		return keyVal, ivVal, true
	}
	if dict, ok := v.(*starlark.Dict); ok {
		keyVal, found, err := dict.Get(starlark.String("key"))
		if err != nil || !found {
			keyVal = starlark.None
		}
		ivVal, found, err = dict.Get(starlark.String("iv"))
		if err != nil || !found {
			ivVal = starlark.None
		}
		return keyVal, ivVal, true
	}
	return nil, nil, false
}

func ivBytes(meta *KeyMeta) []byte {
	if meta == nil || meta.IV == "" {
		return nil
	}
	return []byte(meta.IV)
}

func metaURI(meta *KeyMeta) string {
	if meta == nil {
		return ""
	}
	return meta.URI
}
