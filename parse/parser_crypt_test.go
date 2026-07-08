package parse

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0377/m3u8/crypt"
)

func TestFromURL_aes128_without_cryptSvc(t *testing.T) {
	rawKey := []byte("1234567890123456")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/playlist.m3u8":
			_, _ = fmt.Fprintf(w, `#EXTM3U
#EXT-X-VERSION:3
#EXT-X-TARGETDURATION:10
#EXT-X-MEDIA-SEQUENCE:0
#EXT-X-KEY:METHOD=AES-128,URI="key.bin",IV=0x0102030405060708090a0b0c0d0e0f10
#EXTINF:10.0,
seg0.ts
#EXT-X-ENDLIST
`)
		case "/key.bin":
			_, _ = w.Write(rawKey)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	result, err := FromURL(srv.URL+"/playlist.m3u8", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	mat, ok := result.Keys[1]
	if !ok {
		t.Fatal("expected key at index 1")
	}
	if string(mat.Key) != string(rawKey) {
		t.Fatalf("key: got %q want %q", mat.Key, rawKey)
	}
	wantIV := "0x0102030405060708090a0b0c0d0e0f10"
	if string(mat.IV) != wantIV {
		t.Fatalf("iv: got %q want %q", mat.IV, wantIV)
	}
}

func TestFromURL_custom_method_without_cryptSvc_rejected(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/playlist.m3u8":
			_, _ = fmt.Fprintf(w, `#EXTM3U
#EXT-X-VERSION:3
#EXT-X-TARGETDURATION:10
#EXT-X-MEDIA-SEQUENCE:0
#EXT-X-KEY:METHOD=CUSTOM,URI="key.bin"
#EXTINF:10.0,
seg0.ts
#EXT-X-ENDLIST
`)
		case "/key.bin":
			_, _ = w.Write([]byte("raw-key-bytes!!"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	_, err := FromURL(srv.URL+"/playlist.m3u8", nil, nil)
	if err == nil {
		t.Fatal("expected error for unsupported method without crypt service")
	}
}

func TestFromURL_custom_method_with_key_hook(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "CUSTOM.star")
	starScript := `def decrypt_key(raw_key, method, uri, iv, m3u8_url):
    return {"key": raw_key[::-1], "iv": b"hooked-iv"}
`
	if err := os.WriteFile(script, []byte(starScript), 0644); err != nil {
		t.Fatal(err)
	}
	reg, err := crypt.NewRegistry(crypt.RegistryOptions{
		ScriptsDir:    dir,
		ScriptsDirAbs: dir,
		CLIScript:     script,
	})
	if err != nil {
		t.Fatal(err)
	}
	svc := crypt.NewService(reg)

	rawKey := []byte("1234567890123456")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/playlist.m3u8":
			_, _ = fmt.Fprintf(w, `#EXTM3U
#EXT-X-VERSION:3
#EXT-X-TARGETDURATION:10
#EXT-X-MEDIA-SEQUENCE:0
#EXT-X-KEY:METHOD=CUSTOM,URI="key.bin"
#EXTINF:10.0,
seg0.ts
#EXT-X-ENDLIST
`)
		case "/key.bin":
			_, _ = w.Write(rawKey)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	result, err := FromURL(srv.URL+"/playlist.m3u8", nil, svc)
	if err != nil {
		t.Fatal(err)
	}
	mat, ok := result.Keys[1]
	if !ok {
		t.Fatal("expected key at index 1")
	}
	if string(mat.Key) != "6543210987654321" {
		t.Fatalf("hooked key: got %q", mat.Key)
	}
	if string(mat.IV) != "hooked-iv" {
		t.Fatalf("hooked iv: got %q", mat.IV)
	}
}

func TestParse_allows_unknown_key_method(t *testing.T) {
	body := `#EXTM3U
#EXT-X-VERSION:3
#EXT-X-TARGETDURATION:10
#EXT-X-KEY:METHOD=SAMPLE-AES,URI="enc.key"
#EXTINF:10.0,
seg0.ts
#EXT-X-ENDLIST
`
	m3u8, err := parse(strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	key := m3u8.Keys[1]
	if key == nil {
		t.Fatal("expected key at index 1")
	}
	if key.Method != CryptMethod("SAMPLE-AES") {
		t.Fatalf("method: got %q", key.Method)
	}
}
